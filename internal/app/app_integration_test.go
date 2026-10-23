package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/OsolabAuth/go-auth-api/internal/database"
)

func TestReadinessWithPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, databaseURL, 1)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(pool.Close)

	router := Router(pool, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	check := func(wantStatus int, wantBody map[string]string) {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/ready", nil)
		request.Header.Set("X-Request-ID", "readiness-test")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != wantStatus {
			t.Fatalf("status = %d, want %d; body = %s", response.Code, wantStatus, response.Body.String())
		}
		if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want JSON", got)
		}
		var body map[string]string
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode readiness response: %v", err)
		}
		if len(body) != len(wantBody) {
			t.Fatalf("readiness response = %v, want %v", body, wantBody)
		}
		for key, want := range wantBody {
			if got := body[key]; got != want {
				t.Fatalf("readiness response %s = %q, want %q", key, got, want)
			}
		}
	}

	check(http.StatusOK, map[string]string{"status": "ready"})

	// Closing only this test's pool simulates unavailable DB access without
	// stopping PostgreSQL or modifying any database schema or data.
	pool.Close()
	check(http.StatusServiceUnavailable, map[string]string{
		"error":      "database unavailable",
		"request_id": "readiness-test",
	})
}
