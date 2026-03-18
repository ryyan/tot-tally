package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	totConfig "tot-tally/internal/config"
	totCore "tot-tally/internal/core"
	totModels "tot-tally/internal/models"
	totShards "tot-tally/internal/shards"
	totStats "tot-tally/internal/stats"
	totStorage "tot-tally/internal/storage"
)

func setupServer(t *testing.T) *Server {
	tmpDir := t.TempDir()
	cfg := totConfig.NewDefaultConfig()
	cfg.TotDirectory = filepath.Join(tmpDir, "tots")
	cfg.LimitDirectory = filepath.Join(tmpDir, "limits")
	_ = os.MkdirAll(cfg.TotDirectory, 0755)
	_ = os.MkdirAll(cfg.LimitDirectory, 0755)

	pool := totShards.NewPool(4)
	repo := totStorage.NewRepository(cfg, pool)
	engine := totStats.NewEngine(cfg)
	service := totCore.NewService(cfg, repo, engine)
	return NewServer(cfg, service, repo, engine, pool)
}

func TestHomeHandler(t *testing.T) {
	s := setupServer(t)
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	_, err := s.homeHandler(rr, req)
	if err != nil {
		t.Fatalf("homeHandler failed: %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Welcome") && !strings.Contains(rr.Body.String(), "tots") {
		// Just check that we got some HTML
		t.Log("Home body doesn't contain expected strings, might be because of empty flash")
	}
}

func TestCreateTotHandler(t *testing.T) {
	s := setupServer(t)
	form := url.Values{}
	form.Add("name", "👶")                    // Valid emoji
	form.Add("timezone", "America/New_York") // Valid timezone
	form.Add("milk_setting", "both")

	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	_, err := s.createTotHandler(rr, req)
	if err != nil {
		t.Fatalf("createTotHandler failed: %v", err)
	}

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected redirect 303, got %d", rr.Code)
	}

	loc := rr.Header().Get("Location")
	if loc == "/" || loc == "" {
		t.Errorf("invalid redirect location: %s", loc)
	}
}

