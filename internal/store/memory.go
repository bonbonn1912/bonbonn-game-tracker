package store

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/bonbonn1912/bonbonn-game-tracker/internal/models"
)

type MemoryStore struct {
	mu              sync.RWMutex
	streamers       map[string]models.Streamer
	liveGames       map[string]map[string]interface{}
	redirectLogs    []models.RedirectEvent
	players         []map[string]interface{}
	webhookPlayers  map[string]models.WebhookPlayer
	webhookMatches  []models.WebhookMatch
	webhookEvents   []models.WebhookEvent
	webhookRequests []models.WebhookRequest
	pingErr         error
}

func NewMemoryStore() *MemoryStore {
	m := &MemoryStore{
		streamers:       make(map[string]models.Streamer),
		liveGames:       make(map[string]map[string]interface{}),
		redirectLogs:    make([]models.RedirectEvent, 0),
		players:         make([]map[string]interface{}, 0),
		webhookPlayers:  make(map[string]models.WebhookPlayer),
		webhookMatches:  make([]models.WebhookMatch, 0),
		webhookEvents:   make([]models.WebhookEvent, 0),
		webhookRequests: make([]models.WebhookRequest, 0),
	}

	for _, s := range DefaultSeedStreamers {
		m.streamers[s.ID] = s
	}

	return m
}

func (m *MemoryStore) RecordWebhookRequest(ctx context.Context, request models.WebhookRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if request.ID == "" {
		request.ID = newMemUUID()
	}
	if request.Timestamp == "" {
		request.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	request.Headers = cloneHeaders(request.Headers)
	m.webhookRequests = append(m.webhookRequests, request)
	return nil
}

func (m *MemoryStore) GetWebhookRequests(ctx context.Context, limit int) ([]models.WebhookRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	start := len(m.webhookRequests) - limit
	if start < 0 {
		start = 0
	}
	result := make([]models.WebhookRequest, 0, len(m.webhookRequests)-start)
	for i := len(m.webhookRequests) - 1; i >= start; i-- {
		request := m.webhookRequests[i]
		request.Headers = cloneHeaders(request.Headers)
		result = append(result, request)
	}
	return result, nil
}

func cloneHeaders(headers map[string][]string) map[string][]string {
	copy := make(map[string][]string, len(headers))
	for key, values := range headers {
		copy[key] = append([]string(nil), values...)
	}
	return copy
}

func (m *MemoryStore) SetPingError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pingErr = err
}

