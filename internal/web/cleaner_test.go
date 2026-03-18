package web

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
	totConfig "tot-tally/internal/config"
	totModels "tot-tally/internal/models"
	totShards "tot-tally/internal/shards"
	totStorage "tot-tally/internal/storage"
)

func TestCleaner_CleanFolder(t *testing.T) {
	tmpDir := t.TempDir()
	totDir := filepath.Join(tmpDir, "tots")
	limitDir := filepath.Join(tmpDir, "limits")
	_ = os.Mkdir(totDir, 0755)
	_ = os.Mkdir(limitDir, 0755)

	cfg := &totConfig.Config{
		TotDirectory:   totDir,
		LimitDirectory: limitDir,
		CleanupAge:     24 * time.Hour,
	}
	store := totStorage.NewRepository(cfg, totShards.NewPool(1))
	cleaner := NewCleaner(cfg, store)

	// Create an old tot (manually to avoid UpdatedAt update in SaveTot)
	oldTot := &totModels.Tot{
		ID:        "old-tot",
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-48 * time.Hour),
	}
	f, _ := os.Create(filepath.Join(totDir, "old-tot.json"))
	json.NewEncoder(f).Encode(oldTot)
	f.Close()

	// Create a new tot
	newTot := &totModels.Tot{
		ID:        "new-tot",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = store.SaveTot(newTot)

	// Create an old limit
	oldLimitPath := filepath.Join(limitDir, "old-limit")
	_ = os.WriteFile(oldLimitPath, []byte("1\n"+strconv.FormatInt(time.Now().Add(-48*time.Hour).UnixMilli(), 10)), 0644)

	// Create a new limit
	newLimitPath := filepath.Join(limitDir, "new-limit")
	_ = os.WriteFile(newLimitPath, []byte("1\n"+strconv.FormatInt(time.Now().UnixMilli(), 10)), 0644)

	cleaner.cleanFolder(totDir, cfg.CleanupAge, true)
	cleaner.cleanFolder(limitDir, cfg.CleanupAge, false)

	if _, err := os.Stat(filepath.Join(totDir, "old-tot.json")); !os.IsNotExist(err) {
		t.Error("old tot should have been deleted")
	}
	if _, err := os.Stat(filepath.Join(totDir, "new-tot.json")); os.IsNotExist(err) {
		t.Error("new tot should NOT have been deleted")
	}
	if _, err := os.Stat(oldLimitPath); !os.IsNotExist(err) {
		t.Error("old limit should have been deleted")
	}
	if _, err := os.Stat(newLimitPath); os.IsNotExist(err) {
		t.Error("new limit should NOT have been deleted")
	}
}

func TestCleaner_CleanFolder_UnreadableTot(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{TotDirectory: tmpDir, CleanupAge: 24 * time.Hour}
	store := totStorage.NewRepository(cfg, totShards.NewPool(1))
	cleaner := NewCleaner(cfg, store)

	path := filepath.Join(tmpDir, "unreadable.json")
	_ = os.WriteFile(path, []byte("invalid json"), 0644)

	cleaner.cleanFolder(tmpDir, cfg.CleanupAge, true)

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("unreadable tot should have been deleted")
	}
}

func TestCleaner_CleanFolder_MalformedLimit(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{LimitDirectory: tmpDir, CleanupAge: 24 * time.Hour}
	store := totStorage.NewRepository(cfg, totShards.NewPool(1))
	cleaner := NewCleaner(cfg, store)

	// Short file (only 1 line)
	path1 := filepath.Join(tmpDir, "short")
	_ = os.WriteFile(path1, []byte("1"), 0644)

	// Malformed timestamp
	path2 := filepath.Join(tmpDir, "bad-ts")
	_ = os.WriteFile(path2, []byte("1\nnot-a-timestamp"), 0644)

	cleaner.cleanFolder(tmpDir, cfg.CleanupAge, false)

	if _, err := os.Stat(path1); !os.IsNotExist(err) {
		t.Error("short limit should have been deleted")
	}
	if _, err := os.Stat(path2); !os.IsNotExist(err) {
		t.Error("bad timestamp limit should have been deleted")
	}
}

func TestCleaner_CleanFolder_ReadDirError(t *testing.T) {
	cfg := &totConfig.Config{TotDirectory: "/nonexistent", CleanupAge: 24 * time.Hour}
	store := totStorage.NewRepository(cfg, totShards.NewPool(1))
	cleaner := NewCleaner(cfg, store)

	// This should just return without panicking and log an error
	cleaner.cleanFolder("/nonexistent", cfg.CleanupAge, true)
}

func TestCleaner_StartBackgroundCleaner(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{
		TotDirectory:   tmpDir,
		LimitDirectory: tmpDir,
		CleanupAge:     24 * time.Hour,
	}
	store := totStorage.NewRepository(cfg, totShards.NewPool(1))
	cleaner := NewCleaner(cfg, store)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Stop it immediately

	cleaner.StartBackgroundCleaner(ctx)
	// Give it a tiny bit of time to start and stop
	time.Sleep(50 * time.Millisecond)
}

func TestCleaner_CleanFolder_LastActiveZero(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{TotDirectory: tmpDir, CleanupAge: 24 * time.Hour}
	store := totStorage.NewRepository(cfg, totShards.NewPool(1))
	cleaner := NewCleaner(cfg, store)

	// Tot with no UpdatedAt, and CreatedAt is old
	tot := &totModels.Tot{
		ID:        "no-update",
		CreatedAt: time.Now().Add(-48 * time.Hour),
		// UpdatedAt is Zero
	}
	f, _ := os.Create(filepath.Join(tmpDir, "no-update.json"))
	json.NewEncoder(f).Encode(tot)
	f.Close()

	cleaner.cleanFolder(tmpDir, cfg.CleanupAge, true)

	if _, err := os.Stat(filepath.Join(tmpDir, "no-update.json")); !os.IsNotExist(err) {
		t.Error("tot with zero UpdatedAt and old CreatedAt should have been deleted")
	}
}

func TestCleaner_CleanFolder_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{TotDirectory: tmpDir}
	store := totStorage.NewRepository(cfg, totShards.NewPool(1))
	cleaner := NewCleaner(cfg, store)

	cleaner.cleanFolder(tmpDir, 24*time.Hour, true)
}

func TestCleaner_CleanFolder_WithDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{TotDirectory: tmpDir}
	store := totStorage.NewRepository(cfg, totShards.NewPool(1))
	cleaner := NewCleaner(cfg, store)

	_ = os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	cleaner.cleanFolder(tmpDir, 24*time.Hour, true)
}

func TestCleaner_CleanFolder_LimitReadError(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{LimitDirectory: tmpDir, CleanupAge: 24 * time.Hour}
	store := totStorage.NewRepository(cfg, totShards.NewPool(1))
	cleaner := NewCleaner(cfg, store)

	path := filepath.Join(tmpDir, "noread")
	_ = os.WriteFile(path, []byte(""), 0000)

	cleaner.cleanFolder(tmpDir, cfg.CleanupAge, false)
}
