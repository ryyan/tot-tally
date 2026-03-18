package web

import (
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerWrapper_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) (string, error) {
		w.Write([]byte("ok"))
		return "123e4567-e89b-12d3-a456-426614174000", nil
	}

	wrapped := handlerWrapper(handler)
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options header missing or incorrect")
	}
}

func TestHandlerWrapper_Error(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) (string, error) {
		return "123e4567-e89b-12d3-a456-426614174000", errors.New("some error")
	}

	wrapped := handlerWrapper(handler)
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected status 303, got %d", rr.Code)
	}

	// Check for flash cookie
	cookies := rr.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "flash_msg" {
			found = true
			if c.Value != "error_unexpected" {
				t.Errorf("expected flash_msg error_unexpected, got %s", c.Value)
			}
		}
	}
	if !found {
		t.Error("flash_msg cookie not found")
	}
}

func TestHandlerWrapper_Panic(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) (string, error) {
		panic("test panic")
	}

	wrapped := handlerWrapper(handler)
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected status 303 after panic, got %d", rr.Code)
	}
}

func TestHandlerWrapper_Gzip(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) (string, error) {
		w.Write([]byte("gzipped data"))
		return "123e4567-e89b-12d3-a456-426614174000", nil
	}

	wrapped := handlerWrapper(handler)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Error("expected Content-Encoding: gzip")
	}

	// Verify it's actually gzipped
	gr, err := gzip.NewReader(rr.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()
	body, _ := io.ReadAll(gr)
	if string(body) != "gzipped data" {
		t.Errorf("expected 'gzipped data', got %s", string(body))
	}
}

func TestHandlerWrapper_CORS_Deny(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) (string, error) {
		return "", nil
	}

	wrapped := handlerWrapper(handler)
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Origin", "http://malicious.com")
	req.Host = "legit.com"
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", rr.Code)
	}
}

func TestHandlerWrapper_CORS_Allow(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) (string, error) {
		w.WriteHeader(http.StatusOK)
		return "", nil
	}

	wrapped := handlerWrapper(handler)
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Origin", "http://legit.com")
	req.Host = "legit.com"
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}
}

func TestIsValidID(t *testing.T) {
	valid := "123e4567-e89b-12d3-a456-426614174000"
	invalid := "not-a-uuid"

	if !isValidID(valid) {
		t.Errorf("expected %s to be valid", valid)
	}
	if isValidID(invalid) {
		t.Errorf("expected %s to be invalid", invalid)
	}
}