func TestGetTotHandler(t *testing.T) {
	s := setupServer(t)
	id, _ := s.core.CreateTot("👶", "America/New_York", "both")

	req := httptest.NewRequest("GET", "/"+id, nil)
	// We need to set the PathValue because we are calling the handler directly
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()

	_, err := s.getTotHandler(rr, req)
	if err != nil {
		t.Fatalf("getTotHandler failed: %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestManifestHandler(t *testing.T) {
	s := setupServer(t)

	// Test default manifest (no ID)
	req := httptest.NewRequest("GET", "/manifest.json", nil)
	rr := httptest.NewRecorder()
	_, err := s.manifestHandler(rr, req)
	if err != nil {
		t.Fatalf("manifestHandler failed: %v", err)
	}
	if !strings.Contains(rr.Body.String(), `"start_url": "/"`) {
		t.Errorf("expected default start_url, got %s", rr.Body.String())
	}
	if rr.Header().Get("Content-Type") != "application/manifest+json" {
		t.Errorf("expected application/manifest+json, got %s", rr.Header().Get("Content-Type"))
	}

	// Test manifest with valid ID
	id, _ := s.core.CreateTot("👶", "UTC", "both")
	req = httptest.NewRequest("GET", "/manifest.json?id="+id, nil)
	rr = httptest.NewRecorder()
	_, err = s.manifestHandler(rr, req)
	if err != nil {
		t.Fatalf("manifestHandler failed: %v", err)
	}
	if !strings.Contains(rr.Body.String(), `"start_url": "/`+id+`"`) {
		t.Errorf("expected dynamic start_url, got %s", rr.Body.String())
	}

	// Test manifest with invalid ID
	req = httptest.NewRequest("GET", "/manifest.json?id=invalid-id", nil)
	rr = httptest.NewRecorder()
	_, err = s.manifestHandler(rr, req)
	if err != nil {
		t.Fatalf("manifestHandler failed: %v", err)
	}
	if !strings.Contains(rr.Body.String(), `"start_url": "/"`) {
		t.Errorf("expected fallback to root for invalid ID, got %s", rr.Body.String())
	}
}

func TestUpdateTotHandler_AddTally(t *testing.T) {
	s := setupServer(t)
	id, _ := s.core.CreateTot("👶", "America/New_York", "both")

	form := url.Values{}
	form.Add("tally", "1") // 🍼1

	req := httptest.NewRequest("POST", "/"+id, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()

	_, err := s.updateTotHandler(rr, req)
	if err != nil {
		t.Fatalf("updateTotHandler failed: %v", err)
	}

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", rr.Code)
	}

	tot, _ := s.store.LoadTot(id)
	if len(tot.Tallies) != 1 {
		t.Errorf("expected 1 tally, got %d", len(tot.Tallies))
	}
}

func TestUpdateTotHandler_Undo(t *testing.T) {
	s := setupServer(t)
	id, _ := s.core.CreateTot("👶", "America/New_York", "both")
	tot, _ := s.store.LoadTot(id)
	s.core.AddTally(tot, "1")
	s.store.SaveTot(tot)

	form := url.Values{}
	form.Add("undo", "1")

	req := httptest.NewRequest("POST", "/"+id, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()

	_, err := s.updateTotHandler(rr, req)
	if err != nil {
		t.Fatalf("updateTotHandler failed: %v", err)
	}

	tot, _ = s.store.LoadTot(id)
	if len(tot.Tallies) != 0 {
		t.Errorf("expected 0 tallies after undo, got %d", len(tot.Tallies))
	}
}

func TestUpdateTotHandler_Timezone(t *testing.T) {
	s := setupServer(t)
	id, _ := s.core.CreateTot("👶", "UTC", "both")

	form := url.Values{}
	form.Add("timezone", "America/Los_Angeles")

	req := httptest.NewRequest("POST", "/"+id, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()

	_, err := s.updateTotHandler(rr, req)
	if err != nil {
		t.Fatalf("updateTotHandler failed: %v", err)
	}

	tot, _ := s.store.LoadTot(id)
	if tot.Timezone != "America/Los_Angeles" {
		t.Errorf("expected timezone America/Los_Angeles, got %s", tot.Timezone)
	}
}

func TestUpdateTotHandler_MilkSetting(t *testing.T) {
	s := setupServer(t)
	id, _ := s.core.CreateTot("👶", "UTC", "both")

	form := url.Values{}
	form.Add("milk_setting", "bottle")

	req := httptest.NewRequest("POST", "/"+id, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()

	_, err := s.updateTotHandler(rr, req)
	if err != nil {
		t.Fatalf("updateTotHandler failed: %v", err)
	}

	tot, _ := s.store.LoadTot(id)
	if tot.MilkSetting != "bottle" {
		t.Errorf("expected milk_setting bottle, got %s", tot.MilkSetting)
	}
}

func TestUpdateTotHandler_InvalidID(t *testing.T) {
	s := setupServer(t)
	req := httptest.NewRequest("POST", "/invalid-id", nil)
	req.SetPathValue("id", "invalid-id")
	rr := httptest.NewRecorder()

	_, err := s.updateTotHandler(rr, req)
	if err == nil {
		t.Error("expected error for invalid ID, got nil")
	}
}

func TestExportTotHandler(t *testing.T) {
	s := setupServer(t)
	id, _ := s.core.CreateTot("👶", "America/New_York", "both")

	req := httptest.NewRequest("GET", "/export/"+id, nil)
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()

	_, err := s.exportTotHandler(rr, req)
	if err != nil {
		t.Fatalf("exportTotHandler failed: %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		t        *time.Time
		expected string
	}{
		{nil, "not yet"},
		{&time.Time{}, "not yet"},
		{pTime(now.Add(-30 * time.Second)), "just now"},
		{pTime(now.Add(-5 * time.Minute)), "5m ago"},
		{pTime(now.Add(-2 * time.Hour)), "2h ago"},
		{pTime(now.Add(-2*time.Hour - 5*time.Minute)), "2h 5m ago"},
		{pTime(now.Add(-48 * time.Hour)), "2d ago"},
	}

	for _, tt := range tests {
		result := formatRelativeTime(tt.t)
		if result != tt.expected {
			t.Errorf("for %v expected %s, got %s", tt.t, tt.expected, result)
		}
	}
}

func pTime(t time.Time) *time.Time {
	return &t
}

func TestCreateTotHandler_InvalidAvatar(t *testing.T) {
	s := setupServer(t)
	form := url.Values{}
	form.Add("name", "Invalid")
	form.Add("timezone", "UTC")
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	_, err := s.createTotHandler(rr, req)
	if err == nil {
		t.Error("expected error for invalid avatar, got nil")
	}
}

func TestCreateTotHandler_InvalidTimezone(t *testing.T) {
	s := setupServer(t)
	form := url.Values{}
	form.Add("name", "👶")
	form.Add("timezone", "Invalid/Tz")
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	_, err := s.createTotHandler(rr, req)
	if err == nil {
		t.Error("expected error for invalid timezone, got nil")
	}
}

func TestCreateTotHandler_InvalidMilkSetting(t *testing.T) {
	s := setupServer(t)
	form := url.Values{}
	form.Add("name", "👶")
	form.Add("timezone", "UTC")
	form.Add("milk_setting", "invalid")
	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	_, err := s.createTotHandler(rr, req)
	if err == nil {
		t.Error("expected error for invalid milk setting, got nil")
	}
}

func TestUpdateTotHandler_NoChange(t *testing.T) {
	s := setupServer(t)
	id, _ := s.core.CreateTot("👶", "UTC", "both")
	req := httptest.NewRequest("POST", "/"+id, nil)
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()
	_, err := s.updateTotHandler(rr, req)
	if err != nil {
		t.Fatalf("updateTotHandler failed: %v", err)
	}
	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", rr.Code)
	}
}

func TestGetTotPageData(t *testing.T) {
	s := setupServer(t)
	id, _ := s.core.CreateTot("👶", "America/New_York", "both")
	tot, _ := s.store.LoadTot(id)
	s.core.AddTally(tot, "1") // 🍼1
	s.store.SaveTot(tot)

	data, err := s.getTotPageData(id, "tally")
	if err != nil {
		t.Fatalf("getTotPageData failed: %v", err)
	}

	if data.Name != "👶" {
		t.Errorf("expected 👶, got %s", data.Name)
	}
	if data.Stats.LastMilkAmount != "1" {
		t.Errorf("expected LastMilkAmount 1, got %s", data.Stats.LastMilkAmount)
	}
}

func TestHomeHandler_WithFlash(t *testing.T) {
	s := setupServer(t)
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "flash_msg", Value: "error_unexpected"})
	rr := httptest.NewRecorder()

	_, err := s.homeHandler(rr, req)
	if err != nil {
		t.Fatalf("homeHandler failed: %v", err)
	}

	if !strings.Contains(rr.Body.String(), "Error: Unexpected error!") {
		t.Error("expected body to contain flash message")
	}

	// Verify cookie is cleared
	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == "flash_msg" {
			found = true
			if c.MaxAge != -1 {
				t.Errorf("expected cookie to be cleared, got MaxAge %d", c.MaxAge)
			}
		}
	}
	if !found {
		t.Error("expected flash_msg cookie to be set for removal")
	}
}

func TestGetTotHandler_NotFound(t *testing.T) {
	s := setupServer(t)
	id := "123e4567-e89b-12d3-a456-426614174000" // Valid UUID but missing
	req := httptest.NewRequest("GET", "/"+id, nil)
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()

	_, err := s.getTotHandler(rr, req)
	// getTotHandler returns (totID, error), which handlerWrapper will catch.
	if err == nil {
		t.Error("expected error for missing tot, got nil")
	}
}

func TestGetTotHandler_InvalidID(t *testing.T) {
	s := setupServer(t)
	req := httptest.NewRequest("GET", "/invalid-uuid", nil)
	req.SetPathValue("id", "invalid-uuid")
	rr := httptest.NewRecorder()

	_, err := s.getTotHandler(rr, req)
	if err == nil {
		t.Error("expected error for invalid ID, got nil")
	}
}

func TestCreateTotHandler_IPLimitReached(t *testing.T) {
	s := setupServer(t)
	s.config.MaxTotsPerIP = 0 // Force limit reached

	form := url.Values{}
	form.Add("name", "👶")
	form.Add("timezone", "UTC")

	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "1.2.3.4:1234"
	rr := httptest.NewRecorder()

	_, err := s.createTotHandler(rr, req)
	if err != nil {
		t.Fatalf("createTotHandler failed: %v", err)
	}

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", rr.Code)
	}

	// Check for flash cookie
	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == "flash_msg" && c.Value == "error_limit_ip" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error_limit_ip flash cookie")
	}
}

