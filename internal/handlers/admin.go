package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/bonbonn1912/bonbonn-game-tracker/internal/config"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/models"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/store"
)

var lowerUUIDRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type AdminHandler struct {
	cfg   *config.Config
	store store.Store
}

func NewAdminHandler(cfg *config.Config, s store.Store) *AdminHandler {
	return &AdminHandler{
		cfg:   cfg,
		store: s,
	}
}

// HandleGetStatus handles GET /admin/api/status.
func (h *AdminHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dbStatus := "ok"
	if err := h.store.Ping(ctx); err != nil {
		dbStatus = "unreachable"
	}

	activeMatches, err := h.store.CountActiveMatches(ctx)
	if err != nil {
		dbStatus = "unreachable"
		activeMatches = 0
	}

	status := models.AdminStatus{
		Database:      dbStatus,
		ActiveMatches: activeMatches,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(status)
}

// HandleGetStreamers handles GET /admin/api/streamers.
func (h *AdminHandler) HandleGetStreamers(w http.ResponseWriter, r *http.Request) {
	streamers, err := h.store.GetStreamers(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"failed to fetch streamers"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(streamers)
}

// HandleCreateStreamer handles POST /admin/api/streamers.
func (h *AdminHandler) HandleCreateStreamer(w http.ResponseWriter, r *http.Request) {
	var s models.Streamer
	if err := decodeAdminStreamer(r, &s); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"invalid json payload"}`))
		return
	}

	// Lowercase UUID validation
	s.ID = strings.TrimSpace(s.ID)
	if !lowerUUIDRegex.MatchString(s.ID) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"streamer id must be a valid lowercase UUID"}`))
		return
	}

	// Name validation
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" || len(s.Name) > 64 {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"streamer name must be non-empty and at most 64 characters"}`))
		return
	}

	// URL validation
	s.URL = strings.TrimSpace(s.URL)
	if s.URL != "" {
		if err := validateURL(s.URL); err != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid streamer url: " + err.Error()})
			return
		}
	}

	created, err := h.store.CreateStreamer(r.Context(), s)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if errors.Is(err, store.ErrDuplicateKey) {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"message":"streamer already exists"}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"failed to create streamer"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// HandleUpdateStreamer handles PUT /admin/api/streamers/{id}.
func (h *AdminHandler) HandleUpdateStreamer(w http.ResponseWriter, r *http.Request, id string) {
	id = strings.ToLower(strings.TrimSpace(id))
	if !lowerUUIDRegex.MatchString(id) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"streamer id must be a valid lowercase UUID"}`))
		return
	}

	var s models.Streamer
	if err := decodeAdminStreamer(r, &s); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"invalid json payload"}`))
		return
	}

	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" || len(s.Name) > 64 {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"streamer name must be non-empty and at most 64 characters"}`))
		return
	}

	s.URL = strings.TrimSpace(s.URL)
	if s.URL != "" {
		if err := validateURL(s.URL); err != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid streamer url: " + err.Error()})
			return
		}
	}

	updated, err := h.store.UpdateStreamer(r.Context(), id, s)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if errors.Is(err, store.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"streamer not found"}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"failed to update streamer"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(updated)
}

// HandleDeleteStreamer handles DELETE /admin/api/streamers/{id}.
func (h *AdminHandler) HandleDeleteStreamer(w http.ResponseWriter, r *http.Request, id string) {
	id = strings.ToLower(strings.TrimSpace(id))
	if !lowerUUIDRegex.MatchString(id) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"streamer id must be a valid lowercase UUID"}`))
		return
	}

	err := h.store.DeleteStreamer(r.Context(), id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if errors.Is(err, store.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"streamer not found"}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"failed to delete streamer"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"message":"streamer deleted"}`))
}

// HandleGetEvents handles GET /admin/api/events.
func (h *AdminHandler) HandleGetEvents(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	events, err := h.store.GetRedirectEvents(r.Context(), limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"failed to fetch redirect events"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(events)
}

// HandleGetWebhookRequests handles GET /admin/api/webhook-requests.
func (h *AdminHandler) HandleGetWebhookRequests(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	requests, err := h.store.GetWebhookRequests(r.Context(), limit)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"failed to fetch webhook requests"}`))
		return
	}
	if requests == nil {
		requests = []models.WebhookRequest{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(requests)
}

// HandleGetPlayers handles GET /admin/api/players.
func (h *AdminHandler) HandleGetPlayers(w http.ResponseWriter, r *http.Request) {
	players, err := h.store.GetWebhookPlayers(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"failed to fetch tracked players"}`))
		return
	}
	if players == nil {
		players = []models.WebhookPlayer{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(players)
}

// HandleGetPlayerMatches handles GET /admin/api/players/{id}/matches.
func (h *AdminHandler) HandleGetPlayerMatches(w http.ResponseWriter, r *http.Request, id string) {
	id = strings.ToLower(strings.TrimSpace(id))
	if !models.RegexFaceitID.MatchString(id) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"player id must be a valid UUID"}`))
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 200 {
		limit = 200
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	matches, total, err := h.store.GetPlayerMatches(r.Context(), id, limit, offset)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"failed to fetch player matches"}`))
		return
	}
	if matches == nil {
		matches = []models.WebhookMatch{}
	}

	resp := models.PlayerMatchesResponse{
		Matches: matches,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleGetMatches handles GET /admin/api/matches.
func (h *AdminHandler) HandleGetMatches(w http.ResponseWriter, r *http.Request) {
	live, past, err := h.store.GetAdminMatches(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"failed to fetch matches"}`))
		return
	}
	if live == nil {
		live = []models.WebhookMatch{}
	}
	if past == nil {
		past = []models.WebhookMatch{}
	}

	resp := models.MatchesResponse{
		Live: live,
		Past: past,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func decodeAdminStreamer(r *http.Request, s *models.Streamer) error {
	if contentType := r.Header.Get("Content-Type"); contentType != "" {
		media, _, err := mime.ParseMediaType(contentType)
		if err != nil || media != "application/json" {
			return fmt.Errorf("application/json required")
		}
	}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(s); err != nil {
		return err
	}
	var extra interface{}
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("single JSON document required")
	}
	return nil
}
