package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alireza/work-today/internal/model"
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
		for _, t := range doc.Tasks {
			if t.Status != model.StatusDone {
				carried = append(carried, t)
			}
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
