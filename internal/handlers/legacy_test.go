package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bonbonn1912/bonbonn-game-tracker/internal/config"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/faceit"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/models"
	"github.com/bonbonn1912/bonbonn-game-tracker/internal/store"
)

func setupTestServer(t *testing.T) (*Server, *store.MemoryStore, *httptest.Server) {
	// Fake Faceit Server
	faceitServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasPrefix(path, "/players"):
			nickname := r.URL.Query().Get("nickname")
			gamePlayerID := r.URL.Query().Get("game_player_id")

			if strings.Contains(path, "/history") {
				// Match history endpoint
				resp := models.MatchHistoryResponse{
					Items: []models.MatchHistoryItem{
						{
							MatchID: "m1",
							Results: models.MatchHistoryResult{Winner: "faction1"},
							Teams: models.MatchHistoryTeamsMap{
								Faction1: models.MatchHistoryFaction{
									Players: []models.MatchHistoryPlayer{{PlayerID: "test-player-id", Nickname: "testuser"}},
								},
								Faction2: models.MatchHistoryFaction{
									Players: []models.MatchHistoryPlayer{{PlayerID: "other-id", Nickname: "other"}},
								},
							},
						},
						{
							MatchID: "m2",
							Results: models.MatchHistoryResult{Winner: "faction2"},
							Teams: models.MatchHistoryTeamsMap{
								Faction1: models.MatchHistoryFaction{
									Players: []models.MatchHistoryPlayer{{PlayerID: "test-player-id", Nickname: "testuser"}},
								},
								Faction2: models.MatchHistoryFaction{
									Players: []models.MatchHistoryPlayer{{PlayerID: "other-id", Nickname: "other"}},
								},
							},
						},
						{
							MatchID: "m3",
							Results: models.MatchHistoryResult{Winner: "faction1"},
							Teams: models.MatchHistoryTeamsMap{
								Faction1: models.MatchHistoryFaction{
									Players: []models.MatchHistoryPlayer{{PlayerID: "test-player-id", Nickname: "testuser"}},
								},
								Faction2: models.MatchHistoryFaction{
									Players: []models.MatchHistoryPlayer{{PlayerID: "other-id", Nickname: "other"}},
								},
							},
						},
						{
							MatchID: "m4",
							Results: models.MatchHistoryResult{Winner: "faction1"},
							Teams: models.MatchHistoryTeamsMap{
								Faction1: models.MatchHistoryFaction{
									Players: []models.MatchHistoryPlayer{{PlayerID: "test-player-id", Nickname: "testuser"}},
								},
								Faction2: models.MatchHistoryFaction{
									Players: []models.MatchHistoryPlayer{{PlayerID: "other-id", Nickname: "other"}},
								},
							},
						},
					},
				}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}

			if nickname == "unknown" {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message":"not found"}`))
				return
			}

			cs2Elo := 2150
			player := models.FaceitPlayer{
				PlayerID: "test-player-id",
				Nickname: "testuser",
				Games: models.FaceitGames{
					CSGO: &models.FaceitGameDetail{
						FaceitElo: 2100,
					},
					CS2: &models.FaceitGameDetail{
						FaceitElo: cs2Elo,
					},
				},
			}
			if nickname == "csgoonly" || gamePlayerID == "76561198000000002" {
				player.Games.CS2 = nil
				player.Nickname = "csgoonly"
			}
			_ = json.NewEncoder(w).Encode(player)
			return

		case strings.HasPrefix(path, "/matches/"):
			var match models.MatchDetails
			match.Voting.Map.Pick = []string{"de_mirage"}
			_ = json.NewEncoder(w).Encode(match)
			return

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	memStore := store.NewMemoryStore()
	cfg := &config.Config{
		Port:          "12000",
		Env:           "test",
		AdminUsername: "admin",
		AdminPassword: "secretpassword",
		AdminHost:     "admin.faceitlobby.com",
		FaceitBaseURL: faceitServer.URL,
	}

	fc := faceit.NewClient(faceitServer.URL, "fake-api-key", nil)
	srv, err := NewServer(cfg, memStore, fc)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	return srv, memStore, faceitServer
}

