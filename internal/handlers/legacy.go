package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bonbonn1912/bonbonn-game-tracker/internal/config"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/faceit"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/models"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/store"
)

type LegacyHandler struct {
	cfg          *config.Config
	store        store.Store
	faceitClient *faceit.Client
}

func NewLegacyHandler(cfg *config.Config, s store.Store, fc *faceit.Client) *LegacyHandler {
	return &LegacyHandler{
		cfg:          cfg,
		store:        s,
		faceitClient: fc,
	}
}

// HandleElo handles GET /elo and GET /checkelo.
func (h *LegacyHandler) HandleElo(insertType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := legacyQueryValue(r, "username")
		if username == "" || !models.RegexFaceitUsername.MatchString(username) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Nicknames can be made of letters (a-z, A-Z), numbers (0-9), '_' and '-' only. 3-12 characters"))
			return
		}

		var player *models.FaceitPlayer
		var err error

		// In legacy Express elo.ts: req.query.id overrides nickname lookup if present
		queryID := legacyQueryValue(r, "id")
		if r.URL.Query().Has("id") || r.URL.Query().Has("id[]") {
			player, err = h.faceitClient.GetPlayerByID(r.Context(), queryID)
		} else {
			player, err = h.faceitClient.GetPlayerByNickname(r.Context(), username)
		}

		if err != nil || player == nil || player.Games.CSGO == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Invalid username"))
			return
		}

		respStr := formatEloResponse(player)
		h.asyncSavePlayer(insertType, player, respStr)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respStr))
	}
}

// HandleEloByID handles GET /elobyid.
func (h *LegacyHandler) HandleEloByID(w http.ResponseWriter, r *http.Request) {
	id := legacyQueryValue(r, "id")
	if id == "" || !models.RegexFaceitID.MatchString(id) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Faceit Ids must match following pattern : ^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$"))
		return
	}

	player, err := h.faceitClient.GetPlayerByID(r.Context(), id)
	if err != nil || player == nil || player.Games.CSGO == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Invalid username"))
		return
	}

	respStr := formatEloResponse(player)
	h.asyncSavePlayer("ELOBYID", player, respStr)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(respStr))
}

// HandleEloBySteamID handles GET /elobysteamid.
func (h *LegacyHandler) HandleEloBySteamID(w http.ResponseWriter, r *http.Request) {
	id := legacyQueryValue(r, "id")
	if id == "" || !models.RegexSteam64.MatchString(id) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Steam 64 Ids are made of 17 numbers (0-9)"))
		return
	}

	player, err := h.faceitClient.GetPlayerByID(r.Context(), id)
	if err != nil || player == nil || player.Games.CSGO == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Invalid username"))
		return
	}

	respStr := formatEloResponse(player)
	h.asyncSavePlayer("ELOBYSTEAMID", player, respStr)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(respStr))
}