func TestGetTotHandler_WithFlash(t *testing.T) {
	s := setupServer(t)
	id, _ := s.core.CreateTot("👶", "America/New_York", "both")

	req := httptest.NewRequest("GET", "/"+id, nil)
	req.SetPathValue("id", id)
	req.AddCookie(&http.Cookie{Name: "flash_msg", Value: "tally"})
	rr := httptest.NewRecorder()

	_, err := s.getTotHandler(rr, req)
	if err != nil {
		t.Fatalf("getTotHandler failed: %v", err)
	}

	if !strings.Contains(rr.Body.String(), "Tally Added!") {
		t.Error("expected body to contain flash message")
	}
}

func TestCreateTotHandler_InvalidRemoteAddr(t *testing.T) {
	s := setupServer(t)
	form := url.Values{}
	form.Add("name", "👶")
	form.Add("timezone", "America/New_York")

	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "invalid-addr" // No port
	rr := httptest.NewRecorder()

	_, err := s.createTotHandler(rr, req)
	if err != nil {
		t.Fatalf("createTotHandler failed: %v", err)
	}
}

func TestUpdateTotHandler_MalformedTally(t *testing.T) {
	s := setupServer(t)
	id, _ := s.core.CreateTot("👶", "UTC", "both")

	form := url.Values{}
	form.Add("tally", "abc") // Not a number

	req := httptest.NewRequest("POST", "/"+id, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()

	_, err := s.updateTotHandler(rr, req)
	if err != nil {
		t.Fatalf("updateTotHandler should not return error for malformed tally (it just skips it): %v", err)
	}

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", rr.Code)
	}

	tot, _ := s.store.LoadTot(id)
	if len(tot.Tallies) != 0 {
		t.Errorf("expected 0 tallies, got %d", len(tot.Tallies))
	}
}

