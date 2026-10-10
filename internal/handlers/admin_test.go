package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bonbonn1912/bonbonn-game-tracker/internal/config"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/faceit"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/models"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/store"
)

func setupAdminTestServer(t *testing.T) (*Server, *store.MemoryStore) {
	memStore := store.NewMemoryStore()
	cfg := &config.Config{
		Port:          "12000",
		Env:           "test",
		AdminUsername: "admin",
		AdminPassword: "supersecretadminpassword",
		AdminHost:     "admin.faceitlobby.com",
		FaceitBaseURL: "http://fake-faceit.local",
	}

	fc := faceit.NewClient(cfg.FaceitBaseURL, "fake-api-key", nil)
	srv, err := NewServer(cfg, memStore, fc)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	return srv, memStore
}

func TestHealthCheck(t *testing.T) {
	srv, memStore := setupAdminTestServer(t)

	// Healthy
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}

	// Database failure -> 503
	memStore.SetPingError(errors.New("db disconnected"))
	req = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 Service Unavailable, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"unreachable"`) {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func TestAdminAuthAndCSRF(t *testing.T) {
	srv, _ := setupAdminTestServer(t)

	t.Run("Unauthorized without basic auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api/status", nil)
		req.Host = "admin.faceitlobby.com"
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
		if !strings.Contains(rec.Header().Get("WWW-Authenticate"), "Basic realm=") {
			t.Errorf("missing WWW-Authenticate header: %v", rec.Header())
		}
	})

	t.Run("Unauthorized with invalid credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api/status", nil)
		req.Host = "admin.faceitlobby.com"
		req.SetBasicAuth("admin", "wrongpassword")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Authorized with valid credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api/status", nil)
		req.Host = "admin.faceitlobby.com"
		req.SetBasicAuth("admin", "supersecretadminpassword")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		// Confirm NO wildcard CORS header on admin endpoints!
		if rec.Header().Get("Access-Control-Allow-Origin") == "*" {
			t.Errorf("admin endpoint must not expose wildcard Access-Control-Allow-Origin: *")
		}
	})

	t.Run("CSRF origin check rejects foreign origin on mutation", func(t *testing.T) {
		body := `{"id":"11111111-1111-1111-1111-111111111111","name":"Hacker","url":"http://evil.com","enabled":true}`
		req := httptest.NewRequest(http.MethodPost, "/admin/api/streamers", strings.NewReader(body))
		req.Host = "admin.faceitlobby.com"
		req.SetBasicAuth("admin", "supersecretadminpassword")
		req.Header.Set("Origin", "http://attacker.com")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for foreign origin, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("CSRF origin check rejects localhost origin against production admin host", func(t *testing.T) {
		body := `{"id":"11111111-1111-1111-1111-111111111111","name":"LocalhostSpoof","url":"http://test.com","enabled":true}`
		req := httptest.NewRequest(http.MethodPost, "/admin/api/streamers", strings.NewReader(body))
		req.Host = "admin.faceitlobby.com"
		req.SetBasicAuth("admin", "supersecretadminpassword")
		req.Header.Set("Origin", "http://localhost:3000")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden when localhost origin is sent to production admin host, got %d", rec.Code)
		}
	})

	t.Run("CSRF origin check accepts exact production origin", func(t *testing.T) {
		body := `{"id":"11111111-1111-1111-1111-111111111111","name":"ValidStreamer","url":"http://test.com","enabled":true}`
		req := httptest.NewRequest(http.MethodPost, "/admin/api/streamers", strings.NewReader(body))
		req.Host = "admin.faceitlobby.com"
		req.SetBasicAuth("admin", "supersecretadminpassword")
		req.Header.Set("Origin", "https://admin.faceitlobby.com")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201 Created for exact admin origin, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestAdminStreamerValidation(t *testing.T) {
	srv, _ := setupAdminTestServer(t)

	cases := []struct {
		name       string
		payload    models.Streamer
		expectCode int
	}{
		{
			name: "Invalid uppercase UUID",
			payload: models.Streamer{
				ID:      "11111111-1111-1111-1111-11111111111G", // invalid hex
				Name:    "InvalidUUID",
				URL:     "http://example.com",
				Enabled: true,
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Empty name",
			payload: models.Streamer{
				ID:      "22222222-2222-2222-2222-222222222222",
				Name:    "   ",
				URL:     "http://example.com",
				Enabled: true,
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "URL with credentials prohibited",
			payload: models.Streamer{
				ID:      "33333333-3333-3333-3333-333333333333",
				Name:    "CredentialsProhibited",
				URL:     "http://user:password@example.com/lobby",
				Enabled: true,
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Non-http URL scheme prohibited",
			payload: models.Streamer{
				ID:      "44444444-4444-4444-4444-444444444444",
				Name:    "FtpProhibited",
				URL:     "ftp://example.com/lobby",
				Enabled: true,
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Valid Streamer Creation",
			payload: models.Streamer{
				ID:      "55555555-5555-5555-5555-555555555555",
				Name:    "ValidStreamer5",
				URL:     "https://streamer5.faceitlobby.com",
				Enabled: true,
			},
			expectCode: http.StatusCreated,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest(http.MethodPost, "/admin/api/streamers", bytes.NewReader(b))
			req.Host = "127.0.0.1:12000"
			req.SetBasicAuth("admin", "supersecretadminpassword")
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			if rec.Code != tc.expectCode {
				t.Errorf("%s: expected status %d, got %d: %s", tc.name, tc.expectCode, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAdminStreamerCRUD(t *testing.T) {
	srv, _ := setupAdminTestServer(t)

	streamerID := "abcdef01-2345-6789-abcd-ef0123456789"
	createPayload := models.Streamer{
		ID:      streamerID,
		Name:    "TestStreamerCRUD",
		URL:     "https://crud.faceitlobby.com",
		Enabled: true,
	}
	b, _ := json.Marshal(createPayload)

	// CREATE
	req := httptest.NewRequest(http.MethodPost, "/admin/api/streamers", bytes.NewReader(b))
	req.Host = "127.0.0.1:12000"
	req.SetBasicAuth("admin", "supersecretadminpassword")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create failed with %d: %s", rec.Code, rec.Body.String())
	}

	// GET LIST
	reqGet := httptest.NewRequest(http.MethodGet, "/admin/api/streamers", nil)
	reqGet.Host = "127.0.0.1:12000"
	reqGet.SetBasicAuth("admin", "supersecretadminpassword")
	recGet := httptest.NewRecorder()
	srv.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("get list failed with %d", recGet.Code)
	}
	if !strings.Contains(recGet.Body.String(), "TestStreamerCRUD") {
		t.Errorf("created streamer missing in list: %s", recGet.Body.String())
	}

	// UPDATE
	updatePayload := models.Streamer{
		Name:    "TestStreamerCRUDUpdated",
		URL:     "https://updated.faceitlobby.com",
		Enabled: false,
	}
	bUpdate, _ := json.Marshal(updatePayload)
	reqPut := httptest.NewRequest(http.MethodPut, "/admin/api/streamers/"+streamerID, bytes.NewReader(bUpdate))
	reqPut.Host = "127.0.0.1:12000"
	reqPut.SetBasicAuth("admin", "supersecretadminpassword")
	recPut := httptest.NewRecorder()
	srv.ServeHTTP(recPut, reqPut)

	if recPut.Code != http.StatusOK {
		t.Fatalf("update failed with %d: %s", recPut.Code, recPut.Body.String())
	}

	// DELETE
	reqDel := httptest.NewRequest(http.MethodDelete, "/admin/api/streamers/"+streamerID, nil)
	reqDel.Host = "127.0.0.1:12000"
	reqDel.SetBasicAuth("admin", "supersecretadminpassword")
	recDel := httptest.NewRecorder()
	srv.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusOK {
		t.Fatalf("delete failed with %d: %s", recDel.Code, recDel.Body.String())
	}

	// DELETE AGAIN -> 404
	recDel2 := httptest.NewRecorder()
	reqDel2 := httptest.NewRequest(http.MethodDelete, "/admin/api/streamers/"+streamerID, nil)
	reqDel2.Host = "127.0.0.1:12000"
	reqDel2.SetBasicAuth("admin", "supersecretadminpassword")
	srv.ServeHTTP(recDel2, reqDel2)

	if recDel2.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleting non-existent streamer, got %d", recDel2.Code)
	}
}

func TestHostSegregation(t *testing.T) {
	srv, _ := setupAdminTestServer(t)

	t.Run("Admin Host serves UI and blocks public routes", func(t *testing.T) {
		// Root on admin host serves UI
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "admin.faceitlobby.com"
		req.SetBasicAuth("admin", "supersecretadminpassword")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for UI on admin host root, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "BonBonn Game Tracker") {
			t.Errorf("unexpected UI content: %s", rec.Body.String())
		}

		// Public endpoint /elo on admin host returns 404
		reqElo := httptest.NewRequest(http.MethodGet, "/elo?username=testuser", nil)
		reqElo.Host = "admin.faceitlobby.com"
		reqElo.SetBasicAuth("admin", "supersecretadminpassword")
		recElo := httptest.NewRecorder()
		srv.ServeHTTP(recElo, reqElo)

		if recElo.Code != http.StatusNotFound {
			t.Errorf("expected 404 for public route on admin host, got %d", recElo.Code)
		}
	})

	t.Run("Public Host blocks admin routes", func(t *testing.T) {
		// Admin UI on public host returns 404
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Host = "api.faceitlobby.com"
		req.SetBasicAuth("admin", "supersecretadminpassword")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 for /admin on public host, got %d", rec.Code)
		}

		// Admin API on public host returns 404
		reqAPI := httptest.NewRequest(http.MethodGet, "/admin/api/status", nil)
		reqAPI.Host = "api.faceitlobby.com"
		reqAPI.SetBasicAuth("admin", "supersecretadminpassword")
		recAPI := httptest.NewRecorder()
		srv.ServeHTTP(recAPI, reqAPI)

		if recAPI.Code != http.StatusNotFound {
			t.Errorf("expected 404 for /admin/api on public host, got %d", recAPI.Code)
		}
	})
}
