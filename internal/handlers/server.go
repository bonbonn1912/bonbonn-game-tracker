package handlers

import (
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strings"

	"github.com/bonbonn1912/bonbonn-game-tracker/internal/config"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/faceit"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/store"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/web"
)

// Server aggregates router, handlers, store, and config.
type Server struct {
	cfg          *config.Config
	store        store.Store
	faceitClient *faceit.Client
	legacy       *LegacyHandler
	admin        *AdminHandler
	staticFS     fs.FS
	handler      http.Handler
}

// NewServer builds and configures the HTTP server and routing.
func NewServer(cfg *config.Config, s store.Store, fc *faceit.Client) (*Server, error) {
	staticFS, err := web.GetStaticFS()
	if err != nil {
		return nil, fmt.Errorf("failed to load static embedded filesystem: %w", err)
	}

	legacyH := NewLegacyHandler(cfg, s, fc)
	adminH := NewAdminHandler(cfg, s)

	srv := &Server{
		cfg:          cfg,
		store:        s,
		faceitClient: fc,
		legacy:       legacyH,
		admin:        adminH,
		staticFS:     staticFS,
	}

	srv.handler = srv.buildRouter()
	return srv, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) buildRouter() http.Handler {
	return LoggingMiddleware(RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqHost := r.Host
		if h, _, err := net.SplitHostPort(reqHost); err == nil {
			reqHost = h
		}
		reqHost = strings.ToLower(reqHost)

		adminHost := strings.ToLower(s.cfg.AdminHost)
		if h, _, err := net.SplitHostPort(adminHost); err == nil {
			adminHost = h
		}

		isAdminHost := adminHost != "" && reqHost == adminHost
		isLocal := isLocalHost(reqHost)
		isPublicHost := !isAdminHost && !isLocal

		path := r.URL.Path
		normPath := strings.ToLower(strings.TrimRight(path, "/"))
		if normPath == "" {
			normPath = "/"
		}

		// Dedicated health check endpoint: available on all hosts without auth
		if normPath == "/healthz" {
			s.handleHealthz(w, r)
			return
		}

		// Host segregation: Admin host must ONLY serve Admin UI and Admin API
		if isAdminHost {
			if normPath == "/" || strings.HasPrefix(normPath, "/admin") || normPath == "/index.html" || normPath == "/app.js" || normPath == "/style.css" {
				s.dispatchAdminRoute(w, r, normPath)
				return
			}
			s.legacy.HandleNotFound(w, r)
			return
		}

		// Host segregation: Public API host must BLOCK /admin and /admin/api
		if isPublicHost && strings.HasPrefix(normPath, "/admin") {
			s.legacy.HandleNotFound(w, r)
			return
		}

		// If on local host, /admin routes go to admin dispatcher
		if isLocal && (strings.HasPrefix(normPath, "/admin") || normPath == "/index.html" || normPath == "/app.js" || normPath == "/style.css") {
			s.dispatchAdminRoute(w, r, normPath)
			return
		}

		// Public API routing wrapped in PublicCORSMiddleware
		PublicCORSMiddleware(http.HandlerFunc(func(pw http.ResponseWriter, pr *http.Request) {
			s.dispatchPublicRoute(pw, pr, normPath)
		})).ServeHTTP(w, r)
	})))
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"message":"method not allowed"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := s.store.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"error","database":"unreachable"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","database":"ok"}`))
}