func (m *MemoryStore) GetStreamers(ctx context.Context) ([]models.Streamer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]models.Streamer, 0, len(m.streamers))
	for _, s := range m.streamers {
		result = append(result, s)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

func (m *MemoryStore) GetStreamer(ctx context.Context, id string) (*models.Streamer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.streamers[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &s, nil
}

func (m *MemoryStore) CreateStreamer(ctx context.Context, streamer models.Streamer) (*models.Streamer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.streamers[streamer.ID]; exists {
		return nil, ErrDuplicateKey
	}

	m.streamers[streamer.ID] = streamer
	return &streamer, nil
}

func (m *MemoryStore) UpdateStreamer(ctx context.Context, id string, streamer models.Streamer) (*models.Streamer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.streamers[id]; !exists {
		return nil, ErrNotFound
	}

	streamer.ID = id
	m.streamers[id] = streamer
	return &streamer, nil
}

func (m *MemoryStore) DeleteStreamer(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.streamers[id]; !exists {
		return ErrNotFound
	}

	delete(m.streamers, id)
	return nil
}

func (m *MemoryStore) LogRedirectEvent(ctx context.Context, event models.RedirectEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.redirectLogs = append(m.redirectLogs, event)
	return nil
}

func (m *MemoryStore) GetRedirectEvents(ctx context.Context, limit int) ([]models.RedirectEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	total := len(m.redirectLogs)
	start := 0
	if total > limit {
		start = total - limit
	}

	slice := make([]models.RedirectEvent, total-start)
	copy(slice, m.redirectLogs[start:])

	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}

	return slice, nil
}

func (m *MemoryStore) AddLiveMatch(ctx context.Context, streamerID string, matchData map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.liveGames[streamerID]; exists {
		return ErrAlreadyLive
	}

	docCopy := cloneJSONMap(matchData)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	docCopy["meta"] = map[string]interface{}{
		"inserted_at": now,
		"timestamp":   now,
		"isRunning":   true,
	}
	docCopy["streamer"] = streamerID
	m.liveGames[streamerID] = docCopy

	// Enrich canonical record in webhookMatches
	matchID := getNestedString(matchData, "payload", "id")
	if matchID != "" {
		payload, _ := matchData["payload"].(map[string]interface{})
		mapVal := matchData["map"]
		matchupVal := matchData["matchup"]

		found := false
		for i := range m.webhookMatches {
			if m.webhookMatches[i].PlayerID == streamerID && m.webhookMatches[i].MatchID == matchID {
				// Enrichment should not reset closed canonical back to live!
				if m.webhookMatches[i].Status != "closed" {
					m.webhookMatches[i].Status = "live"
				}
				m.webhookMatches[i].LastSeenAt = now
				if mapVal != nil && mapVal != "" && mapVal != "unknown" {
					m.webhookMatches[i].Map = mapVal
				}
				if matchupVal != nil {
					m.webhookMatches[i].Matchup = matchupVal
				}
				if hasTeams(payload) {
					m.webhookMatches[i].Payload = cloneJSONMap(payload)
				}
				found = true
				break
			}
		}

		if !found {
			username := streamerID
			if s, ok := m.streamers[streamerID]; ok && s.Name != "" {
				username = s.Name
			}
			m.webhookMatches = append(m.webhookMatches, models.WebhookMatch{
				ID:          fmt.Sprintf("%s:%s", streamerID, matchID),
				PlayerID:    streamerID,
				Username:    username,
				MatchID:     matchID,
				Status:      "live",
				FirstSeenAt: now,
				LastSeenAt:  now,
				Map:         mapVal,
				Matchup:     matchupVal,
				Payload:     cloneJSONMap(payload),
				Event:       "match_status_configuring",
			})
			// Update player game count
			count := 0
			for _, wm := range m.webhookMatches {
				if wm.PlayerID == streamerID && wm.MatchID != "" {
					count++
				}
			}
			if p, ok := m.webhookPlayers[streamerID]; ok {
				p.GameCount = count
				m.webhookPlayers[streamerID] = p
			}
		}
	}

	return nil
}

func (m *MemoryStore) GetLiveMatch(ctx context.Context, streamerID string) (map[string]interface{}, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	g, ok := m.liveGames[streamerID]
	if !ok {
		return nil, false, nil
	}
	res := cloneJSONMap(g)
	delete(res, "_id")
	delete(res, "meta")
	return res, true, nil
}

func (m *MemoryStore) CloseLiveMatch(ctx context.Context, streamerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.liveGames, streamerID)

	// Close matching live match for streamerID only
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for i := range m.webhookMatches {
		if m.webhookMatches[i].PlayerID == streamerID && m.webhookMatches[i].Status == "live" {
			m.webhookMatches[i].Status = "closed"
			m.webhookMatches[i].LastSeenAt = now
		}
	}

	return nil
}

func (m *MemoryStore) GetAllLiveMatches(ctx context.Context) (map[string]map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make(map[string]map[string]interface{}, len(m.liveGames))
	for k, v := range m.liveGames {
		clean := cloneJSONMap(v)
		delete(clean, "_id")
		delete(clean, "meta")
		res[k] = clean
	}
	return res, nil
}

func (m *MemoryStore) CountActiveMatches(ctx context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.liveGames), nil
}