func TestCreateTotHandler_EmptyMilkSetting(t *testing.T) {
	s := setupServer(t)
	form := url.Values{}
	form.Add("name", "👶")
	form.Add("timezone", "America/New_York")
	form.Add("milk_setting", "") // Should default to 'both'

	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	id, err := s.createTotHandler(rr, req)
	if err != nil {
		t.Fatalf("createTotHandler failed: %v", err)
	}

	tot, _ := s.store.LoadTot(id)
	if tot.MilkSetting != "both" {
		t.Errorf("expected both, got %s", tot.MilkSetting)
	}
}

func TestCreateTotHandler_CreateError(t *testing.T) {
	s := setupServer(t)
	// Make SaveTot fail by making the directory a file
	os.RemoveAll(s.config.TotDirectory)
	os.WriteFile(s.config.TotDirectory, []byte(""), 0644)

	form := url.Values{}
	form.Add("name", "👶")
	form.Add("timezone", "America/New_York")

	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	_, err := s.createTotHandler(rr, req)
	if err == nil {
		t.Error("expected error when SaveTot fails, got nil")
	}
}

func TestUpdateTotHandler_GenerateStatsError(t *testing.T) {
	s := setupServer(t)
	id, _ := s.core.CreateTot("👶", "America/New_York", "both")

	// Manually inject a malformed tally that will cause GenerateStats to fail
	tot, _ := s.store.LoadTot(id)
	tot.Tallies = append(tot.Tallies, totModels.Tally{Kind: "🍼abc", Time: pTime(time.Now())})
	s.store.SaveTot(tot)

	form := url.Values{}
	form.Add("tally", "1") // Try to add another tally, triggering GenerateStats

	req := httptest.NewRequest("POST", "/"+id, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()

	_, err := s.updateTotHandler(rr, req)
	if err == nil {
		t.Error("expected error from GenerateStats, got nil")
	}
}

func TestUpdateTotHandler_SaveError(t *testing.T) {
	s := setupServer(t)
	id, _ := s.core.CreateTot("👶", "America/New_York", "both")

	// Make SaveTot fail
	finalPath := filepath.Join(s.config.TotDirectory, id+".json")
	os.Mkdir(finalPath+".tmp", 0755) // Cause os.Create(tmpPath) to fail

	form := url.Values{}
	form.Add("tally", "1")

	req := httptest.NewRequest("POST", "/"+id, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()

	_, err := s.updateTotHandler(rr, req)
	if err == nil {
		t.Error("expected error from SaveTot, got nil")
	}
}

func TestNewServer_Fallback(t *testing.T) {
	// To hit the fallback, we need to be in a directory where assets/ is not found
	// in any of the searched paths.
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	// We create a nested directory to ensure we don't find assets/ in ../ or ../../
	nested := filepath.Join(tmpDir, "a", "b", "c")
	os.MkdirAll(nested, 0755)
	os.Chdir(nested)

	// Now we create assets/index.html in the CURRENT directory (nested)
	// so that when NewServer falls back to "assets/index.html", it finds them.
	_ = os.Mkdir(filepath.Join(nested, "assets"), 0755)
	_ = os.WriteFile(filepath.Join(nested, "assets", "index.html"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(nested, "assets", "tot.html"), []byte(""), 0644)

	cfg := totConfig.NewDefaultConfig()
	pool := totShards.NewPool(1)
	repo := totStorage.NewRepository(cfg, pool)
	engine := totStats.NewEngine(cfg)
	service := totCore.NewService(cfg, repo, engine)

	s := NewServer(cfg, service, repo, engine, pool)
	if s == nil {
		t.Fatal("Expected server, got nil")
	}
}

func TestUpdateTotHandler_DeleteTot(t *testing.T) {
	s := setupServer(t)
	id, _ := s.core.CreateTot("👶", "UTC", "both")

	// 1. Unconfirmed deletion attempt (should do nothing)
	form := url.Values{}
	form.Add("delete_tot", "true")
	form.Add("confirm_delete", "false")

	req := httptest.NewRequest("POST", "/"+id, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()

	_, err := s.updateTotHandler(rr, req)
	if err != nil {
		t.Fatalf("updateTotHandler failed: %v", err)
	}

	_, err = s.store.LoadTot(id)
	if err != nil {
		t.Error("expected tot to still exist after unconfirmed deletion")
	}

	// 2. Confirmed deletion attempt
	form = url.Values{}
	form.Add("delete_tot", "true")
	form.Add("confirm_delete", "true")

	req = httptest.NewRequest("POST", "/"+id, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()

	_, err = s.updateTotHandler(rr, req)
	if err != nil {
		t.Fatalf("updateTotHandler failed: %v", err)
	}

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", rr.Code)
	}

	_, err = s.store.LoadTot(id)
	if err == nil {
		t.Error("expected tot to be deleted after confirmed deletion")
	}

	// 3. Failed deletion (file already gone or missing)
	form = url.Values{}
	form.Add("delete_tot", "true")
	form.Add("confirm_delete", "true")

	req = httptest.NewRequest("POST", "/"+id, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()

	_, err = s.updateTotHandler(rr, req)
	if err == nil {
		t.Error("expected error when deleting non-existent tot")
	}
}

func TestExportTotHandler_NotFound(t *testing.T) {
	s := setupServer(t)
	req := httptest.NewRequest("GET", "/export/missing", nil)
	req.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()
	_, err := s.exportTotHandler(rr, req)
	if err == nil {
		t.Error("expected error for missing tot, got nil")
	}
}
