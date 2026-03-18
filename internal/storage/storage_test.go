package storage

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	totConfig "tot-tally/internal/config"
	totModels "tot-tally/internal/models"
	totShards "tot-tally/internal/shards"
)

func TestSaveAndLoadTot(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{
		TotDirectory: tmpDir,
		MaxTallies:   10,
	}
	pool := totShards.NewPool(4)
	repo := NewRepository(cfg, pool)

	tot := &totModels.Tot{
		ID:   "test-tot",
		Name: "Baby",
		Tallies: []totModels.Tally{
			{Kind: "🍼100"},
		},
	}

	err := repo.SaveTot(tot)
	if err != nil {
		t.Fatalf("SaveTot failed: %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, "test-tot.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Expected file %s to exist", expectedPath)
	}

	// Load it back
	loaded, err := repo.LoadTot("test-tot")
	if err != nil {
		t.Fatalf("LoadTot failed: %v", err)
	}

	if loaded.Name != tot.Name {
		t.Errorf("Expected name %s, got %s", tot.Name, loaded.Name)
	}
}

func TestLoadNonExistentTot(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{TotDirectory: tmpDir}
	repo := NewRepository(cfg, totShards.NewPool(1))

	_, err := repo.LoadTot("missing")
	if err == nil {
		t.Error("Expected error for missing tot, got nil")
	}
	if err.Error() != "tot does not exist" {
		t.Errorf("Expected 'tot does not exist' error, got: %v", err)
	}
}

func TestCheckAndIncrementIPLimit(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{
		LimitDirectory: tmpDir,
		MaxTotsPerIP:   2,
	}
	repo := NewRepository(cfg, totShards.NewPool(4))

	ip := "127.0.0.1"

	// 1st increment
	err := repo.CheckAndIncrementIPLimit(ip)
	if err != nil {
		t.Fatalf("1st increment failed: %v", err)
	}

	// 2nd increment
	err = repo.CheckAndIncrementIPLimit(ip)
	if err != nil {
		t.Fatalf("2nd increment failed: %v", err)
	}

	// 3rd increment (should fail)
	err = repo.CheckAndIncrementIPLimit(ip)
	if err == nil {
		t.Error("Expected limit reached error, got nil")
	}
	if err.Error() != "limit reached" {
		t.Errorf("Expected 'limit reached' error, got: %v", err)
	}
}

func TestGenerateID(t *testing.T) {
	repo := NewRepository(&totConfig.Config{}, nil)
	id, err := repo.GenerateID()
	if err != nil {
		t.Fatalf("GenerateID failed: %v", err)
	}
	if len(id) == 0 {
		t.Error("Generated ID is empty")
	}
}

func TestSaveTot_MaxTallies(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{
		TotDirectory: tmpDir,
		MaxTallies:   2,
	}
	repo := NewRepository(cfg, totShards.NewPool(1))

	tot := &totModels.Tot{
		ID: "test",
		Tallies: []totModels.Tally{
			{Kind: "1"}, {Kind: "2"}, {Kind: "3"},
		},
	}

	err := repo.SaveTot(tot)
	if err != nil {
		t.Fatalf("SaveTot failed: %v", err)
	}

	loaded, _ := repo.LoadTot("test")
	if len(loaded.Tallies) != 2 {
		t.Errorf("Expected 2 tallies, got %d", len(loaded.Tallies))
	}
}

func TestLoadTot_MaxTallies(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{
		TotDirectory: tmpDir,
		MaxTallies:   1,
	}
	repo := NewRepository(cfg, totShards.NewPool(1))

	tot := &totModels.Tot{
		ID: "test",
		Tallies: []totModels.Tally{
			{Kind: "1"}, {Kind: "2"},
		},
	}
	// Manually save it with 2 tallies
	f, _ := os.Create(filepath.Join(tmpDir, "test.json"))
	json.NewEncoder(f).Encode(tot)
	f.Close()

	loaded, _ := repo.LoadTot("test")
	if len(loaded.Tallies) != 1 {
		t.Errorf("Expected 1 tally after load pruning, got %d", len(loaded.Tallies))
	}
}

func TestCheckAndIncrementIPLimit_MalformedFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{
		LimitDirectory: tmpDir,
		MaxTotsPerIP:   10,
	}
	repo := NewRepository(cfg, totShards.NewPool(1))
	ip := "1.2.3.4"
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(ip)))
	path := filepath.Join(tmpDir, hash)

	os.WriteFile(path, []byte("not-a-number"), 0644)

	err := repo.CheckAndIncrementIPLimit(ip)
	if err != nil {
		t.Fatalf("Should handle malformed file gracefully: %v", err)
	}
}