// HandleMatchHistory handles GET /matchhistory.
func (h *LegacyHandler) HandleMatchHistory(w http.ResponseWriter, r *http.Request) {
	username := legacyQueryValue(r, "username")
	limitStr := legacyQueryValue(r, "limit")

	limit, err := legacyLimit(limitStr)
	if err != nil || limit <= 3 || limit >= 101 || !models.RegexFaceitUsername.MatchString(username) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Incorrect faceit username or invalid limit (4-100)"))
		return
	}

	var player *models.FaceitPlayer
	queryID := legacyQueryValue(r, "id")
	if r.URL.Query().Has("id") || r.URL.Query().Has("id[]") {
		player, err = h.faceitClient.GetPlayerByID(r.Context(), queryID)
	} else {
		player, err = h.faceitClient.GetPlayerByNickname(r.Context(), username)
	}

	if err != nil || player == nil || player.Games.CSGO == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Invalid username"))
		return
	}

	history, err := h.faceitClient.GetPlayerHistoryCS2(r.Context(), player.PlayerID, limitStr)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Something went wrong"))
		return
	}

	matchHistory := make([]string, 0, len(history.Items))
	totalWins := 0
	for _, item := range history.Items {
		isWinner := false
		if item.Results.Winner == "faction1" {
			for _, p := range item.Teams.Faction1.Players {
				if p.PlayerID == player.PlayerID {
					isWinner = true
					break
				}
			}
		} else {
			for _, p := range item.Teams.Faction2.Players {
				if p.PlayerID == player.PlayerID {
					isWinner = true
					break
				}
			}
		}

		if isWinner {
			matchHistory = append(matchHistory, "1")
			totalWins++
		} else {
			matchHistory = append(matchHistory, "0")
		}
	}

	if len(matchHistory) <= 3 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Play at least 3 Games to see your history"))
		return
	}

	var parts []string
	for _, res := range matchHistory {
		if res == "1" {
			parts = append(parts, "W/")
		} else {
			parts = append(parts, "L/")
		}
	}
	winLoseStr := strings.TrimSuffix(strings.Join(parts, ""), "/")
	winrate := int(math.Floor(float64(totalWins) / float64(len(matchHistory)) * 100))
	respStr := fmt.Sprintf("%s's last %s Games: %s. Winrate: %d%%", player.Nickname, limitStr, winLoseStr, winrate)

	h.asyncSavePlayer("MATCHHISTORY", player, respStr)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(respStr))
}

// HandleGetMatch handles GET /getmatch.
func (h *LegacyHandler) HandleGetMatch(w http.ResponseWriter, r *http.Request) {
	key := legacyQueryValue(r, "key")
	if !models.RegexFaceitID.MatchString(key) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Invalid Key"))
		return
	}

	game, found, err := h.store.GetLiveMatch(r.Context(), key)
	if err != nil {
		h.databaseError(w, r, err)
		return
	}
	if !found {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("This User is currently not in a matchroom"))
		return
	}

	lobbyURL := ""
	streamer, err := h.store.GetStreamer(r.Context(), key)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		h.databaseError(w, r, err)
		return
	}
	if err == nil && streamer.Enabled && streamer.URL != "" {
		lobbyURL = streamer.URL
	} else {
		payloadID := getNestedString(game, "payload", "id")
		lobbyURL = fmt.Sprintf("https://www.faceit.com/en/cs2/room/%s", payloadID)
	}

	team1Name := getNestedString(game, "matchup", "team1", "name")
	team1Avg := getNestedValue(game, "matchup", "team1", "avgElo")
	team2Name := getNestedString(game, "matchup", "team2", "name")
	team2Avg := getNestedValue(game, "matchup", "team2", "avgElo")
	mapName := game["map"]

	respStr := fmt.Sprintf("%v (%v) vs. %v (%v) on %v. Link zum Matchroom: %s",
		team1Name, team1Avg, team2Name, team2Avg, mapName, lobbyURL)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(respStr))
}

