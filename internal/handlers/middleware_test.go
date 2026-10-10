package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHeadPublicResponses(t *testing.T) {
	srv, _ := setupAdminTestServer(t)
	for _, path := range []string{"/test", "/getmatch", "/healthz", "/unknown"} {
		get := httptest.NewRecorder()
		srv.ServeHTTP(get, httptest.NewRequest("GET", path, nil))
		head := httptest.NewRecorder()
		srv.ServeHTTP(head, httptest.NewRequest("HEAD", path, nil))
		if head.Code != get.Code || head.Body.Len() != 0 || head.Header().Get("Content-Type") != get.Header().Get("Content-Type") {
			t.Fatalf("HEAD mismatch %s: %d %s", path, head.Code, head.Body.String())
		}
		if head.Header().Get("Content-Length") == "" {
			t.Fatalf("missing HEAD content length: %s", path)
		}
	}
}

func TestAdminOriginSchemeAndPort(t *testing.T) {
	srv, _ := setupAdminTestServer(t)
	for _, origin := range []string{"http://admin.faceitlobby.com", "https://admin.faceitlobby.com:444", "https://admin.faceitlobby.com.evil.com", "null"} {
		r := httptest.NewRequest(http.MethodPost, "/admin/api/streamers", strings.NewReader(`{"id":"abcdef01-2345-6789-abcd-ef0123456789","name":"test","url":"https://example.com","enabled":true}`))
		r.Host = "admin.faceitlobby.com"
		r.SetBasicAuth("admin", "supersecretadminpassword")
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("accepted foreign origin %q: %d", origin, w.Code)
		}
	}
}

func TestAdminRejectsFormAndTrailingJSON(t *testing.T) {
	srv, _ := setupAdminTestServer(t)
	payload := `{"id":"abcdef01-2345-6789-abcd-ef0123456789","name":"test","url":"https://example.com","enabled":true}`
	for _, tc := range []struct{ contentType, body string }{{"text/plain", payload}, {"application/json", payload + ` {"ignored":true}`}} {
		r := httptest.NewRequest(http.MethodPost, "/admin/api/streamers", strings.NewReader(tc.body))
		r.Host = "admin.faceitlobby.com"
		r.SetBasicAuth("admin", "supersecretadminpassword")
		r.Header.Set("Content-Type", tc.contentType)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid body accepted: %d %s", w.Code, w.Body.String())
		}
	}
}
