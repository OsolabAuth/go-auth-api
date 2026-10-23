package app

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestAccessLogIncludesResponseStatus(t *testing.T) {
	for _, tt := range []struct {
		path   string
		status int
	}{
		{"/health", http.StatusOK},
		{"/empty", http.StatusOK},
		{"/missing", http.StatusNotFound},
		{"/panic", http.StatusInternalServerError},
	} {
		t.Run(tt.path, func(t *testing.T) {
			var logs bytes.Buffer
			router := Router(nil, slog.New(slog.NewJSONHandler(&logs, nil))).(*chi.Mux)
			router.Get("/panic", func(http.ResponseWriter, *http.Request) { panic("test panic") })
			router.Get("/empty", func(http.ResponseWriter, *http.Request) {})
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			request.Header.Set("X-Request-Id", "access-test")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d", response.Code, tt.status)
			}
			var entry struct {
				Status    int    `json:"status"`
				RequestID string `json:"request_id"`
			}
			if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
				t.Fatalf("decode access log: %v", err)
			}
			if entry.Status != tt.status || entry.RequestID != "access-test" {
				t.Fatalf("incorrect status or request ID in access log: %+v", entry)
			}
		})
	}
}

func TestHealth(t *testing.T) {
	router := Router(nil, slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)))
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want JSON", got)
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if len(body) != 1 || body["status"] != "ok" {
		t.Fatalf("health response = %v, want only status=ok", body)
	}
}

func TestTemplateResourcesAreNotExposed(t *testing.T) {
	router := Router(nil, slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)))
	for _, path := range []string{
		"/api/v1/tasks", "/api/v1/tasks/", "/api/v1/tasks/1",
		"/api/v1/notes", "/api/v1/notes/", "/api/v1/notes/1",
	} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
			}
		})
	}
}