// HandleRedirect handles GET /redirect.
func (h *LegacyHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	key := legacyQueryValue(r, "key")
	reqID := r.Header.Get("X-Request-Id")
	if reqID == "" {
		reqID = generateUUID()
	}
	clientIP := getClientIP(r)
	ua := r.UserAgent()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	streamerName := ""
	if s, err := h.store.GetStreamer(r.Context(), key); err == nil {
		streamerName = s.Name
	}

	if !models.RegexFaceitID.MatchString(key) {
		_ = h.store.LogRedirectEvent(r.Context(), models.RedirectEvent{
			ID:           generateUUID(),
			StreamerID:   key,
			StreamerName: streamerName,
			Outcome:      "invalid_key",
			CreatedAt:    now,
			RequestID:    reqID,
			IP:           clientIP,
			UserAgent:    ua,
		})

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Invalid Key"))
		return
	}

	game, found, err := h.store.GetLiveMatch(r.Context(), key)
	if err != nil {
		h.databaseError(w, r, err)
		return
	}
	if found {
		payloadID := getNestedString(game, "payload", "id")
		target := fmt.Sprintf("https://www.faceit.com/en/csgo/room/%s", payloadID)

		_ = h.store.LogRedirectEvent(r.Context(), models.RedirectEvent{
			ID:           generateUUID(),
			StreamerID:   key,
			StreamerName: streamerName,
			MatchID:      payloadID,
			Target:       target,
			Outcome:      "redirected",
			CreatedAt:    now,
			RequestID:    reqID,
			IP:           clientIP,
			UserAgent:    ua,
		})

		writeLegacyRedirect(w, r, target)
		return
	}

	_ = h.store.LogRedirectEvent(r.Context(), models.RedirectEvent{
		ID:           generateUUID(),
		StreamerID:   key,
		StreamerName: streamerName,
		Outcome:      "not_ingame",
		CreatedAt:    now,
		RequestID:    reqID,
		IP:           clientIP,
		UserAgent:    ua,
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf("Player with id: %s is currently not ingame", key)))
}

// HandleMatch handles POST /match.
func (h *LegacyHandler) HandleMatch(w http.ResponseWriter, r *http.Request) {
	requestBody, readErr := io.ReadAll(io.LimitReader(r.Body, 100*1024+1))
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(requestBody))
	if readErr != nil {
		requestBody = []byte("[body read error: " + readErr.Error() + "]")
	}
	if len(requestBody) > 100*1024 {
		requestBody = requestBody[:100*1024]
	}
	requestHeaders := make(map[string][]string, len(r.Header))
	for key, values := range r.Header {
		requestHeaders[key] = append([]string(nil), values...)
	}
	requestRecord := models.WebhookRequest{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Method:    r.Method, Path: r.URL.RequestURI(), Headers: requestHeaders, Body: string(requestBody),
	}
	if err := h.store.RecordWebhookRequest(r.Context(), requestRecord); err != nil {
		slog.Error("failed to record webhook HTTP request", "error", err)
	}

	authHeader := r.Header.Get("Authorization")
	if !models.RegexFaceitID.MatchString(authHeader) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Invalid Authorization Header"))
		return
	}

	body, err := parseBodyMap(r)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Invalid Request oder match is already closed"))
		return
	}

	// Webhook audit captures EVERY parsed webhook with valid header UUID BEFORE game mode/event validation
	username := h.resolvePlayerUsername(r.Context(), authHeader, body)
	if err := h.store.RecordWebhook(r.Context(), authHeader, username, body); err != nil {
		slog.Error("failed to record webhook audit", "error", err, "player", authHeader)
		h.databaseError(w, r, err)
		return
	}

	entityType := getNestedString(body, "payload", "entity", "type")
	if entityType != "matchmaking" && entityType != "hub" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Invalid Gamemode"))
		return
	}

	event := getNestedString(body, "event")
	if event == "match_status_configuring" {
		payloadID := getNestedString(body, "payload", "id")
		matchDetails, err := h.faceitClient.GetMatchDetails(r.Context(), payloadID)
		if err == nil && len(matchDetails.Voting.Map.Pick) > 0 {
			body["map"] = matchDetails.Voting.Map.Pick[0]
		} else {
			body["map"] = "unknown"
		}

		// Calculate matchup average ELO for CS2
		h.calculateMatchup(r.Context(), body)
		body["streamer"] = authHeader

		// Persist live match - AddLiveMatch safely clones document and avoids mutating body with meta or _id
		if err := h.store.AddLiveMatch(r.Context(), authHeader, body); err != nil {
			if errors.Is(err, store.ErrAlreadyLive) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("There is already a running game for streamer " + authHeader))
				return
			}
			slog.Error("failed to add live match to store", "error", err)
			h.databaseError(w, r, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
		return
	} else if event != "match_object_created" {
		if err := h.store.CloseLiveMatch(r.Context(), authHeader); err != nil {
			h.databaseError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Close match"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Invalid Request oder match is already closed"))
}