func (s *Server) dispatchPublicRoute(w http.ResponseWriter, r *http.Request, normPath string) {
	if r.Method == http.MethodHead {
		copy := r.Clone(r.Context())
		copy.Method = http.MethodGet
		r = copy
	}
	switch {
	case normPath == "/test":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			s.legacy.HandleTest(w, r)
			return
		}
	case normPath == "/elo":
		if r.Method == http.MethodGet {
			s.legacy.HandleElo("ELO")(w, r)
			return
		}
	case normPath == "/checkelo":
		if r.Method == http.MethodGet {
			s.legacy.HandleElo("CHECKELO")(w, r)
			return
		}
	case normPath == "/elobyid":
		if r.Method == http.MethodGet {
			s.legacy.HandleEloByID(w, r)
			return
		}
	case normPath == "/elobysteamid":
		if r.Method == http.MethodGet {
			s.legacy.HandleEloBySteamID(w, r)
			return
		}
	case normPath == "/matchhistory":
		if r.Method == http.MethodGet {
			s.legacy.HandleMatchHistory(w, r)
			return
		}
	case normPath == "/getmatch":
		if r.Method == http.MethodGet {
			s.legacy.HandleGetMatch(w, r)
			return
		}
	case normPath == "/redirect":
		if r.Method == http.MethodGet {
			s.legacy.HandleRedirect(w, r)
			return
		}
	case normPath == "/match":
		if r.Method == http.MethodPost {
			s.legacy.HandleMatch(w, r)
			return
		}
	case normPath == "/internal/api/getmultipleplayers":
		if r.Method == http.MethodPost {
			s.legacy.HandleGetMultiplePlayers(w, r)
			return
		}
	}

	s.legacy.HandleNotFound(w, r)
}

func (s *Server) dispatchAdminRoute(w http.ResponseWriter, r *http.Request, normPath string) {
	// All Admin UI and Admin API routes require Basic Auth and CSRF protection
	authHandler := AdminAuthMiddleware(s.cfg, CSRFOriginMiddleware(s.cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Admin API
		if strings.HasPrefix(normPath, "/admin/api") {
			subPath := strings.TrimPrefix(normPath, "/admin/api")

			switch {
			case subPath == "/status" && r.Method == http.MethodGet:
				s.admin.HandleGetStatus(w, r)
				return
			case subPath == "/events" && r.Method == http.MethodGet:
				s.admin.HandleGetEvents(w, r)
				return
			case subPath == "/webhook-requests" && r.Method == http.MethodGet:
				s.admin.HandleGetWebhookRequests(w, r)
				return
			case subPath == "/streamers" && r.Method == http.MethodGet:
				s.admin.HandleGetStreamers(w, r)
				return
			case subPath == "/streamers" && r.Method == http.MethodPost:
				s.admin.HandleCreateStreamer(w, r)
				return
			case strings.HasPrefix(subPath, "/streamers/"):
				id := strings.TrimPrefix(subPath, "/streamers/")
				if id != "" {
					if r.Method == http.MethodPut {
						s.admin.HandleUpdateStreamer(w, r, id)
						return
					}
					if r.Method == http.MethodDelete {
						s.admin.HandleDeleteStreamer(w, r, id)
						return
					}
				}
			case subPath == "/players" && r.Method == http.MethodGet:
				s.admin.HandleGetPlayers(w, r)
				return
			case strings.HasPrefix(subPath, "/players/") && strings.HasSuffix(subPath, "/matches") && r.Method == http.MethodGet:
				id := strings.TrimPrefix(subPath, "/players/")
				id = strings.TrimSuffix(id, "/matches")
				id = strings.Trim(id, "/")
				if id != "" {
					s.admin.HandleGetPlayerMatches(w, r, id)
					return
				}
			case subPath == "/matches" && r.Method == http.MethodGet:
				s.admin.HandleGetMatches(w, r)
				return
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
			return
		}

		// Admin UI Static Assets
		s.serveAdminStatic(w, r, normPath)
	})))

	authHandler.ServeHTTP(w, r)
}

func (s *Server) serveAdminStatic(w http.ResponseWriter, r *http.Request, normPath string) {
	var filePath string
	switch normPath {
	case "/", "/admin", "/index.html":
		filePath = "index.html"
	case "/app.js", "/admin/app.js":
		filePath = "app.js"
	case "/style.css", "/admin/style.css":
		filePath = "style.css"
	default:
		clean := strings.TrimPrefix(normPath, "/admin/")
		clean = strings.TrimPrefix(clean, "/")
		if clean == "" {
			filePath = "index.html"
		} else {
			filePath = clean
		}
	}

	file, err := s.staticFS.Open(filePath)
	if err != nil {
		file, err = s.staticFS.Open("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		filePath = "index.html"
	}
	defer file.Close()

	switch {
	case strings.HasSuffix(filePath, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(filePath, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(filePath, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	}

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