func TestLegacyEndpoints(t *testing.T) {
	srv, memStore, faceitServer := setupTestServer(t)
	defer faceitServer.Close()

	t.Run("GET /test and POST /test", func(t *testing.T) {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			req := httptest.NewRequest(method, "/test", nil)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("%s /test expected 200, got %d", method, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), `{"message":"Hello World!"}`) {
				t.Errorf("unexpected body: %s", rec.Body.String())
			}
		}
	})

	t.Run("404 catch-all", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `{"message":"not found"}`) {
			t.Errorf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("CORS preflight and methods", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/elo", nil)
		req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected 204 No Content for OPTIONS, got %d", rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("expected Allow-Origin: *, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
		}
		if rec.Header().Get("Access-Control-Allow-Methods") != "GET,HEAD,PUT,PATCH,POST,DELETE" {
			t.Errorf("unexpected Allow-Methods: %s", rec.Header().Get("Access-Control-Allow-Methods"))
		}
		if rec.Header().Get("Access-Control-Allow-Headers") != "Content-Type, Authorization" {
			t.Errorf("unexpected Allow-Headers: %s", rec.Header().Get("Access-Control-Allow-Headers"))
		}
	})

	t.Run("GET /elo and /checkelo case-insensitivity and trailing slash", func(t *testing.T) {
		cases := []string{
			"/elo?username=testuser",
			"/ELO?username=testuser",
			"/elo/?username=testuser",
			"/checkelo?username=testuser",
			"/CheckElo/?username=testuser",
		}
		for _, url := range cases {
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("%s expected 200, got %d", url, rec.Code)
			}
			if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
				t.Errorf("expected Content-Type text/html; charset=utf-8, got %s", rec.Header().Get("Content-Type"))
			}
			expected := "testuser's Elo:  2100 (csgo)  / 2150 (cs2)"
			if rec.Body.String() != expected {
				t.Errorf("got %q, expected %q", rec.Body.String(), expected)
			}
		}
	})

	t.Run("GET /elo csgo only player", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/elo?username=csgoonly", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		expected := "csgoonly's Elo:  2100 (csgo) "
		if rec.Body.String() != expected {
			t.Errorf("got %q, expected %q", rec.Body.String(), expected)
		}
	})

	t.Run("GET /elo validation and not found", func(t *testing.T) {
		// Invalid regex (too short)
		req := httptest.NewRequest(http.MethodGet, "/elo?username=ab", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "Nicknames can be made of") {
			t.Errorf("unexpected response: %s", rec.Body.String())
		}

		// Unknown player in Faceit
		req = httptest.NewRequest(http.MethodGet, "/elo?username=unknown", nil)
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Body.String() != "Invalid username" {
			t.Errorf("expected 'Invalid username', got %q", rec.Body.String())
		}
	})

	t.Run("GET /elobyid", func(t *testing.T) {
		// Valid Faceit ID
		req := httptest.NewRequest(http.MethodGet, "/elobyid?id=4967963b-566f-4e8a-908b-02c8315d12f1", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		expected := "testuser's Elo:  2100 (csgo)  / 2150 (cs2)"
		if rec.Body.String() != expected {
			t.Errorf("got %q, expected %q", rec.Body.String(), expected)
		}

		// Invalid Faceit ID
		req = httptest.NewRequest(http.MethodGet, "/elobyid?id=invalid-id", nil)
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "Faceit Ids must match following pattern") {
			t.Errorf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("GET /elobysteamid", func(t *testing.T) {
		// Valid 17-digit Steam ID
		req := httptest.NewRequest(http.MethodGet, "/elobysteamid?id=76561198000000001", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}

		// Invalid Steam ID
		req = httptest.NewRequest(http.MethodGet, "/elobysteamid?id=123", nil)
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
		if rec.Body.String() != "Steam 64 Ids are made of 17 numbers (0-9)" {
			t.Errorf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("GET /matchhistory", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/matchhistory?username=testuser&limit=4", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		expected := "testuser's last 4 Games: W/L/W/W. Winrate: 75%"
		if rec.Body.String() != expected {
			t.Errorf("got %q, expected %q", rec.Body.String(), expected)
		}

		// Invalid limit (< 4)
		req = httptest.NewRequest(http.MethodGet, "/matchhistory?username=testuser&limit=3", nil)
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 for limit 3, got %d", rec.Code)
		}
		if rec.Body.String() != "Incorrect faceit username or invalid limit (4-100)" {
			t.Errorf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("GET /getmatch and GET /redirect", func(t *testing.T) {
		streamerID := "4967963b-566f-4e8a-908b-02c8315d12f1"

		// Not in matchroom initially
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/getmatch?key=%s", streamerID), nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Body.String() != "This User is currently not in a matchroom" {
			t.Errorf("unexpected: %s", rec.Body.String())
		}

		// Redirect when not ingame
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/redirect?key=%s", streamerID), nil)
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "is currently not ingame") {
			t.Errorf("unexpected redirect body: %s", rec.Body.String())
		}

		// Now add active match to store
		matchData := map[string]interface{}{
			"payload": map[string]interface{}{
				"id": "match-uuid-1234",
			},
			"map": "de_mirage",
			"matchup": map[string]interface{}{
				"team1": map[string]interface{}{"name": "Team A", "avgElo": 2000},
				"team2": map[string]interface{}{"name": "Team B", "avgElo": 2050},
			},
		}
		if err := memStore.AddLiveMatch(context.Background(), streamerID, matchData); err != nil {
			t.Fatalf("failed to add live match: %v", err)
		}

		// Now test /getmatch with live game and configured streamer URL
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/getmatch?key=%s", streamerID), nil)
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		expectedGetMatch := "Team A (2000) vs. Team B (2050) on de_mirage. Link zum Matchroom: http://edox.faceitlobby.com"
		if rec.Body.String() != expectedGetMatch {
			t.Errorf("got %q, expected %q", rec.Body.String(), expectedGetMatch)
		}

		// Test /redirect with live game
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/redirect?key=%s", streamerID), nil)
		rec = httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusFound {
			t.Errorf("expected 302 Found, got %d", rec.Code)
		}
		if rec.Header().Get("Location") != "https://www.faceit.com/en/csgo/room/match-uuid-1234" {
			t.Errorf("unexpected redirect location: %s", rec.Header().Get("Location"))
		}

		// Verify redirect event was recorded
		events, err := memStore.GetRedirectEvents(context.Background(), 10)
		if err != nil || len(events) == 0 {
			t.Fatalf("expected logged events, got %v", events)
		}
		if events[0].Outcome != "redirected" || events[0].MatchID != "match-uuid-1234" {
			t.Errorf("unexpected event: %+v", events[0])
		}
	})

	t.Run("POST /match configuring, duplicate rejection, and close", func(t *testing.T) {
		streamerID := "e3cefc97-36b1-4615-91d6-cdcca13e4e58"

		matchWebhook := map[string]interface{}{
			"event": "match_status_configuring",
			"payload": map[string]interface{}{
				"id": "match-uuid-configuring",
				"entity": map[string]interface{}{
					"type": "matchmaking",
				},
				"teams": []interface{}{
					map[string]interface{}{
						"name": "Team 1",
						"roster": []interface{}{
							map[string]interface{}{"nickname": "testuser"},
						},
					},
					map[string]interface{}{
						"name": "Team 2",
						"roster": []interface{}{
							map[string]interface{}{"nickname": "testuser"},
						},
					},
				},
			},
		}
		bodyBytes, _ := json.Marshal(matchWebhook)

		req := httptest.NewRequest(http.MethodPost, "/match", bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", streamerID)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var respMap map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &respMap)
		if respMap["map"] != "de_mirage" || respMap["streamer"] != streamerID {
			t.Errorf("unexpected webhook response: %+v", respMap)
		}
		// Confirm meta or _id is NOT in the public response!
		if _, hasMeta := respMap["meta"]; hasMeta {
			t.Errorf("public match response must not contain internal 'meta'")
		}
		if _, hasID := respMap["_id"]; hasID {
			t.Errorf("public match response must not contain internal '_id'")
		}

		// Try configuring duplicate game for same streamer -> should be rejected with 404
		reqDup := httptest.NewRequest(http.MethodPost, "/match", bytes.NewReader(bodyBytes))
		reqDup.Header.Set("Authorization", streamerID)
		reqDup.Header.Set("Content-Type", "application/json")
		recDup := httptest.NewRecorder()
		srv.ServeHTTP(recDup, reqDup)

		if recDup.Code != http.StatusNotFound {
			t.Errorf("expected 404 duplicate rejection, got %d", recDup.Code)
		}
		expectedErr := "There is already a running game for streamer " + streamerID
		if recDup.Body.String() != expectedErr {
			t.Errorf("got %q, expected %q", recDup.Body.String(), expectedErr)
		}

		// Close match with event != match_object_created
		closeWebhook := map[string]interface{}{
			"event": "match_status_finished",
			"payload": map[string]interface{}{
				"entity": map[string]interface{}{"type": "matchmaking"},
			},
		}
		closeBytes, _ := json.Marshal(closeWebhook)
		reqClose := httptest.NewRequest(http.MethodPost, "/match", bytes.NewReader(closeBytes))
		reqClose.Header.Set("Authorization", streamerID)
		reqClose.Header.Set("Content-Type", "application/json")
		recClose := httptest.NewRecorder()
		srv.ServeHTTP(recClose, reqClose)

		if recClose.Code != http.StatusOK || recClose.Body.String() != "Close match" {
			t.Errorf("expected 'Close match', got %s", recClose.Body.String())
		}

		// Confirm game is now closed in store
		_, live, _ := memStore.GetLiveMatch(context.Background(), streamerID)
		if live {
			t.Errorf("match should be closed")
		}
	})

	t.Run("POST /internal/api/getMultiplePlayers", func(t *testing.T) {
		input := map[string]interface{}{
			"players": []map[string]interface{}{
				{"nickname": "testuser", "steam64ID": "76561198000000001"},
			},
		}
		b, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/internal/api/getMultiplePlayers", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var output []models.ConsolePlayerInput
		if err := json.Unmarshal(rec.Body.Bytes(), &output); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if len(output) != 1 || output[0].Elo != 2100 {
			t.Errorf("unexpected output: %+v", output)
		}
	})
}
