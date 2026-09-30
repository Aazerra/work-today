package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aazerra/work-today/internal/model"
)

const (
	legacyFileName = ".work_today.json"
	xdgAppName     = "work-today"
	xdgFileName    = "tasks.json"
)

var mu sync.Mutex

func xdgDataDir(home string) string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" && filepath.IsAbs(xdg) {
		return xdg
	}
	if home != "" {
		return filepath.Join(home, ".local", "share")
	}
	return ""
}

// Path returns the path to the tasks JSON file.
// It prioritizes:
//  1. WORK_TODAY_PATH environment variable (if non-empty).
//  2. Existing XDG data file ($XDG_DATA_HOME/work-today/tasks.json or ~/.local/share/work-today/tasks.json).
//  3. Existing legacy fallback file ($HOME/.work_today.json).
//  4. Default XDG data path ($XDG_DATA_HOME/work-today/tasks.json or ~/.local/share/work-today/tasks.json).
//  5. Relative fallback file (./.work_today.json) if home directory is unavailable.
func Path() string {
	if custom := os.Getenv("WORK_TODAY_PATH"); custom != "" {
		return custom
	}

	home, err := os.UserHomeDir()
	var legacyPath string
	if err == nil && home != "" {
		legacyPath = filepath.Join(home, legacyFileName)
	}

	var xdgFile string
	if dataDir := xdgDataDir(home); dataDir != "" {
		xdgDir := filepath.Join(dataDir, xdgAppName)
		xdgFile = filepath.Join(xdgDir, xdgFileName)

		if _, err := os.Stat(xdgFile); err == nil {
			return xdgFile
		}
		xdgAlt := filepath.Join(xdgDir, legacyFileName)
		if _, err := os.Stat(xdgAlt); err == nil {
			return xdgAlt
		}
	}

	if legacyPath != "" {
		if _, err := os.Stat(legacyPath); err == nil {
			return legacyPath
		}
	}

	if xdgFile != "" {
		return xdgFile
	}

	if legacyPath != "" {
		return legacyPath
	}

	return filepath.Join(".", legacyFileName)
}

func today() string {
	return time.Now().Format("2006-01-02")
}

// Load reads the tasks document from Path(). A missing file yields an empty document for today.
// If the stored date is not today, completed tasks are dropped and unfinished ones
// are carried forward so the day starts clean without losing open work.
func Load() (*model.Document, error) {
	mu.Lock()
	defer mu.Unlock()

	path := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &model.Document{Date: today(), Tasks: []model.Task{}}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var doc model.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Tasks == nil {
		doc.Tasks = []model.Task{}
	}

	current := today()
	if doc.Date != current {
		carried := make([]model.Task, 0, len(doc.Tasks))
		completed := make([]model.Task, 0, len(doc.Tasks))
		for _, t := range doc.Tasks {
			if t.Status == model.StatusDone {
				completed = append(completed, t)
			} else {
				carried = append(carried, t)
			}
		}
		if len(completed) > 0 {
			_ = ArchiveTasksUnlocked(doc.Date, completed)
		}
		doc.Date = current
		doc.Tasks = carried
		if err := saveUnlocked(&doc); err != nil {
			return nil, err
		}
	}

	return &doc, nil
}

// Save writes the document atomically to Path().
func Save(doc *model.Document) error {
	mu.Lock()
	defer mu.Unlock()
	return saveUnlocked(doc)
}

func saveUnlocked(doc *model.Document) error {
	if doc == nil {
		return errors.New("nil document")
	}
	if doc.Date == "" {
		doc.Date = today()
	}
	if doc.Tasks == nil {
		doc.Tasks = []model.Task{}
	}

	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	payload = append(payload, '\n')

	path := Path()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".work_today.*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()

	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp: %w", err)
	}
	cleanup = false
	return nil
}

// ArchiveDir returns the path to the daily archive directory.
func ArchiveDir() string {
	path := Path()
	dir := filepath.Dir(path)
	home, err := os.UserHomeDir()
	if err == nil && dir == home {
		return filepath.Join(home, ".work_today_archive")
	}
	return filepath.Join(dir, "archive")
}

