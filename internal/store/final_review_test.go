package store

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func exerciseWebhookHistory(t *testing.T, s Store) {
	t.Helper()
	ctx := context.Background()
	player := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	rich := map[string]interface{}{"event": "match_status_configuring", "payload": map[string]interface{}{"id": "history-review-match", "teams": []interface{}{map[string]interface{}{"name": "Alpha"}, map[string]interface{}{"name": "Beta"}}}}
	if err := s.RecordWebhook(ctx, player, "KnownNickname", rich); err != nil {
		t.Fatal(err)
	}
	rich["map"] = "de_mirage"
	rich["matchup"] = map[string]interface{}{"team1": map[string]interface{}{"name": "Alpha", "avgElo": 1600}, "team2": map[string]interface{}{"name": "Beta", "avgElo": 1700}}
	if err := s.AddLiveMatch(ctx, player, rich); err != nil {
		t.Fatal(err)
	}
	live, _, err := s.GetAdminMatches(ctx)
	if err != nil || len(live) != 1 {
		t.Fatalf("live: %+v err %v", live, err)
	}
	for i := 0; i < 3; i++ {
		if err := s.RecordWebhook(ctx, player, "KnownNickname", rich); err != nil {
			t.Fatal(err)
		}
	}
	closeBody := map[string]interface{}{"event": "match_status_finished", "payload": map[string]interface{}{"id": "history-review-match"}}
	if err := s.RecordWebhook(ctx, player, "KnownNickname", closeBody); err != nil {
		t.Fatal(err)
	}
	if err := s.CloseLiveMatch(ctx, player); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordWebhook(ctx, player, "KnownNickname", rich); err != nil {
		t.Fatal(err)
	}
	matches, total, err := s.GetPlayerMatches(ctx, player, 50, 0)
	if err != nil || total != 1 || len(matches) != 1 {
		t.Fatalf("unique history: %+v total %d err %v", matches, total, err)
	}
	if matches[0].Status != "closed" || matches[0].Map != "de_mirage" || matches[0].Matchup == nil || !hasTeams(matches[0].Payload) {
		t.Fatalf("lost completion or detail: %+v", matches[0])
	}
	p, err := s.GetWebhookPlayer(ctx, player)
	if err != nil || p.GameCount != 1 || p.Username != "KnownNickname" {
		t.Fatalf("player: %+v err %v", p, err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.RecordWebhook(ctx, player, "KnownNickname", closeBody); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	for i := 0; i < 55; i++ {
		body := map[string]interface{}{"event": "match_status_finished", "payload": map[string]interface{}{"id": fmt.Sprintf("past-%03d", i)}}
		if err := s.RecordWebhook(ctx, player, "KnownNickname", body); err != nil {
			t.Fatal(err)
		}
	}
	matches, total, err = s.GetPlayerMatches(ctx, player, 50, 0)
	if err != nil || total != 56 || len(matches) != 50 {
		t.Fatalf("page1 len %d total %d err %v", len(matches), total, err)
	}
	page2, _, err := s.GetPlayerMatches(ctx, player, 50, 50)
	if err != nil || len(page2) != 6 {
		t.Fatalf("page2 len %d err %v", len(page2), err)
	}
	live, past, err := s.GetAdminMatches(ctx)
	if err != nil || len(live) != 0 || len(past) != 20 {
		t.Fatalf("overview live %d past %d err %v", len(live), len(past), err)
	}
	if err := s.RecordWebhook(ctx, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "SecondPlayer", map[string]interface{}{}); err != nil {
		t.Fatal(err)
	}
	players, err := s.GetWebhookPlayers(ctx)
	if err != nil || len(players) != 2 {
		t.Fatalf("all players: %+v err %v", players, err)
	}
	for i := 0; i < 2; i++ {
		duplicate := map[string]interface{}{"event": "match_status_finished", "payload": map[string]interface{}{"id": "past-054"}}
		if err := s.RecordWebhook(ctx, fmt.Sprintf("duplicate-player-%d", i), "Teammate", duplicate); err != nil {
			t.Fatal(err)
		}
	}
	_, past, err = s.GetAdminMatches(ctx)
	if err != nil || len(past) != 20 {
		t.Fatalf("distinct past: %d %v", len(past), err)
	}
	seen := map[string]bool{}
	for _, match := range past {
		if seen[match.MatchID] {
			t.Fatalf("duplicate past match %s", match.MatchID)
		}
		seen[match.MatchID] = true
	}

}

func TestHistoryMemoryReview(t *testing.T) { exerciseWebhookHistory(t, NewMemoryStore()) }

func TestHistoryMongoReview(t *testing.T) {
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("set MONGO_TEST_URI for persistent history tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	name := fmt.Sprintf("bonbonn_history_review_%d", time.Now().UnixNano())
	s, err := NewMongoStore(ctx, uri, name, "players", "matchrooms")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.db.Drop(context.Background()); s.Close(context.Background()) }()
	exerciseWebhookHistory(t, s)

	// Reuse pre-existing index names without changing their constraints.
	if err = s.streamerCol.Indexes().DropOne(ctx, "name_1"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.streamerCol.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "name", Value: 1}}, Options: options.Index().SetName("legacy_custom_name")}); err != nil {
		t.Fatal(err)
	}
	second, err := NewMongoStore(ctx, uri, name, "players", "matchrooms")
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close(context.Background())
	p, err := second.GetWebhookPlayer(ctx, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	if err != nil || p.GameCount != 56 {
		t.Fatalf("reconnect lost count: %+v %v", p, err)
	}
	// BSON nested documents must survive importing existing matchroom documents.
	player := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	_, err = s.matchroomCol.InsertOne(ctx, bson.M{"streamer": player, "payload": bson.M{"id": "backfilled-match", "teams": bson.A{bson.M{"name": "Old A"}, bson.M{"name": "Old B"}}}, "map": "de_nuke", "meta": bson.M{"isRunning": false, "timestamp": "2026-10-01T12:00:00Z"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.migrations.DeleteOne(ctx, bson.M{"_id": backfillMigrationID}); err != nil {
		t.Fatal(err)
	}
	if err = s.backfillMatchrooms(ctx); err != nil {
		t.Fatal(err)
	}
	matches, total, err := s.GetPlayerMatches(ctx, player, 50, 0)
	if err != nil || total != 1 || len(matches) != 1 || matches[0].MatchID != "backfilled-match" || !hasTeams(matches[0].Payload) {
		t.Fatalf("BSON backfill failed: %+v total %d err %v", matches, total, err)
	}
}