func (m *MemoryStore) SavePlayer(ctx context.Context, insertType string, playerData map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	docCopy := cloneJSONMap(playerData)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	docCopy["insertType"] = insertType
	docCopy["meta"] = map[string]interface{}{
		"inserted_at": now,
		"timestamp":   now,
		"isRunning":   nil,
	}
	m.players = append(m.players, docCopy)
	return nil
}

func (m *MemoryStore) RecordWebhook(ctx context.Context, playerID string, resolvedUsername string, body map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	event := getNestedString(body, "event")
	matchID := getNestedString(body, "payload", "id")

	// 1. Raw event
	m.webhookEvents = append(m.webhookEvents, models.WebhookEvent{
		ID:        newMemUUID(),
		PlayerID:  playerID,
		MatchID:   matchID,
		Event:     event,
		Timestamp: now,
		RawBody:   cloneJSONMap(body),
	})

	// 2. Webhook player
	wp, exists := m.webhookPlayers[playerID]
	if !exists {
		wp = models.WebhookPlayer{
			ID:            playerID,
			Username:      resolvedUsername,
			GameCount:     0,
			LastWebhookAt: now,
		}
	} else {
		wp.LastWebhookAt = now
		if resolvedUsername != "" && resolvedUsername != playerID {
			wp.Username = resolvedUsername
		}
	}

	// 3. Webhook match
	if matchID != "" {
		status := "live"
		if event != "match_status_configuring" && event != "match_object_created" && event != "" {
			status = "closed"
		}

		payload, _ := body["payload"].(map[string]interface{})
		mapVal := body["map"]
		matchupVal := body["matchup"]

		foundIdx := -1
		for i, wm := range m.webhookMatches {
			if wm.PlayerID == playerID && wm.MatchID == matchID {
				foundIdx = i
				break
			}
		}

		if foundIdx == -1 {
			newMatch := models.WebhookMatch{
				ID:          fmt.Sprintf("%s:%s", playerID, matchID),
				PlayerID:    playerID,
				Username:    resolvedUsername,
				MatchID:     matchID,
				Status:      status,
				FirstSeenAt: now,
				LastSeenAt:  now,
				Map:         mapVal,
				Matchup:     matchupVal,
				Payload:     cloneJSONMap(payload),
				Event:       event,
			}
			m.webhookMatches = append(m.webhookMatches, newMatch)
		} else {
			existing := &m.webhookMatches[foundIdx]
			existing.LastSeenAt = now
			existing.Event = event
			// Closing canonical must stay closed on delayed config retries!
			if existing.Status == "closed" {
				existing.Status = "closed"
			} else if status != "" {
				existing.Status = status
			}
			if resolvedUsername != "" && resolvedUsername != playerID {
				existing.Username = resolvedUsername
			}
			if mapVal != nil && mapVal != "" && mapVal != "unknown" {
				existing.Map = mapVal
			}
			if matchupVal != nil {
				existing.Matchup = matchupVal
			}
			if hasTeams(payload) || !hasTeams(existing.Payload) {
				existing.Payload = cloneJSONMap(payload)
			}
		}

		// Recalculate distinct game count for player
		count := 0
		for _, wm := range m.webhookMatches {
			if wm.PlayerID == playerID && wm.MatchID != "" {
				count++
			}
		}
		wp.GameCount = count
	}

	m.webhookPlayers[playerID] = wp
	return nil
}

func (m *MemoryStore) GetWebhookPlayer(ctx context.Context, playerID string) (*models.WebhookPlayer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	wp, ok := m.webhookPlayers[playerID]
	if !ok {
		return nil, ErrNotFound
	}
	return &wp, nil
}

func (m *MemoryStore) GetWebhookPlayers(ctx context.Context) ([]models.WebhookPlayer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	players := make([]models.WebhookPlayer, 0, len(m.webhookPlayers))
	for _, p := range m.webhookPlayers {
		players = append(players, p)
	}

	sort.Slice(players, func(i, j int) bool {
		if players[i].LastWebhookAt != players[j].LastWebhookAt {
			return players[i].LastWebhookAt > players[j].LastWebhookAt
		}
		return players[i].ID < players[j].ID
	})

	return players, nil
}