func (h *LegacyHandler) resolvePlayerUsername(ctx context.Context, playerID string, body map[string]interface{}) string {
	// 1. Find roster player id == header ID nickname in payload
	if payload, ok := body["payload"].(map[string]interface{}); ok {
		if teams, ok := payload["teams"].([]interface{}); ok {
			for _, teamItem := range teams {
				if teamMap, ok := teamItem.(map[string]interface{}); ok {
					if roster, ok := teamMap["roster"].([]interface{}); ok {
						for _, pItem := range roster {
							if pMap, ok := pItem.(map[string]interface{}); ok {
								id, _ := pMap["id"].(string)
								if id == "" {
									id, _ = pMap["player_id"].(string)
								}
								if id == playerID {
									if nick, ok := pMap["nickname"].(string); ok && nick != "" {
										return nick
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// 2. Best-effort cached known username in store
	if p, err := h.store.GetWebhookPlayer(ctx, playerID); err == nil && p != nil && p.Username != "" && p.Username != playerID {
		return p.Username
	}

	// 3. Fallback: Context bounded FACEIT player lookup
	if h.faceitClient != nil {
		lookupCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		if fp, err := h.faceitClient.GetPlayerByID(lookupCtx, playerID); err == nil && fp != nil && fp.Nickname != "" {
			return fp.Nickname
		}
	}

	// 4. Fallback: Configured streamer name
	if s, err := h.store.GetStreamer(ctx, playerID); err == nil && s != nil && s.Name != "" {
		return s.Name
	}

	// 5. Fallback: player ID
	return playerID
}

func (h *LegacyHandler) databaseError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("database operation failed", "requestId", r.Header.Get("X-Request-Id"), "path", r.URL.Path, "error", err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"message":"Database unavailable"}`))
}

// Express redirects negotiate plain text first, then HTML, and otherwise an empty body.
func writeLegacyRedirect(w http.ResponseWriter, r *http.Request, target string) {
	accept := r.Header.Get("Accept")
	if accept == "" {
		accept = "*/*"
	}
	quality := func(candidate string) float64 {
		bestSpecificity, result := -1, 0.0
		for _, item := range strings.Split(accept, ",") {
			parts := strings.Split(strings.TrimSpace(item), ";")
			media := strings.ToLower(strings.TrimSpace(parts[0]))
			specificity := -1
			switch media {
			case candidate:
				specificity = 2
			case "text/*":
				specificity = 1
			case "*/*":
				specificity = 0
			}
			q := 1.0
			for _, param := range parts[1:] {
				if strings.HasPrefix(strings.TrimSpace(param), "q=") {
					q, _ = strconv.ParseFloat(strings.TrimPrefix(strings.TrimSpace(param), "q="), 64)
				}
			}
			if specificity > bestSpecificity {
				bestSpecificity, result = specificity, q
			}
		}
		return result
	}
	plain, markup := quality("text/plain"), quality("text/html")
	body := ""
	if plain > 0 && plain >= markup {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		body = "Found. Redirecting to " + target
	} else if markup > 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		escaped := html.EscapeString(target)
		body = `<p>Found. Redirecting to <a href="` + escaped + `">` + escaped + `</a></p>`
	}
	w.Header().Set("Location", target)
	w.Header().Add("Vary", "Accept")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusFound)
	_, _ = w.Write([]byte(body))
}

// HandleGetMultiplePlayers handles POST /internal/api/getMultiplePlayers.
// Retains all input fields and returns application/json; charset=utf-8.
func (h *LegacyHandler) HandleGetMultiplePlayers(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"invalid request body"}`))
		return
	}
	defer r.Body.Close()

	playersList := make([]map[string]interface{}, 0)

	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		nested, err := ParseNestedForm(string(bodyBytes))
		if err == nil {
			if pl, ok := nested["players"].([]interface{}); ok {
				for _, item := range pl {
					if m, ok := item.(map[string]interface{}); ok {
						playersList = append(playersList, m)
					}
				}
			}
		}
	} else {
		// Strict single JSON document
		dec := json.NewDecoder(bytes.NewReader(bodyBytes))
		var jsonRoot interface{}
		if err := dec.Decode(&jsonRoot); err == nil {
			if rootMap, ok := jsonRoot.(map[string]interface{}); ok {
				if pl, ok := rootMap["players"].([]interface{}); ok {
					for _, item := range pl {
						if m, ok := item.(map[string]interface{}); ok {
							playersList = append(playersList, m)
						}
					}
				}
			} else if rootSlice, ok := jsonRoot.([]interface{}); ok {
				for _, item := range rootSlice {
					if m, ok := item.(map[string]interface{}); ok {
						playersList = append(playersList, m)
					}
				}
			}
		}
	}

	var wg sync.WaitGroup
	result := make([]map[string]interface{}, len(playersList))
	for i, p := range playersList {
		// Clone player map to retain all custom fields
		cloned := make(map[string]interface{}, len(p)+1)
		for k, v := range p {
			cloned[k] = v
		}
		result[i] = cloned
	}

	for i := range result {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			p := result[idx]

			steam64ID, _ := p["steam64ID"].(string)
			nickname, _ := p["nickname"].(string)

			var faceitPlayer *models.FaceitPlayer
			var err error

			if steam64ID != "" {
				faceitPlayer, err = h.faceitClient.GetPlayerByID(r.Context(), steam64ID)
			}
			if err != nil || faceitPlayer == nil {
				if nickname != "" {
					faceitPlayer, err = h.faceitClient.GetPlayerByNickname(r.Context(), nickname)
				}
			}

			if err == nil && faceitPlayer != nil && faceitPlayer.Games.CSGO != nil {
				result[idx]["elo"] = faceitPlayer.Games.CSGO.FaceitElo
			} else {
				result[idx]["elo"] = 0
			}
		}(i)
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// HandleTest handles GET/POST /test.
func (h *LegacyHandler) HandleTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"message":"Hello World!"}`))
}

// HandleNotFound handles 404 catch-all.
func (h *LegacyHandler) HandleNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"message":"not found"}`))
}

func (h *LegacyHandler) calculateMatchup(ctx context.Context, body map[string]interface{}) {
	payload, ok := body["payload"].(map[string]interface{})
	if !ok {
		return
	}
	teams, ok := payload["teams"].([]interface{})
	if !ok || len(teams) < 2 {
		return
	}

	team1Map, _ := teams[0].(map[string]interface{})
	team2Map, _ := teams[1].(map[string]interface{})

	team1Name, _ := team1Map["name"].(string)
	team2Name, _ := team2Map["name"].(string)

	avg1 := h.calcTeamAvgElo(ctx, team1Map)
	avg2 := h.calcTeamAvgElo(ctx, team2Map)

	body["matchup"] = map[string]interface{}{
		"team1": map[string]interface{}{
			"name":   team1Name,
			"avgElo": avg1,
		},
		"team2": map[string]interface{}{
			"name":   team2Name,
			"avgElo": avg2,
		},
	}
}

func (h *LegacyHandler) calcTeamAvgElo(ctx context.Context, teamMap map[string]interface{}) int {
	roster, ok := teamMap["roster"].([]interface{})
	if !ok || len(roster) == 0 {
		return 0
	}

	var elos []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	limit := len(roster)
	if limit > 5 {
		limit = 5
	}

	for i := 0; i < limit; i++ {
		playerMap, ok := roster[i].(map[string]interface{})
		if !ok {
			continue
		}
		nickname, _ := playerMap["nickname"].(string)
		if nickname == "" {
			continue
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			p, err := h.faceitClient.GetPlayerByNickname(ctx, name)
			if err == nil && p != nil && p.Games.CS2 != nil {
				mu.Lock()
				elos = append(elos, p.Games.CS2.FaceitElo)
				mu.Unlock()
			}
		}(nickname)
	}
	wg.Wait()

	if len(elos) == 0 {
		return 0
	}
	sum := 0
	for _, elo := range elos {
		sum += elo
	}
	return sum / 5
}

func (h *LegacyHandler) asyncSavePlayer(insertType string, player *models.FaceitPlayer, respStr string) {
	playerData := map[string]interface{}{
		"player_id":  player.PlayerID,
		"nickname":   player.Nickname,
		"avatar":     player.Avatar,
		"country":    player.Country,
		"coverImage": player.CoverImage,
		"platforms":  player.Platforms,
		"games":      player.Games,
		"local": map[string]interface{}{
			"username":       player.Nickname,
			"responseString": respStr,
		},
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.store.SavePlayer(ctx, insertType, playerData); err != nil {
			slog.Error("player audit save failed", "playerId", player.PlayerID, "insertType", insertType, "error", err)
		}
	}()
}

func formatEloResponse(player *models.FaceitPlayer) string {
	csgoElo := 0
	if player.Games.CSGO != nil {
		csgoElo = player.Games.CSGO.FaceitElo
	}

	if player.Games.CS2 != nil {
		return fmt.Sprintf("%s's Elo:  %d (csgo)  / %d (cs2)", player.Nickname, csgoElo, player.Games.CS2.FaceitElo)
	}

	return fmt.Sprintf("%s's Elo:  %d (csgo) ", player.Nickname, csgoElo)
}

func parseBodyMap(r *http.Request) (map[string]interface{}, error) {
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 100*1024+1))
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if len(bodyBytes) > 100*1024 {
		return nil, fmt.Errorf("body exceeds 100kb limit")
	}

	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		return ParseNestedForm(string(bodyBytes))
	}

	if len(bodyBytes) == 0 {
		return map[string]interface{}{}, nil
	}
	var result map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(bodyBytes))
	if err := dec.Decode(&result); err != nil {
		return nil, err
	}
	var extra interface{}
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("single JSON document required")
	}
	return result, nil
}

