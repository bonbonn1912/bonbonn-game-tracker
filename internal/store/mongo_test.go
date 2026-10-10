package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bonbonn1912/bonbonn-game-tracker/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoStoreIntegration(t *testing.T) {
	mongoURI := os.Getenv("MONGO_TEST_URI")
	if mongoURI == "" {
		mongoURI = os.Getenv("TEST_MONGO_URI")
	}
	if mongoURI == "" && os.Getenv("TEST_MONGO") == "true" {
		mongoURI = "mongodb://127.0.0.1:27017"
	}
	if mongoURI == "" {
		t.Skip("skipping MongoDB integration test; set MONGO_TEST_URI to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	testDBName := "bonbonn_test_" + time.Now().Format("20060102150405")
	ms, err := NewMongoStore(ctx, mongoURI, testDBName, "test_players", "test_matchrooms")
	if err != nil {
		t.Fatalf("failed to connect to test mongodb: %v", err)
	}
	defer func() {
		_ = ms.db.Drop(context.Background())
		_ = ms.Close(context.Background())
	}()

	t.Run("Seed streamers on clean database", func(t *testing.T) {
		streamers, err := ms.GetStreamers(ctx)
		if err != nil {
			t.Fatalf("failed to get streamers: %v", err)
		}
		if len(streamers) != len(DefaultSeedStreamers) {
			t.Errorf("expected %d seed streamers, got %d", len(DefaultSeedStreamers), len(streamers))
		}
	})

	t.Run("Seed idempotency after edits", func(t *testing.T) {
		// Modify a streamer
		seedID := DefaultSeedStreamers[0].ID
		_, err := ms.UpdateStreamer(ctx, seedID, models.Streamer{
			Name:    "CustomModifiedName",
			URL:     "https://custom.faceitlobby.com",
			Enabled: false,
		})
		if err != nil {
			t.Fatalf("failed to update streamer: %v", err)
		}

		// Re-run seedStreamers
		err = ms.seedStreamers(ctx)
		if err != nil {
			t.Fatalf("failed to run second seed: %v", err)
		}

		// Verify custom name was NOT overwritten by seed
		s, err := ms.GetStreamer(ctx, seedID)
		if err != nil || s.Name != "CustomModifiedName" {
			t.Errorf("seed migration overwrote modified streamer! Name: %v", s.Name)
		}
	})

	t.Run("Live match lifecycle, nested team avg/id roundtrip, strip _id and meta", func(t *testing.T) {
		streamerID := "e3cefc97-36b1-4615-91d6-cdcca13e4e58"

		matchDoc := map[string]interface{}{
			"payload": map[string]interface{}{
				"id": "mongo-test-match-1",
				"teams": []interface{}{
					map[string]interface{}{
						"name": "Team Alpha",
						"stats": map[string]interface{}{
							"avgElo": 2350,
							"rating": 1.25,
						},
					},
					map[string]interface{}{
						"name": "Team Beta",
						"stats": map[string]interface{}{
							"avgElo": 2280,
						},
					},
				},
			},
			"map":               "de_mirage",
			"unknownFieldEcho":  "preserved_custom_value",
			"nestedCustomField": map[string]interface{}{"subKey": "subVal"},
		}

		// Add live match
		err := ms.AddLiveMatch(ctx, streamerID, matchDoc)
		if err != nil {
			t.Fatalf("failed to add live match: %v", err)
		}

		// Verify original caller matchData wasn't mutated with meta or _id
		if _, hasMeta := matchDoc["meta"]; hasMeta {
			t.Errorf("caller matchData must not be mutated with meta")
		}
		if _, hasID := matchDoc["_id"]; hasID {
			t.Errorf("caller matchData must not be mutated with _id")
		}

		// Try adding duplicate live match for same streamer -> ErrAlreadyLive
		errDup := ms.AddLiveMatch(ctx, streamerID, matchDoc)
		if errDup != ErrAlreadyLive {
			t.Errorf("expected ErrAlreadyLive on duplicate, got %v", errDup)
		}

		// Get active match directly from DB
		doc, found, err := ms.GetLiveMatch(ctx, streamerID)
		if err != nil || !found {
			t.Fatalf("failed to get live match from db: %v", err)
		}

		// Verify public retrieval strips _id and meta
		if _, hasID := doc["_id"]; hasID {
			t.Errorf("public retrieval must strip _id from returned match document")
		}
		if _, hasMeta := doc["meta"]; hasMeta {
			t.Errorf("public retrieval must strip meta from returned match document")
		}

		// Verify custom unknown fields are preserved
		if doc["unknownFieldEcho"] != "preserved_custom_value" {
			t.Errorf("expected preserved_custom_value, got %v", doc["unknownFieldEcho"])
		}

		// Verify nested navigation on maps and slices
		nestedCustom, ok := doc["nestedCustomField"].(map[string]interface{})
		if !ok || nestedCustom["subKey"] != "subVal" {
			t.Errorf("nestedCustomField must be plain map[string]interface{}, got %T (%v)", doc["nestedCustomField"], doc["nestedCustomField"])
		}

		payload, ok := doc["payload"].(map[string]interface{})
		if !ok {
			t.Fatalf("payload must be plain map[string]interface{}, got %T", doc["payload"])
		}
		if payload["id"] != "mongo-test-match-1" {
			t.Errorf("expected match ID mongo-test-match-1, got %v", payload["id"])
		}

		teams, ok := payload["teams"].([]interface{})
		if !ok || len(teams) != 2 {
			t.Fatalf("teams must be plain []interface{}, got %T (%v)", payload["teams"], payload["teams"])
		}

		team0, ok := teams[0].(map[string]interface{})
		if !ok {
			t.Fatalf("team 0 must be plain map[string]interface{}, got %T", teams[0])
		}
		stats, ok := team0["stats"].(map[string]interface{})
		if !ok {
			t.Fatalf("team stats must be plain map[string]interface{}, got %T", team0["stats"])
		}
		if stats["avgElo"] == nil {
			t.Errorf("expected avgElo to be present in stats: %v", stats)
		}

		// Close live match
		err = ms.CloseLiveMatch(ctx, streamerID)
		if err != nil {
			t.Fatalf("failed to close live match: %v", err)
		}

		// Now GetLiveMatch should return found = false
		_, foundAfter, _ := ms.GetLiveMatch(ctx, streamerID)
		if foundAfter {
			t.Errorf("live match should not be found after closing")
		}
	})

	t.Run("Redirect events logging and retrieval", func(t *testing.T) {
		event := models.RedirectEvent{
			ID:           "test-event-uuid-1",
			StreamerID:   "streamer-1",
			StreamerName: "Streamer One",
			MatchID:      "match-1",
			Target:       "https://faceit.com/room/1",
			Outcome:      "redirected",
			CreatedAt:    time.Now().UTC().Format(time.RFC3339),
			RequestID:    "req-1",
			IP:           "127.0.0.1",
			UserAgent:    "Go-Test",
		}

		err := ms.LogRedirectEvent(ctx, event)
		if err != nil {
			t.Fatalf("failed to log redirect event: %v", err)
		}

		events, err := ms.GetRedirectEvents(ctx, 10)
		if err != nil || len(events) == 0 {
			t.Fatalf("failed to get redirect events: %v", err)
		}
		if events[0].ID != "test-event-uuid-1" {
			t.Errorf("unexpected event: %+v", events[0])
		}
	})

	t.Run("RecordWebhook real Mongo execution, path conflict prevention, distinct game count, enrichment", func(t *testing.T) {
		playerID := "player-uuid-1111"
		matchID1 := "match-uuid-aaaa"

		// 1. First webhook: configuring event
		body1 := map[string]interface{}{
			"event": "match_status_configuring",
			"payload": map[string]interface{}{
				"id": matchID1,
				"teams": []interface{}{
					map[string]interface{}{
						"name": "Team 1",
						"roster": []interface{}{
							map[string]interface{}{"id": playerID, "nickname": "ProGamer1"},
						},
					},
					map[string]interface{}{
						"name": "Team 2",
					},
				},
			},
			"map":     "de_mirage",
			"matchup": "Team 1 vs Team 2",
		}

		err := ms.RecordWebhook(ctx, playerID, "ProGamer1", body1)
		if err != nil {
			t.Fatalf("RecordWebhook failed on configuring event: %v", err)
		}

		// Verify player was inserted without path conflict
		p, err := ms.GetWebhookPlayer(ctx, playerID)
		if err != nil {
			t.Fatalf("failed to get webhook player: %v", err)
		}
		if p.Username != "ProGamer1" {
			t.Errorf("expected username 'ProGamer1', got %q", p.Username)
		}
		if p.GameCount != 1 {
			t.Errorf("expected gameCount 1, got %d", p.GameCount)
		}

		// 2. Repeated webhook for same match (e.g. retry or redundant configuring event)
		err = ms.RecordWebhook(ctx, playerID, "ProGamer1", body1)
		if err != nil {
			t.Fatalf("RecordWebhook failed on retry event: %v", err)
		}

		// Game count must NOT duplicate
		p, _ = ms.GetWebhookPlayer(ctx, playerID)
		if p.GameCount != 1 {
			t.Errorf("gameCount duplicated on retry! Expected 1, got %d", p.GameCount)
		}

		// 3. Sparse closing webhook arrives (no teams, no map/matchup)
		bodyClosing := map[string]interface{}{
			"event": "match_status_finished",
			"payload": map[string]interface{}{
				"id": matchID1,
			},
		}
		err = ms.RecordWebhook(ctx, playerID, "ProGamer1", bodyClosing)
		if err != nil {
			t.Fatalf("RecordWebhook failed on closing event: %v", err)
		}

		// Verify canonical match status is closed, but map/matchup and teams are preserved!
		matches, total, err := ms.GetPlayerMatches(ctx, playerID, 10, 0)
		if err != nil || total != 1 || len(matches) != 1 {
			t.Fatalf("expected 1 match, got total=%d len=%d, err=%v", total, len(matches), err)
		}
		m := matches[0]
		if m.Status != "closed" {
			t.Errorf("expected status 'closed', got %q", m.Status)
		}
		if m.Map != "de_mirage" {
			t.Errorf("expected preserved map 'de_mirage', got %v", m.Map)
		}
		if m.Matchup != "Team 1 vs Team 2" {
			t.Errorf("expected preserved matchup 'Team 1 vs Team 2', got %v", m.Matchup)
		}
		if !hasTeams(m.Payload) {
			t.Errorf("expected richer teams payload to be preserved after sparse closing event")
		}

		// 4. Delayed config retry arrives after match is closed -> MUST STAY CLOSED!
		err = ms.RecordWebhook(ctx, playerID, "ProGamer1", body1)
		if err != nil {
			t.Fatalf("RecordWebhook failed on delayed config retry: %v", err)
		}
		matches, _, _ = ms.GetPlayerMatches(ctx, playerID, 10, 0)
		if matches[0].Status != "closed" {
			t.Errorf("delayed config retry reset closed match to %q! Must stay closed", matches[0].Status)
		}

		// 5. AddLiveMatch enrichment on an already closed canonical match must not reset it to live
		err = ms.AddLiveMatch(ctx, playerID, body1)
		// Clean up live match if added
		_ = ms.CloseLiveMatch(ctx, playerID)
		matches, _, _ = ms.GetPlayerMatches(ctx, playerID, 10, 0)
		if matches[0].Status != "closed" {
			t.Errorf("AddLiveMatch enrichment reset closed canonical match to %q! Must stay closed", matches[0].Status)
		}
	})

	t.Run("Close-only player with no matchId is recorded with gameCount 0", func(t *testing.T) {
		closeOnlyPlayerID := "close-only-player-9999"
		bodyCloseOnly := map[string]interface{}{
			"event": "match_status_finished",
			// No payload.id
			"payload": map[string]interface{}{},
		}

		err := ms.RecordWebhook(ctx, closeOnlyPlayerID, closeOnlyPlayerID, bodyCloseOnly)
		if err != nil {
			t.Fatalf("RecordWebhook failed for close-only player: %v", err)
		}

		p, err := ms.GetWebhookPlayer(ctx, closeOnlyPlayerID)
		if err != nil {
			t.Fatalf("failed to get close-only player: %v", err)
		}
		if p.ID != closeOnlyPlayerID {
			t.Errorf("expected ID %s, got %s", closeOnlyPlayerID, p.ID)
		}
		if p.GameCount != 0 {
			t.Errorf("expected gameCount 0 for close-only player, got %d", p.GameCount)
		}

		// Player matches should be empty
		matches, total, err := ms.GetPlayerMatches(ctx, closeOnlyPlayerID, 50, 0)
		if err != nil || total != 0 || len(matches) != 0 {
			t.Errorf("expected 0 matches for close-only player, got total=%d len=%d", total, len(matches))
		}
	})

	t.Run("Pagination on GetPlayerMatches", func(t *testing.T) {
		pID := "pagination-test-player"
		// Insert 5 matches
		for i := 1; i <= 5; i++ {
			body := map[string]interface{}{
				"event": "match_status_configuring",
				"payload": map[string]interface{}{
					"id": fmt.Sprintf("paginated-match-%d", i),
				},
				"map": fmt.Sprintf("de_map_%d", i),
			}
			err := ms.RecordWebhook(ctx, pID, "PaginationTester", body)
			if err != nil {
				t.Fatalf("failed to record match %d: %v", i, err)
			}
			// Slight delay to ensure distinct timestamps
			time.Sleep(2 * time.Millisecond)
		}

		// Page 1: limit 2, offset 0
		page1, total, err := ms.GetPlayerMatches(ctx, pID, 2, 0)
		if err != nil || total != 5 || len(page1) != 2 {
			t.Fatalf("page 1 failed: total=%d len=%d err=%v", total, len(page1), err)
		}

		// Page 2: limit 2, offset 2
		page2, total, err := ms.GetPlayerMatches(ctx, pID, 2, 2)
		if err != nil || total != 5 || len(page2) != 2 {
			t.Fatalf("page 2 failed: total=%d len=%d err=%v", total, len(page2), err)
		}
		if page1[0].ID == page2[0].ID || page1[1].ID == page2[0].ID {
			t.Errorf("page 1 and page 2 overlap! page1: %+v, page2: %+v", page1, page2)
		}

		// Page 3: limit 2, offset 4
		page3, total, err := ms.GetPlayerMatches(ctx, pID, 2, 4)
		if err != nil || total != 5 || len(page3) != 1 {
			t.Fatalf("page 3 failed: total=%d len=%d err=%v", total, len(page3), err)
		}

		// Offset beyond total: offset 10
		pageEmpty, total, err := ms.GetPlayerMatches(ctx, pID, 2, 10)
		if err != nil || total != 5 || len(pageEmpty) != 0 {
			t.Fatalf("offset beyond total failed: total=%d len=%d", total, len(pageEmpty))
		}
	})

	t.Run("Backfill migration with nested bson.M parses matchID correctly", func(t *testing.T) {
		streamerID := "backfill-test-streamer"
		matchID := "backfill-match-id-xyz"

		// Insert directly into matchroomCol a document with nested bson.M structure
		rawMatchroom := bson.M{
			"streamer": streamerID,
			"payload": bson.M{
				"id": matchID,
				"teams": bson.A{
					bson.M{"name": "Team 1"},
				},
			},
			"meta": bson.M{
				"isRunning":   false,
				"inserted_at": "2026-01-01T00:00:00Z",
				"timestamp":   "2026-01-01T00:30:00Z",
			},
			"map":     "de_nuke",
			"matchup": "Team 1 vs Team 2",
		}
		_, err := ms.matchroomCol.InsertOne(ctx, rawMatchroom)
		if err != nil {
			t.Fatalf("failed inserting raw matchroom: %v", err)
		}

		// Delete migration marker so backfill runs again
		_, _ = ms.migrations.DeleteOne(ctx, bson.M{"_id": backfillMigrationID})

		// Run backfill
		err = ms.backfillMatchrooms(ctx)
		if err != nil {
			t.Fatalf("backfillMatchrooms failed: %v", err)
		}

		// Verify player was created and matchID was parsed (NOT empty)
		player, err := ms.GetWebhookPlayer(ctx, streamerID)
		if err != nil {
			t.Fatalf("failed to find backfilled player: %v", err)
		}
		if player.GameCount != 1 {
			t.Errorf("expected backfilled player gameCount 1, got %d", player.GameCount)
		}

		matches, total, err := ms.GetPlayerMatches(ctx, streamerID, 10, 0)
		if err != nil || total != 1 || len(matches) != 1 {
			t.Fatalf("expected 1 backfilled match, got total=%d len=%d err=%v", total, len(matches), err)
		}
		if matches[0].MatchID != matchID {
			t.Errorf("backfill failed to extract matchID from nested bson.M! Got %q, want %q", matches[0].MatchID, matchID)
		}
		if matches[0].Status != "closed" {
			t.Errorf("expected status 'closed' for non-running matchroom, got %q", matches[0].Status)
		}
	})

	t.Run("Reconnection and Ping error handling", func(t *testing.T) {
		// ms is currently connected
		if err := ms.Ping(ctx); err != nil {
			t.Errorf("expected Ping to succeed on active store, got %v", err)
		}

		// Create a separate store to test Disconnect
		tempDB := "bonbonn_ping_test_" + time.Now().Format("20060102150405")
		tempStore, err := NewMongoStore(ctx, mongoURI, tempDB, "p", "m")
		if err != nil {
			t.Fatalf("failed creating temp store: %v", err)
		}
		defer func() {
			_ = tempStore.db.Drop(context.Background())
		}()

		// Close it
		_ = tempStore.Close(ctx)

		// Now ping should fail
		pingErr := tempStore.Ping(ctx)
		if pingErr == nil {
			t.Errorf("expected Ping to return error after Close, got nil")
		}
	})
}
