package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestCheckHealthHelper(t *testing.T) {
	t.Run("Healthy server returns nil", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok","database":"ok"}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer ts.Close()

		_, portStr, err := net.SplitHostPort(ts.Listener.Addr().String())
		if err != nil {
			t.Fatalf("failed to split host port: %v", err)
		}

		if err := checkHealth(portStr); err != nil {
			t.Errorf("expected health check to succeed, got %v", err)
		}
	})

	t.Run("Failing server returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"error","database":"unreachable"}`))
		}))
		defer ts.Close()

		_, portStr, err := net.SplitHostPort(ts.Listener.Addr().String())
		if err != nil {
			t.Fatalf("failed to split host port: %v", err)
		}

		if err := checkHealth(portStr); err == nil {
			t.Errorf("expected health check to fail on 503")
		}
	})

	t.Run("Down server returns error", func(t *testing.T) {
		// Pick an unused port
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to listen: %v", err)
		}
		port := l.Addr().(*net.TCPAddr).Port
		_ = l.Close() // immediately close to make port unused

		if err := checkHealth(strconv.Itoa(port)); err == nil {
			t.Errorf("expected health check to fail on down server")
		}
	})
}