func getNestedString(m map[string]interface{}, keys ...string) string {
	val := getNestedValue(m, keys...)
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

func getNestedValue(m map[string]interface{}, keys ...string) interface{} {
	var current interface{} = m
	for _, key := range keys {
		currMap, ok := current.(map[string]interface{})
		if !ok || currMap == nil {
			return nil
		}
		current = currMap[key]
	}
	return current
}

// Express converts repeated query values to comma-separated strings.
func legacyQueryValue(r *http.Request, key string) string {
	values, ok := r.URL.Query()[key]
	if !ok {
		values = r.URL.Query()[key+"[]"]
	}
	return strings.Join(values, ",")
}

// Legacy validation combines JavaScript Number() and parseInt(), allowing decimals.
func legacyLimit(raw string) (int, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return 0, fmt.Errorf("invalid limit")
	}
	_, err := strconv.ParseFloat(text, 64)
	if err != nil && !strings.HasPrefix(text, "0x") && !strings.HasPrefix(text, "0X") {
		return 0, err
	}
	prefix := text
	sign := ""
	if strings.HasPrefix(prefix, "+") || strings.HasPrefix(prefix, "-") {
		sign = prefix[:1]
		prefix = prefix[1:]
	}
	if strings.HasPrefix(prefix, "0x") || strings.HasPrefix(prefix, "0X") {
		n, e := strconv.ParseInt(sign+prefix[2:], 16, 64)
		return int(n), e
	}
	i := 0
	for i < len(prefix) && prefix[i] >= '0' && prefix[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("invalid limit")
	}
	return strconv.Atoi(sign + prefix[:i])
}
