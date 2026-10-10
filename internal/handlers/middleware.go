package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bonbonn1912/bonbonn-game-tracker/internal/config"
)

type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
	isHead     bool
	bodyLength int
}

func (w *responseWriterInterceptor) WriteHeader(code int) {
	if w.statusCode != 0 {
		return
	}
	w.statusCode = code
	if !w.isHead {
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *responseWriterInterceptor) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	if w.isHead {
		w.bodyLength += len(b)
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}

// RecoveryMiddleware recovers from panics and returns 500 error response.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered in http handler", "error", rec, "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"message":"Internal Server Error"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs HTTP requests in slog JSON without leaking secrets.
// Enforces 100KB request body limit matching Express default (bodyParser.json limit 100kb).
// Automatically supports HEAD methods by discarding body bytes while returning full headers.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = generateUUID()
			r.Header.Set("X-Request-Id", reqID)
		}

		// Enforce Express default 100KB body limit
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, 100*1024)
		}

		isHead := r.Method == http.MethodHead
		rw := &responseWriterInterceptor{ResponseWriter: w, statusCode: 0, isHead: isHead}
		rw.Header().Set("X-Request-Id", reqID)

		next.ServeHTTP(rw, r)

		if rw.statusCode == 0 {
			rw.statusCode = http.StatusOK
		}

		if isHead {
			if rw.Header().Get("Content-Length") == "" && rw.statusCode != http.StatusNoContent && rw.statusCode != http.StatusNotModified {
				rw.Header().Set("Content-Length", fmt.Sprint(rw.bodyLength))
			}
			rw.ResponseWriter.WriteHeader(rw.statusCode)
		}
		duration := time.Since(start)
		clientIP := getClientIP(r)

		slog.Info("http request",
			"requestId", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"durationMs", duration.Milliseconds(),
			"ip", clientIP,
			"userAgent", r.UserAgent(),
		)
	})
}

// PublicCORSMiddleware applies Express-compatible CORS headers for public routes only.
// Preflight returns 204 No Content with methods GET,HEAD,PUT,PATCH,POST,DELETE.
func PublicCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,PUT,PATCH,POST,DELETE")
			if headers := r.Header.Get("Access-Control-Request-Headers"); headers != "" {
				w.Header().Set("Access-Control-Allow-Headers", headers)
			}
			w.Header().Add("Vary", "Access-Control-Request-Headers")
			w.Header().Set("Content-Length", "0")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AdminAuthMiddleware validates HTTP Basic Auth against configured admin credentials.
func AdminAuthMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		user, pass, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Bonbonn Admin"`)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"Unauthorized"}`))
			return
		}

		userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(cfg.AdminUsername)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(cfg.AdminPassword)) == 1

		if !userMatch || !passMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="Bonbonn Admin"`)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"Unauthorized"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// CSRFOriginMiddleware enforces exact scheme + authority (including port) for mutation requests.
// Never allows localhost origins against production hosts.
func CSRFOriginMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			http.Error(w, "Forbidden: cross-site request", http.StatusForbidden)
			return
		}
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = r.Header.Get("Referer")
		}

		if origin != "" {
			parsed, err := url.Parse(origin)
			if err != nil || !isExactAllowedOrigin(parsed, r, cfg.AdminHost) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"message":"Forbidden: invalid origin"}`))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func isExactAllowedOrigin(originURL *url.URL, r *http.Request, adminHost string) bool {
	scheme := "https"
	if isLocalHost(r.Host) && r.TLS == nil {
		scheme = "http"
	}
	if originURL.Scheme != scheme || originURL.User != nil || originURL.Host == "" {
		return false
	}
	normalizeAuthority := func(authority string) string {
		u := &url.URL{Scheme: scheme, Host: strings.ToLower(authority)}
		port := u.Port()
		if port == "" {
			if scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}
		return net.JoinHostPort(u.Hostname(), port)
	}
	return normalizeAuthority(originURL.Host) == normalizeAuthority(r.Host)
}

func isLocalHost(host string) bool {
	h := host
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		h = parsedHost
	}
	h = strings.ToLower(h)
	return h == "localhost" || h == "127.0.0.1" || h == "::1" || h == "testserver" || h == ""
}

func getClientIP(r *http.Request) string {
	xf := r.Header.Get("X-Forwarded-For")
	if xf != "" {
		parts := strings.Split(xf, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func generateUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
