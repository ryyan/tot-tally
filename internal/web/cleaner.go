// cleaner.go includes a background maintenance task for deleting old data files.
package web

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	totConfig "tot-tally/internal/config"
	totStorage "tot-tally/internal/storage"
)

// Cleaner manages the background pruning of old records.
type Cleaner struct {
	config *totConfig.Config
	store  *totStorage.Repository
}

// NewCleaner initializes the maintenance service.
func NewCleaner(cfg *totConfig.Config, store *totStorage.Repository) *Cleaner {
	return &Cleaner{config: cfg, store: store}
}

// StartBackgroundCleaner initiates a daily goroutine that prunes old data.
func (c *Cleaner) StartBackgroundCleaner(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			slog.Info("background cleanup starting")
			c.cleanFolder(c.config.TotDirectory, c.config.CleanupAge, true)
			c.cleanFolder(c.config.LimitDirectory, c.config.CleanupAge, false)

			select {
			case <-ticker.C:
				continue
			case <-ctx.Done():
				slog.Info("background cleaner stopping")
				return
			}
		}
	}()
}

func (c *Cleaner) cleanFolder(dir string, maxAge time.Duration, isTot bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		slog.Error("cleanup directory read failed", "dir", dir, "err", err)
		return
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())

		if isTot {
			tot, err := c.store.LoadTot(strings.TrimSuffix(entry.Name(), ".json"))
			if err != nil {
				slog.Warn("cleanup removing unreadable tot", "path", path)
				os.Remove(path)
				continue
			}
			lastActive := tot.UpdatedAt
			if lastActive.IsZero() {
				lastActive = tot.CreatedAt
			}
			if now.Sub(lastActive) > maxAge {
				slog.Info("cleanup removing expired tot", "id", tot.ID)
				os.Remove(path)
			}
		} else {
			data, err := os.ReadFile(path)
			if err != nil {
				os.Remove(path)
				continue
			}
			lines := strings.Split(string(data), "\n")
			if len(lines) >= 2 {
				ts, err := strconv.ParseInt(lines[1], 10, 64)
				if err != nil || now.Sub(time.UnixMilli(ts)) > maxAge {
					slog.Info("cleanup removing expired limit", "file", entry.Name())
					os.Remove(path)
				}
			} else {
				os.Remove(path)
			}
		}
	}
}