func (m *MemoryStore) GetPlayerMatches(ctx context.Context, playerID string, limit, offset int) ([]models.WebhookMatch, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var filtered []models.WebhookMatch
	for _, wm := range m.webhookMatches {
		if wm.PlayerID == playerID && wm.MatchID != "" {
			filtered = append(filtered, wm)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].LastSeenAt != filtered[j].LastSeenAt {
			return filtered[i].LastSeenAt > filtered[j].LastSeenAt
		}
		return filtered[i].ID < filtered[j].ID
	})

	total := len(filtered)
	if offset >= total {
		return []models.WebhookMatch{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	result := make([]models.WebhookMatch, end-offset)
	copy(result, filtered[offset:end])
	return result, total, nil
}

func (m *MemoryStore) GetAdminMatches(ctx context.Context) (live []models.WebhookMatch, past []models.WebhookMatch, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	live = make([]models.WebhookMatch, 0)
	past = make([]models.WebhookMatch, 0)

	// 1. Live authoritative from liveGames
	for streamerID, g := range m.liveGames {
		matchID := getNestedString(g, "payload", "id")
		found := false
		for _, wm := range m.webhookMatches {
			if wm.PlayerID == streamerID && wm.MatchID == matchID {
				clone := wm
				clone.Status = "live"
				if clone.Map == nil || clone.Map == "" {
					clone.Map = g["map"]
				}
				if clone.Matchup == nil {
					clone.Matchup = g["matchup"]
				}
				live = append(live, clone)
				found = true
				break
			}
		}
		if !found {
			streamerName := streamerID
			if s, ok := m.streamers[streamerID]; ok && s.Name != "" {
				streamerName = s.Name
			}
			now := time.Now().UTC().Format(time.RFC3339Nano)
			pMap, _ := g["payload"].(map[string]interface{})
			if pMap == nil {
				pMap = map[string]interface{}{}
			}
			live = append(live, models.WebhookMatch{
				ID:          fmt.Sprintf("%s:%s", streamerID, matchID),
				PlayerID:    streamerID,
				Username:    streamerName,
				MatchID:     matchID,
				Status:      "live",
				FirstSeenAt: now,
				LastSeenAt:  now,
				Map:         g["map"],
				Matchup:     g["matchup"],
				Payload:     pMap,
				Event:       "match_status_configuring",
			})
		}
	}

	sort.Slice(live, func(i, j int) bool {
		if live[i].LastSeenAt != live[j].LastSeenAt {
			return live[i].LastSeenAt > live[j].LastSeenAt
		}
		return live[i].ID < live[j].ID
	})

	// 2. Past: last 20 completed matches
	var completed []models.WebhookMatch
	for _, wm := range m.webhookMatches {
		if wm.Status == "closed" && wm.MatchID != "" {
			completed = append(completed, wm)
		}
	}

	sort.Slice(completed, func(i, j int) bool {
		if completed[i].LastSeenAt != completed[j].LastSeenAt {
			return completed[i].LastSeenAt > completed[j].LastSeenAt
		}
		return completed[i].ID < completed[j].ID
	})

	seen := make(map[string]bool)
	for _, match := range completed {
		if !seen[match.MatchID] {
			seen[match.MatchID] = true
			past = append(past, match)
			if len(past) == 20 {
				break
			}
		}
	}

	return live, past, nil
}

func (m *MemoryStore) Ping(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pingErr
}

func (m *MemoryStore) Close(ctx context.Context) error {
	return nil
}

func newMemUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func cloneJSONMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		c := make(map[string]interface{}, len(m))
		for k, v := range m {
			c[k] = v
		}
		return c
	}
	var res map[string]interface{}
	_ = json.Unmarshal(data, &res)
	return res
}