// ArchiveTasksUnlocked writes completed tasks for a specific date to the archive without locking.
func ArchiveTasksUnlocked(date string, tasks []model.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	if date == "" {
		date = today()
	}

	archiveDir := ArchiveDir()
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("create archive dir: %w", err)
	}

	archivePath := filepath.Join(archiveDir, date+".json")
	existing := &model.Document{Date: date, Tasks: []model.Task{}}

	if data, err := os.ReadFile(archivePath); err == nil {
		_ = json.Unmarshal(data, existing)
	}

	seenIDs := make(map[string]bool)
	for _, t := range existing.Tasks {
		seenIDs[t.ID] = true
	}

	for _, t := range tasks {
		if !seenIDs[t.ID] {
			existing.Tasks = append(existing.Tasks, t)
			seenIDs[t.ID] = true
		}
	}

	payload, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal archive: %w", err)
	}
	payload = append(payload, '\n')

	tmp, err := os.CreateTemp(archiveDir, ".archive.*.tmp")
	if err != nil {
		return fmt.Errorf("create archive temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write archive temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync archive temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close archive temp: %w", err)
	}
	return os.Rename(tmpName, archivePath)
}

// ArchiveTasks archives completed tasks for a date (acquires storage mutex).
func ArchiveTasks(date string, tasks []model.Task) error {
	mu.Lock()
	defer mu.Unlock()
	return ArchiveTasksUnlocked(date, tasks)
}

// ClearCompleted removes all completed tasks from today's active list,
// immediately archives them under today's date, and saves the updated active document.
// Returns the list of cleared tasks.
func ClearCompleted() ([]model.Task, error) {
	mu.Lock()
	defer mu.Unlock()

	path := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var doc model.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	remaining := make([]model.Task, 0, len(doc.Tasks))
	completed := make([]model.Task, 0, len(doc.Tasks))
	for _, t := range doc.Tasks {
		if t.Status == model.StatusDone {
			completed = append(completed, t)
		} else {
			remaining = append(remaining, t)
		}
	}

	if len(completed) == 0 {
		return nil, nil
	}

	if err := ArchiveTasksUnlocked(doc.Date, completed); err != nil {
		return nil, err
	}

	doc.Tasks = remaining
	if err := saveUnlocked(&doc); err != nil {
		return nil, err
	}

	return completed, nil
}

// LoadArchives returns archived documents sorted by date descending (newest first).
// If limit <= 0, all archives are returned.
func LoadArchives(limit int) ([]model.Document, error) {
	mu.Lock()
	defer mu.Unlock()

	archiveDir := ArchiveDir()
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read archive dir: %w", err)
	}

	var dates []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") && !strings.HasPrefix(entry.Name(), ".") {
			dates = append(dates, strings.TrimSuffix(entry.Name(), ".json"))
		}
	}

	sort.Slice(dates, func(i, j int) bool {
		return dates[i] > dates[j]
	})

	if limit > 0 && len(dates) > limit {
		dates = dates[:limit]
	}

	var docs []model.Document
	for _, date := range dates {
		filePath := filepath.Join(archiveDir, date+".json")
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		var doc model.Document
		if err := json.Unmarshal(data, &doc); err == nil {
			docs = append(docs, doc)
		}
	}

	return docs, nil
}

// CalculateStreak computes the consecutive day completion streak.
func CalculateStreak() int {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	todayStr := now.Format("2006-01-02")

	activeDates := make(map[string]int)
	archiveDir := ArchiveDir()
	if entries, err := os.ReadDir(archiveDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") && !strings.HasPrefix(e.Name(), ".") {
				date := strings.TrimSuffix(e.Name(), ".json")
				if data, err := os.ReadFile(filepath.Join(archiveDir, e.Name())); err == nil {
					var doc model.Document
					if err := json.Unmarshal(data, &doc); err == nil && len(doc.Tasks) > 0 {
						activeDates[date] += len(doc.Tasks)
					}
				}
			}
		}
	}

	if data, err := os.ReadFile(Path()); err == nil {
		var doc model.Document
		if err := json.Unmarshal(data, &doc); err == nil {
			for _, t := range doc.Tasks {
				if t.Status == model.StatusDone {
					activeDates[doc.Date]++
				}
			}
		}
	}

	streak := 0
	checkDate := now

	if activeDates[todayStr] > 0 {
		streak++
		checkDate = checkDate.AddDate(0, 0, -1)
	} else {
		yesterdayStr := now.AddDate(0, 0, -1).Format("2006-01-02")
		if activeDates[yesterdayStr] == 0 {
			return 0
		}
		checkDate = checkDate.AddDate(0, 0, -1)
	}

	for {
		dateStr := checkDate.Format("2006-01-02")
		if activeDates[dateStr] > 0 {
			streak++
			checkDate = checkDate.AddDate(0, 0, -1)
		} else {
			break
		}
	}

	return streak
}