func TestLoadTot_DecodeError(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{TotDirectory: tmpDir}
	repo := NewRepository(cfg, totShards.NewPool(1))

	path := filepath.Join(tmpDir, "bad.json")
	os.WriteFile(path, []byte("invalid json"), 0644)

	_, err := repo.LoadTot("bad")
	if err == nil {
		t.Error("Expected error for malformed JSON, got nil")
	}
}

func TestSaveTot_SwapError(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{TotDirectory: tmpDir, MaxTallies: 10}
	repo := NewRepository(cfg, totShards.NewPool(1))

	// Create a directory where the final file should be, causing os.Rename to fail
	path := filepath.Join(tmpDir, "error.json")
	os.Mkdir(path, 0755)

	err := repo.SaveTot(&totModels.Tot{ID: "error"})
	if err == nil {
		t.Error("Expected error for swap failure, got nil")
	}
}

func TestCheckAndIncrementIPLimit_WriteError(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file where the limit directory should be
	limitDir := filepath.Join(tmpDir, "limits")
	os.WriteFile(limitDir, []byte(""), 0644)

	cfg := &totConfig.Config{
		LimitDirectory: limitDir,
		MaxTotsPerIP:   10,
	}
	repo := NewRepository(cfg, totShards.NewPool(1))

	err := repo.CheckAndIncrementIPLimit("1.1.1.1")
	if err == nil {
		t.Error("Expected error when LimitDirectory is invalid, got nil")
	}
}

func TestLoadTot_OpenError(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{TotDirectory: tmpDir}
	repo := NewRepository(cfg, totShards.NewPool(1))

	path := filepath.Join(tmpDir, "error.json")
	os.Mkdir(path, 0755)

	_, err := repo.LoadTot("error")
	if err == nil {
		t.Error("Expected error when file is a directory, got nil")
	}
}

func TestLoadTot_EmptyMilkSetting(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{
		TotDirectory: tmpDir,
		MaxTallies:   10,
	}
	repo := NewRepository(cfg, totShards.NewPool(1))

	tot := &totModels.Tot{
		ID:          "empty-milk",
		MilkSetting: "",
	}
	f, _ := os.Create(filepath.Join(tmpDir, "empty-milk.json"))
	json.NewEncoder(f).Encode(tot)
	f.Close()

	loaded, err := repo.LoadTot("empty-milk")
	if err != nil {
		t.Fatalf("LoadTot failed: %v", err)
	}
	if loaded.MilkSetting != "both" {
		t.Errorf("Expected MilkSetting to default to 'both', got %s", loaded.MilkSetting)
	}
}

func TestCheckAndIncrementIPLimit_RenameError(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{
		LimitDirectory: tmpDir,
		MaxTotsPerIP:   10,
	}
	repo := NewRepository(cfg, totShards.NewPool(1))

	ip := "2.2.2.2"
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(ip)))
	finalPath := filepath.Join(tmpDir, hash)
	os.Mkdir(finalPath, 0755) // Cause rename to fail

	err := repo.CheckAndIncrementIPLimit(ip)
	if err == nil {
		t.Error("Expected error for limit rename failure, got nil")
	}
}

func TestSaveTot_CreateError(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file where the directory should be to cause an error
	cfg := &totConfig.Config{TotDirectory: filepath.Join(tmpDir, "file")}
	os.WriteFile(cfg.TotDirectory, []byte(""), 0644)

	repo := NewRepository(cfg, totShards.NewPool(1))
	err := repo.SaveTot(&totModels.Tot{ID: "test"})
	if err == nil {
		t.Error("Expected error when directory is a file, got nil")
	}
}
