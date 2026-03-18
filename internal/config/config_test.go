package config

import "testing"

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	if cfg.Port != ":5000" {
		t.Errorf("expected port :5000, got %s", cfg.Port)
	}
	if cfg.TotDirectory != "tots" {
		t.Errorf("expected tot dir tots, got %s", cfg.TotDirectory)
	}
}

func TestTallyKindMap(t *testing.T) {
	if len(TallyKindMap) == 0 {
		t.Error("TallyKindMap is empty")
	}
	if TallyKindMap[1] != "🍼1" {
		t.Errorf("expected 🍼1, got %s", TallyKindMap[1])
	}
}

func TestFlashMessages(t *testing.T) {
	if FlashMessages["tally"] == "" {
		t.Error("FlashMessages['tally'] is empty")
	}
}
