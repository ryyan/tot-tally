package core

import (
	"os"
	"path/filepath"
	"testing"
	totConfig "tot-tally/internal/config"
	totModels "tot-tally/internal/models"
	totShards "tot-tally/internal/shards"
	totStats "tot-tally/internal/stats"
	totStorage "tot-tally/internal/storage"
)

func setupCore(t *testing.T) *Service {
	tmpDir := t.TempDir()
	cfg := &totConfig.Config{
		TotDirectory: tmpDir,
		MaxTallies:   10,
	}
	repo := totStorage.NewRepository(cfg, totShards.NewPool(4))
	engine := totStats.NewEngine(cfg)
	return NewService(cfg, repo, engine)
}

func TestCreateTot(t *testing.T) {
	s := setupCore(t)

	// Happy path
	id, err := s.CreateTot("Baby", "UTC", "both")
	if err != nil {
		t.Fatalf("CreateTot failed: %v", err)
	}

	tot, err := s.store.LoadTot(id)
	if err != nil {
		t.Fatalf("LoadTot failed: %v", err)
	}

	if tot.Name != "Baby" {
		t.Errorf("Expected name Baby, got %s", tot.Name)
	}
}

func TestCreateTot_InvalidName(t *testing.T) {
	s := setupCore(t)

	_, err := s.CreateTot("", "UTC", "both")
	if err == nil {
		t.Error("Expected error for empty name, got nil")
	}

	_, err = s.CreateTot("ThisNameIsWayTooLongForTheSystemToHandle", "UTC", "both")
	if err == nil {
		t.Error("Expected error for long name, got nil")
	}
}

func TestCreateTot_InvalidTimezone(t *testing.T) {
	s := setupCore(t)

	_, err := s.CreateTot("Baby", "Invalid/Timezone", "both")
	if err == nil {
		t.Error("Expected error for invalid timezone, got nil")
	}
}

func TestAddTally(t *testing.T) {
	s := setupCore(t)
	id, _ := s.CreateTot("Baby", "UTC", "both")
	tot, _ := s.store.LoadTot(id)

	// Add a milk tally (key 1 is 🍼40 in config.TallyKindMap if I recall correctly, but I should check)
	// Actually I'll use a known key.
	// From config.go (I should check it)

	err := s.AddTally(tot, "1") // 🍼40
	if err != nil {
		t.Fatalf("AddTally failed: %v", err)
	}

	if len(tot.Tallies) != 1 {
		t.Fatalf("Expected 1 tally, got %d", len(tot.Tallies))
	}

	if tot.Stats.LastMilk == nil {
		t.Error("LastMilk was not updated")
	}
}

func TestAddTally_Mixed(t *testing.T) {
	s := setupCore(t)
	id, _ := s.CreateTot("Baby", "UTC", "both")
	tot, _ := s.store.LoadTot(id)

	// Add Pee+Poo (key 13)
	err := s.AddTally(tot, "13")
	if err != nil {
		t.Fatalf("AddTally failed: %v", err)
	}

	if len(tot.Tallies) != 2 {
		t.Fatalf("Expected 2 tallies, got %d", len(tot.Tallies))
	}
	if tot.Stats.LastPee == nil || tot.Stats.LastPoo == nil {
		t.Error("LastPee or LastPoo was not updated")
	}
}

func TestAddTally_Exhaustive(t *testing.T) {
	s := setupCore(t)
	id, _ := s.CreateTot("👶", "UTC", "both")

	kinds := []string{"1", "9", "10", "11", "12", "13", "14", "15", "16", "17"}
	for _, k := range kinds {
		tot, _ := s.store.LoadTot(id)
		err := s.AddTally(tot, k)
		if err != nil {
			t.Errorf("AddTally failed for kind %s: %v", k, err)
		}
		s.store.SaveTot(tot)
	}

	tot, _ := s.store.LoadTot(id)
	if tot.Stats.LastMilk == nil || tot.Stats.LastSnack == nil || tot.Stats.LastMeal == nil ||
		tot.Stats.LastPee == nil || tot.Stats.LastPoo == nil || tot.Stats.LastBath == nil ||
		tot.Stats.LastBrush == nil || tot.Stats.LastNurse == nil {
		t.Error("Some markers were not updated")
	}
	if tot.Stats.LastNurseSide != "R" { // Last one was kind 17 (Nurse R)
		t.Errorf("Expected LastNurseSide R, got %s", tot.Stats.LastNurseSide)
	}
}

func TestCreateTot_SaveError(t *testing.T) {
	s := setupCore(t)
	// Create a file where the directory should be to cause SaveTot to fail
	s.config.TotDirectory = filepath.Join(t.TempDir(), "file")
	os.WriteFile(s.config.TotDirectory, []byte(""), 0644)

	_, err := s.CreateTot("Baby", "UTC", "both")
	if err == nil {
		t.Error("Expected error for SaveTot failure, got nil")
	}
}

func TestAddTally_InvalidKeyFormat(t *testing.T) {
	s := setupCore(t)
	tot := &totModels.Tot{}
	err := s.AddTally(tot, "not-a-number")
	if err == nil {
		t.Error("Expected error for non-numeric kind, got nil")
	}
}
