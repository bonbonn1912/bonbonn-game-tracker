package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/bonbonn1912/bonbonn-game-tracker/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	streamersCollection      = "streamers"
	logsCollection           = "redirect_logs"
	migrationsCollection     = "_migrations"
	webhookPlayersCollection = "webhook_players"
	webhookMatchesCollection = "webhook_matches"
	webhookEventsCollection  = "webhook_events"

	seedMigrationID     = "seed_streamers_v1"
	backfillMigrationID = "backfill_matchrooms_v1"
)

type MongoStore struct {
	client            *mongo.Client
	db                *mongo.Database
	playerCol         *mongo.Collection
	matchroomCol      *mongo.Collection
	streamerCol       *mongo.Collection
	logsCol           *mongo.Collection
	migrations        *mongo.Collection
	webhookPlayersCol *mongo.Collection
	webhookMatchesCol *mongo.Collection
	webhookEventsCol  *mongo.Collection
}

func NewMongoStore(ctx context.Context, uri, dbName, playerColName, matchroomColName string) (*MongoStore, error) {
	clientOptions := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(5 * time.Second).
		SetTimeout(5 * time.Second)
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	db := client.Database(dbName)
	ms := &MongoStore{
		client:            client,
		db:                db,
		playerCol:         db.Collection(playerColName),
		matchroomCol:      db.Collection(matchroomColName),
		streamerCol:       db.Collection(streamersCollection),
		logsCol:           db.Collection(logsCollection),
		migrations:        db.Collection(migrationsCollection),
		webhookPlayersCol: db.Collection(webhookPlayersCollection),
		webhookMatchesCol: db.Collection(webhookMatchesCollection),
		webhookEventsCol:  db.Collection(webhookEventsCollection),
	}

	if err := ms.initIndexes(ctx); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to initialize mongodb indexes: %w", err)
	}

	if err := ms.seedStreamers(ctx); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to run seed streamers migration: %w", err)
	}

	if err := ms.backfillMatchrooms(ctx); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to run backfill matchrooms migration: %w", err)
	}

	return ms, nil
}

func (m *MongoStore) initIndexes(ctx context.Context) error {
	// streamerCol: unique index on _id is default, can also index name
	_, err := ensureMongoIndexes(ctx, m.streamerCol, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index(),
	}})
	if err != nil {
		return err
	}

	// logsCol: index on streamerId and createdAt descending
	_, err = ensureMongoIndexes(ctx, m.logsCol, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "streamerId", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index(),
		},
	})
	if err != nil {
		return err
	}

	// matchroomCol: index on streamer and meta.isRunning
	_, err = ensureMongoIndexes(ctx, m.matchroomCol, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "streamer", Value: 1}, {Key: "meta.isRunning", Value: 1}},
		Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"meta.isRunning": true}),
	}})
	if err != nil {
		return err
	}

	// webhookPlayersCol: index on lastWebhookAt descending
	_, err = ensureMongoIndexes(ctx, m.webhookPlayersCol, []mongo.IndexModel{{
		Keys:    bson.D{{Key: "lastWebhookAt", Value: -1}},
		Options: options.Index(),
	}})
	if err != nil {
		return err
	}

	// webhookMatchesCol: unique compound index on {playerId: 1, matchId: 1}
	_, err = ensureMongoIndexes(ctx, m.webhookMatchesCol, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "playerId", Value: 1}, {Key: "matchId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "playerId", Value: 1}, {Key: "lastSeenAt", Value: -1}, {Key: "_id", Value: -1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}, {Key: "lastSeenAt", Value: -1}},
			Options: options.Index(),
		},
	})
	if err != nil {
		return err
	}

	// webhookEventsCol: index on playerId, matchId, timestamp
	_, err = ensureMongoIndexes(ctx, m.webhookEventsCol, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "playerId", Value: 1}, {Key: "timestamp", Value: -1}},
			Options: options.Index(),
		},
		{
			Keys:    bson.D{{Key: "matchId", Value: 1}},
			Options: options.Index(),
		},
	})
	return err
}

func (m *MongoStore) seedStreamers(ctx context.Context) error {
	var mig bson.M
	err := m.migrations.FindOne(ctx, bson.M{"_id": seedMigrationID}).Decode(&mig)
	if err == nil {
		return nil
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("failed checking migration status: %w", err)
	}

	for _, s := range DefaultSeedStreamers {
		filter := bson.M{"_id": s.ID}
		update := bson.M{
			"$setOnInsert": bson.M{
				"_id":     s.ID,
				"name":    s.Name,
				"url":     s.URL,
				"enabled": s.Enabled,
			},
		}
		_, err := m.streamerCol.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
		if err != nil {
			return fmt.Errorf("failed seeding streamer %s: %w", s.ID, err)
		}
	}

	_, err = m.migrations.InsertOne(ctx, bson.M{
		"_id":        seedMigrationID,
		"applied_at": time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return fmt.Errorf("failed recording migration marker: %w", err)
	}

	return nil
}

// backfillMatchrooms backfills existing previously persisted matchrooms into canonical history
// and tracked players list. Uses normalizeBSON to correctly parse nested bson.M values.
func (m *MongoStore) backfillMatchrooms(ctx context.Context) error {
	var mig bson.M
	err := m.migrations.FindOne(ctx, bson.M{"_id": backfillMigrationID}).Decode(&mig)
	if err == nil {
		return nil
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("failed checking backfill migration status: %w", err)
	}

	cursor, err := m.matchroomCol.Find(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed querying matchrooms for backfill: %w", err)
	}
	defer cursor.Close(ctx)

	streamersSeen := make(map[string]struct{})
	for cursor.Next(ctx) {
		var rawDoc bson.M
		if err := cursor.Decode(&rawDoc); err != nil {
			return fmt.Errorf("failed decoding matchroom for backfill: %w", err)
		}
		doc := normalizeBSON(rawDoc)
		streamer, _ := doc["streamer"].(string)
		if streamer == "" {
			continue
		}
		streamersSeen[streamer] = struct{}{}

		matchID := getNestedString(doc, "payload", "id")
		payload, _ := doc["payload"].(map[string]interface{})
		meta, _ := doc["meta"].(map[string]interface{})
		isRunning, _ := meta["isRunning"].(bool)
		insertedAt, _ := meta["inserted_at"].(string)
		timestamp, _ := meta["timestamp"].(string)
		if insertedAt == "" {
			insertedAt = timestamp
		}
		if timestamp == "" {
			timestamp = insertedAt
		}
		if timestamp == "" {
			timestamp = time.Now().UTC().Format(time.RFC3339Nano)
			insertedAt = timestamp
		}

		// Find streamer username
		username := streamer
		if s, sErr := m.GetStreamer(ctx, streamer); sErr == nil && s.Name != "" {
			username = s.Name
		}

		// Upsert player
		playerFilter := bson.M{"_id": streamer}
		playerUpdate := bson.M{
			"$setOnInsert": bson.M{
				"_id":           streamer,
				"username":      username,
				"gameCount":     0,
				"lastWebhookAt": timestamp,
			},
		}
		if _, err := m.webhookPlayersCol.UpdateOne(ctx, playerFilter, playerUpdate, options.UpdateOne().SetUpsert(true)); err != nil {
			return fmt.Errorf("failed upserting player in backfill: %w", err)
		}

		if matchID != "" {
			status := "closed"
			if isRunning {
				status = "live"
			}
			matchDoc := bson.M{
				"_id":         fmt.Sprintf("%s:%s", streamer, matchID),
				"playerId":    streamer,
				"username":    username,
				"matchId":     matchID,
				"status":      status,
				"firstSeenAt": insertedAt,
				"lastSeenAt":  timestamp,
				"map":         doc["map"],
				"matchup":     doc["matchup"],
				"payload":     payload,
				"event":       "match_status_configuring",
			}
			matchFilter := bson.M{"playerId": streamer, "matchId": matchID}
			if _, err := m.webhookMatchesCol.UpdateOne(ctx, matchFilter, bson.M{"$setOnInsert": matchDoc}, options.UpdateOne().SetUpsert(true)); err != nil {
				return fmt.Errorf("failed upserting match in backfill: %w", err)
			}
		}
	}
	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error during matchrooms backfill: %w", err)
	}

	for sID := range streamersSeen {
		if err := m.syncPlayerGameCount(ctx, sID); err != nil {
			return err
		}
	}

	_, err = m.migrations.InsertOne(ctx, bson.M{
		"_id":        backfillMigrationID,
		"applied_at": time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return fmt.Errorf("failed recording backfill migration marker: %w", err)
	}
	return nil
}

func (m *MongoStore) syncPlayerGameCount(ctx context.Context, playerID string) error {
	count, err := m.webhookMatchesCol.CountDocuments(ctx, bson.M{"playerId": playerID, "matchId": bson.M{"$ne": ""}})
	if err != nil {
		return err
	}
	// The history only grows. Concurrent deliveries must not replace a newer
	// count with an older snapshot, and retries repair interrupted count updates.
	_, err = m.webhookPlayersCol.UpdateOne(ctx, bson.M{"_id": playerID}, bson.M{"$max": bson.M{"gameCount": int(count)}})
	return err
}

func (m *MongoStore) GetStreamers(ctx context.Context) ([]models.Streamer, error) {
	cursor, err := m.streamerCol.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var streamers []models.Streamer
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err == nil {
			id, _ := doc["_id"].(string)
			name, _ := doc["name"].(string)
			url, _ := doc["url"].(string)
			enabled, _ := doc["enabled"].(bool)
			streamers = append(streamers, models.Streamer{
				ID:      id,
				Name:    name,
				URL:     url,
				Enabled: enabled,
			})
		}
	}
	if streamers == nil {
		streamers = []models.Streamer{}
	}
	return streamers, cursor.Err()
}

func (m *MongoStore) GetStreamer(ctx context.Context, id string) (*models.Streamer, error) {
	var doc bson.M
	err := m.streamerCol.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	name, _ := doc["name"].(string)
	url, _ := doc["url"].(string)
	enabled, _ := doc["enabled"].(bool)

	return &models.Streamer{
		ID:      id,
		Name:    name,
		URL:     url,
		Enabled: enabled,
	}, nil
}

func (m *MongoStore) CreateStreamer(ctx context.Context, streamer models.Streamer) (*models.Streamer, error) {
	doc := bson.M{
		"_id":     streamer.ID,
		"name":    streamer.Name,
		"url":     streamer.URL,
		"enabled": streamer.Enabled,
	}

	_, err := m.streamerCol.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDuplicateKey
		}
		return nil, err
	}

	return &streamer, nil
}

func (m *MongoStore) UpdateStreamer(ctx context.Context, id string, streamer models.Streamer) (*models.Streamer, error) {
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"name":    streamer.Name,
			"url":     streamer.URL,
			"enabled": streamer.Enabled,
		},
	}

	res, err := m.streamerCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	if res.MatchedCount == 0 {
		return nil, ErrNotFound
	}

	streamer.ID = id
	return &streamer, nil
}

func (m *MongoStore) DeleteStreamer(ctx context.Context, id string) error {
	res, err := m.streamerCol.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (m *MongoStore) LogRedirectEvent(ctx context.Context, event models.RedirectEvent) error {
	doc := bson.M{
		"_id":          event.ID,
		"streamerId":   event.StreamerID,
		"streamerName": event.StreamerName,
		"matchId":      event.MatchID,
		"target":       event.Target,
		"outcome":      event.Outcome,
		"createdAt":    event.CreatedAt,
		"requestId":    event.RequestID,
		"ip":           event.IP,
		"userAgent":    event.UserAgent,
	}

	_, err := m.logsCol.InsertOne(ctx, doc)
	if err != nil {
		slog.Error("failed to persist redirect event in mongodb",
			"error", err,
			"id", event.ID,
			"streamerId", event.StreamerID,
			"outcome", event.Outcome,
			"target", event.Target,
		)
		return err
	}
	return nil
}

func (m *MongoStore) GetRedirectEvents(ctx context.Context, limit int) ([]models.RedirectEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	findOptions := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := m.logsCol.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []models.RedirectEvent
	for cursor.Next(ctx) {
		var raw bson.M
		if err := cursor.Decode(&raw); err == nil {
			id, _ := raw["_id"].(string)
			sID, _ := raw["streamerId"].(string)
			sName, _ := raw["streamerName"].(string)
			mID, _ := raw["matchId"].(string)
			target, _ := raw["target"].(string)
			outcome, _ := raw["outcome"].(string)
			created, _ := raw["createdAt"].(string)
			reqID, _ := raw["requestId"].(string)
			ip, _ := raw["ip"].(string)
			ua, _ := raw["userAgent"].(string)

			events = append(events, models.RedirectEvent{
				ID:           id,
				StreamerID:   sID,
				StreamerName: sName,
				MatchID:      mID,
				Target:       target,
				Outcome:      outcome,
				CreatedAt:    created,
				RequestID:    reqID,
				IP:           ip,
				UserAgent:    ua,
			})
		}
	}
	if events == nil {
		events = []models.RedirectEvent{}
	}
	return events, cursor.Err()
}

// AddLiveMatch inserts a persisted match document with meta.isRunning: true.
// Deep copies matchData to avoid mutating the caller's map with meta or _id.
// Also synchronizes and enriches canonical match in webhook_matches after map/matchup calculation.
func (m *MongoStore) AddLiveMatch(ctx context.Context, streamerID string, matchData map[string]interface{}) error {
	// Check if already live in database
	var existing bson.M
	err := m.matchroomCol.FindOne(ctx, bson.M{"streamer": streamerID, "meta.isRunning": true}).Decode(&existing)
	if err == nil {
		return ErrAlreadyLive
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	docCopy := cloneMap(matchData)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	docCopy["meta"] = map[string]interface{}{
		"inserted_at": now,
		"timestamp":   now,
		"isRunning":   true,
	}
	docCopy["streamer"] = streamerID

	_, err = m.matchroomCol.InsertOne(ctx, docCopy)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrAlreadyLive
		}
		slog.Error("failed to persist active match in mongodb", "streamer", streamerID, "error", err)
		return err
	}

	// Enrich canonical record in webhook_matches after map/matchup
	matchID := getNestedString(matchData, "payload", "id")
	if matchID != "" {
		payload, _ := matchData["payload"].(map[string]interface{})
		mapVal := matchData["map"]
		matchupVal := matchData["matchup"]

		matchFilter := bson.M{"playerId": streamerID, "matchId": matchID}
		var canonicalRaw bson.M
		findErr := m.webhookMatchesCol.FindOne(ctx, matchFilter).Decode(&canonicalRaw)
		if findErr == nil {
			canonical := normalizeBSON(canonicalRaw)
			// Existing match: enrichment must NOT reset closed canonical back to live!
			existingStatus, _ := canonical["status"].(string)
			setFields := bson.M{
				"lastSeenAt": now,
			}
			if existingStatus != "closed" {
				setFields["status"] = "live"
			}
			if mapVal != nil && mapVal != "" && mapVal != "unknown" {
				setFields["map"] = mapVal
			}
			if matchupVal != nil {
				setFields["matchup"] = matchupVal
			}
			if hasTeams(payload) || !hasTeams(canonical["payload"]) {
				setFields["payload"] = cloneMap(payload)
			}
			_, _ = m.webhookMatchesCol.UpdateOne(ctx, matchFilter, bson.M{"$set": setFields})
		} else if errors.Is(findErr, mongo.ErrNoDocuments) {
			username := streamerID
			if s, sErr := m.GetStreamer(ctx, streamerID); sErr == nil && s.Name != "" {
				username = s.Name
			}
			newDoc := bson.M{
				"_id":         fmt.Sprintf("%s:%s", streamerID, matchID),
				"playerId":    streamerID,
				"username":    username,
				"matchId":     matchID,
				"status":      "live",
				"firstSeenAt": now,
				"lastSeenAt":  now,
				"map":         mapVal,
				"matchup":     matchupVal,
				"payload":     cloneMap(payload),
				"event":       "match_status_configuring",
			}
			_, _ = m.webhookMatchesCol.UpdateOne(ctx, matchFilter, bson.M{"$setOnInsert": newDoc}, options.UpdateOne().SetUpsert(true))
			if err := m.syncPlayerGameCount(ctx, streamerID); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetLiveMatch queries MongoDB directly for active match with meta.isRunning = true.
// Returns (match, found, err). Strips _id and meta from public live match retrieval (DB only).
func (m *MongoStore) GetLiveMatch(ctx context.Context, streamerID string) (map[string]interface{}, bool, error) {
	var doc bson.M
	err := m.matchroomCol.FindOne(ctx, bson.M{"streamer": streamerID, "meta.isRunning": true}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, false, nil
		}
		return nil, false, err
	}

	normalized := normalizeBSON(doc)
	delete(normalized, "_id")
	delete(normalized, "meta")
	return normalized, true, nil
}

// CloseLiveMatch closes the live match for streamerID in matchrooms and marks it closed in webhook_matches.
func (m *MongoStore) CloseLiveMatch(ctx context.Context, streamerID string) error {
	filter := bson.M{"streamer": streamerID, "meta.isRunning": true}
	update := bson.M{"$set": bson.M{"meta.isRunning": false}}

	res, err := m.matchroomCol.UpdateOne(ctx, filter, update)
	if err != nil {
		slog.Error("failed to close live match in mongodb", "streamer", streamerID, "error", err)
		return err
	}
	if res.ModifiedCount == 0 {
		slog.Info("no active match found in db to close for streamer", "streamer", streamerID)
	}

	// Close matching live match in webhook_matches for streamerID only (without global impact)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	matchFilter := bson.M{"playerId": streamerID, "status": "live"}
	_, err = m.webhookMatchesCol.UpdateMany(ctx, matchFilter, bson.M{
		"$set": bson.M{
			"status":     "closed",
			"lastSeenAt": now,
		},
	})

	return err
}

func (m *MongoStore) GetAllLiveMatches(ctx context.Context) (map[string]map[string]interface{}, error) {
	filter := bson.M{"meta.isRunning": true}
	cursor, err := m.matchroomCol.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make(map[string]map[string]interface{})
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err == nil {
			if s, ok := doc["streamer"].(string); ok && s != "" {
				norm := normalizeBSON(doc)
				delete(norm, "_id")
				delete(norm, "meta")
				result[s] = norm
			}
		}
	}
	return result, cursor.Err()
}

func (m *MongoStore) CountActiveMatches(ctx context.Context) (int, error) {
	count, err := m.matchroomCol.CountDocuments(ctx, bson.M{"meta.isRunning": true})
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (m *MongoStore) SavePlayer(ctx context.Context, insertType string, playerData map[string]interface{}) error {
	docCopy := cloneMap(playerData)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	docCopy["insertType"] = insertType
	docCopy["meta"] = map[string]interface{}{
		"inserted_at": now,
		"timestamp":   now,
		"isRunning":   nil,
	}
	_, err := m.playerCol.InsertOne(ctx, docCopy)
	if err != nil {
		slog.Error("failed to save player in mongodb", "error", err)
		return err
	}
	return nil
}

// RecordWebhook persists audit event in webhook_events, upserts player in webhook_players,
// and manages canonical match record in webhook_matches.
func (m *MongoStore) RecordWebhook(ctx context.Context, playerID string, resolvedUsername string, body map[string]interface{}) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	event := getNestedString(body, "event")
	matchID := getNestedString(body, "payload", "id")

	// 1. Raw event audit in webhook_events
	rawDoc := bson.M{
		"_id":       newUUID(),
		"playerId":  playerID,
		"matchId":   matchID,
		"event":     event,
		"timestamp": now,
		"rawBody":   cloneMap(body),
	}
	if _, err := m.webhookEventsCol.InsertOne(ctx, rawDoc); err != nil {
		slog.Warn("could not persist raw webhook event", "error", err, "player", playerID)
	}

	// 2. Tracked player in webhook_players
	playerFilter := bson.M{"_id": playerID}
	playerSet := bson.M{
		"lastWebhookAt": now,
	}
	setOnInsert := bson.M{
		"_id":       playerID,
		"gameCount": 0,
	}
	// Avoid path conflict between $setOnInsert and $set for username!
	if resolvedUsername != "" && resolvedUsername != playerID {
		playerSet["username"] = resolvedUsername
	} else {
		setOnInsert["username"] = resolvedUsername
	}
	playerUpdate := bson.M{
		"$setOnInsert": setOnInsert,
		"$set":         playerSet,
	}
	_, err := m.webhookPlayersCol.UpdateOne(ctx, playerFilter, playerUpdate, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("failed to upsert webhook player: %w", err)
	}

	// 3. Canonical match in webhook_matches (if matchID is non-empty)
	if matchID != "" {
		status := "live"
		if event != "match_status_configuring" && event != "match_object_created" && event != "" {
			status = "closed"
		}

		payload, _ := body["payload"].(map[string]interface{})
		mapVal := body["map"]
		matchupVal := body["matchup"]

		matchFilter := bson.M{"playerId": playerID, "matchId": matchID}
		var existingRaw bson.M
		findErr := m.webhookMatchesCol.FindOne(ctx, matchFilter).Decode(&existingRaw)
		if findErr != nil && !errors.Is(findErr, mongo.ErrNoDocuments) {
			return fmt.Errorf("failed to check existing webhook match: %w", findErr)
		}

		if errors.Is(findErr, mongo.ErrNoDocuments) {
			newDoc := bson.M{
				"_id":         fmt.Sprintf("%s:%s", playerID, matchID),
				"playerId":    playerID,
				"username":    resolvedUsername,
				"matchId":     matchID,
				"status":      status,
				"firstSeenAt": now,
				"lastSeenAt":  now,
				"map":         mapVal,
				"matchup":     matchupVal,
				"payload":     cloneMap(payload),
				"event":       event,
			}
			_, err = m.webhookMatchesCol.UpdateOne(ctx, matchFilter, bson.M{"$setOnInsert": newDoc}, options.UpdateOne().SetUpsert(true))
			if err != nil {
				return fmt.Errorf("failed to insert canonical match: %w", err)
			}
		} else {
			existing := normalizeBSON(existingRaw)
			// Update existing canonical record, merging late closing events without overwriting richer configuring data
			existingStatus, _ := existing["status"].(string)
			setFields := bson.M{
				"lastSeenAt": now,
				"event":      event,
			}
			// Closing canonical must stay closed on delayed config retries!
			if existingStatus == "closed" {
				setFields["status"] = "closed"
			} else if status != "" {
				setFields["status"] = status
			}
			if resolvedUsername != "" && resolvedUsername != playerID {
				setFields["username"] = resolvedUsername
			}
			if mapVal != nil && mapVal != "" && mapVal != "unknown" {
				setFields["map"] = mapVal
			}
			if matchupVal != nil {
				setFields["matchup"] = matchupVal
			}
			if hasTeams(payload) || !hasTeams(existing["payload"]) {
				setFields["payload"] = cloneMap(payload)
			}

			_, err = m.webhookMatchesCol.UpdateOne(ctx, bson.M{"_id": existing["_id"]}, bson.M{"$set": setFields})
			if err != nil {
				return fmt.Errorf("failed to update canonical match: %w", err)
			}
		}
	}

	return m.syncPlayerGameCount(ctx, playerID)
}

// GetWebhookPlayer returns a single tracked player by ID.
func (m *MongoStore) GetWebhookPlayer(ctx context.Context, playerID string) (*models.WebhookPlayer, error) {
	var doc bson.M
	err := m.webhookPlayersCol.FindOne(ctx, bson.M{"_id": playerID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	id, _ := doc["_id"].(string)
	username, _ := doc["username"].(string)
	lastWebhookAt, _ := doc["lastWebhookAt"].(string)
	gameCount := 0
	switch v := doc["gameCount"].(type) {
	case int32:
		gameCount = int(v)
	case int64:
		gameCount = int(v)
	case int:
		gameCount = v
	case float64:
		gameCount = int(v)
	}

	return &models.WebhookPlayer{
		ID:            id,
		Username:      username,
		GameCount:     gameCount,
		LastWebhookAt: lastWebhookAt,
	}, nil
}

// GetWebhookPlayers returns all tracked players sorted by lastWebhookAt descending.
func (m *MongoStore) GetWebhookPlayers(ctx context.Context) ([]models.WebhookPlayer, error) {
	findOptions := options.Find().SetSort(bson.D{{Key: "lastWebhookAt", Value: -1}, {Key: "_id", Value: 1}})
	cursor, err := m.webhookPlayersCol.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	players := make([]models.WebhookPlayer, 0)
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err == nil {
			id, _ := doc["_id"].(string)
			username, _ := doc["username"].(string)
			lastWebhookAt, _ := doc["lastWebhookAt"].(string)
			gameCount := 0
			switch v := doc["gameCount"].(type) {
			case int32:
				gameCount = int(v)
			case int64:
				gameCount = int(v)
			case int:
				gameCount = v
			case float64:
				gameCount = int(v)
			}
			players = append(players, models.WebhookPlayer{
				ID:            id,
				Username:      username,
				GameCount:     gameCount,
				LastWebhookAt: lastWebhookAt,
			})
		}
	}
	return players, cursor.Err()
}

// GetPlayerMatches returns paginated match history for a given player, newest first.
func (m *MongoStore) GetPlayerMatches(ctx context.Context, playerID string, limit, offset int) ([]models.WebhookMatch, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	filter := bson.M{
		"playerId": playerID,
		"matchId":  bson.M{"$ne": ""},
	}

	totalCount, err := m.webhookMatchesCol.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "lastSeenAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := m.webhookMatchesCol.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	matches := make([]models.WebhookMatch, 0)
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err == nil {
			matches = append(matches, decodeWebhookMatch(doc))
		}
	}

	return matches, int(totalCount), cursor.Err()
}

// GetAdminMatches returns live authoritative matches and the last 20 completed past matches.
func (m *MongoStore) GetAdminMatches(ctx context.Context) (live []models.WebhookMatch, past []models.WebhookMatch, err error) {
	live = make([]models.WebhookMatch, 0)
	past = make([]models.WebhookMatch, 0)

	// 1. Live matches from GetAllLiveMatches (authoritative active DB matches)
	allLive, err := m.GetAllLiveMatches(ctx)
	if err != nil {
		return nil, nil, err
	}

	for streamerID, gameMap := range allLive {
		matchID := getNestedString(gameMap, "payload", "id")
		var canonicalDoc bson.M
		findErr := m.webhookMatchesCol.FindOne(ctx, bson.M{"playerId": streamerID, "matchId": matchID}).Decode(&canonicalDoc)
		if findErr == nil {
			wMatch := decodeWebhookMatch(canonicalDoc)
			wMatch.Status = "live"
			if wMatch.Map == nil || wMatch.Map == "" {
				wMatch.Map = gameMap["map"]
			}
			if wMatch.Matchup == nil {
				wMatch.Matchup = gameMap["matchup"]
			}
			live = append(live, wMatch)
		} else {
			streamerName := streamerID
			if s, sErr := m.GetStreamer(ctx, streamerID); sErr == nil && s.Name != "" {
				streamerName = s.Name
			}
			now := time.Now().UTC().Format(time.RFC3339Nano)
			payloadMap, _ := gameMap["payload"].(map[string]interface{})
			if payloadMap == nil {
				payloadMap = map[string]interface{}{}
			}
			live = append(live, models.WebhookMatch{
				ID:          fmt.Sprintf("%s:%s", streamerID, matchID),
				PlayerID:    streamerID,
				Username:    streamerName,
				MatchID:     matchID,
				Status:      "live",
				FirstSeenAt: now,
				LastSeenAt:  now,
				Map:         gameMap["map"],
				Matchup:     gameMap["matchup"],
				Payload:     payloadMap,
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

	// 2. Past matches: last 20 completed matches from webhookMatchesCol
	pastFilter := bson.M{
		"status":  "closed",
		"matchId": bson.M{"$ne": ""},
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: pastFilter}},
		{{Key: "$sort", Value: bson.D{{Key: "lastSeenAt", Value: -1}, {Key: "_id", Value: -1}}}},
		{{Key: "$group", Value: bson.M{"_id": "$matchId", "doc": bson.M{"$first": "$$ROOT"}}}},
		{{Key: "$replaceRoot", Value: bson.M{"newRoot": "$doc"}}},
		{{Key: "$sort", Value: bson.D{{Key: "lastSeenAt", Value: -1}, {Key: "_id", Value: -1}}}},
		{{Key: "$limit", Value: 20}},
	}
	cursor, err := m.webhookMatchesCol.Aggregate(ctx, pipeline)
	if err != nil {
		return live, past, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err == nil {
			past = append(past, decodeWebhookMatch(doc))
		}
	}

	return live, past, cursor.Err()
}

func (m *MongoStore) Ping(ctx context.Context) error {
	return m.client.Ping(ctx, nil)
}

func (m *MongoStore) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

func decodeWebhookMatch(doc bson.M) models.WebhookMatch {
	norm := normalizeBSON(doc)
	id, _ := norm["_id"].(string)
	playerID, _ := norm["playerId"].(string)
	username, _ := norm["username"].(string)
	matchID, _ := norm["matchId"].(string)
	status, _ := norm["status"].(string)
	firstSeenAt, _ := norm["firstSeenAt"].(string)
	lastSeenAt, _ := norm["lastSeenAt"].(string)
	event, _ := norm["event"].(string)

	payload, _ := norm["payload"].(map[string]interface{})
	if payload == nil {
		payload = map[string]interface{}{}
	}

	return models.WebhookMatch{
		ID:          id,
		PlayerID:    playerID,
		Username:    username,
		MatchID:     matchID,
		Status:      status,
		FirstSeenAt: firstSeenAt,
		LastSeenAt:  lastSeenAt,
		Map:         norm["map"],
		Matchup:     norm["matchup"],
		Payload:     payload,
		Event:       event,
	}
}

func hasTeams(payload interface{}) bool {
	if payload == nil {
		return false
	}
	norm := normalizeValue(payload)
	p, ok := norm.(map[string]interface{})
	if !ok || p == nil {
		return false
	}
	teams, ok := p["teams"].([]interface{})
	return ok && len(teams) > 0
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

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func cloneMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	norm := normalizeValue(m)
	if r, ok := norm.(map[string]interface{}); ok {
		return r
	}
	c := make(map[string]interface{}, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case bson.D:
		m := make(map[string]interface{}, len(val))
		for _, elem := range val {
			m[elem.Key] = normalizeValue(elem.Value)
		}
		return m
	case bson.M:
		m := make(map[string]interface{}, len(val))
		for k, v := range val {
			m[k] = normalizeValue(v)
		}
		return m
	case map[string]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, v := range val {
			m[k] = normalizeValue(v)
		}
		return m
	case bson.A:
		arr := make([]interface{}, len(val))
		for i, item := range val {
			arr[i] = normalizeValue(item)
		}
		return arr
	case []interface{}:
		arr := make([]interface{}, len(val))
		for i, item := range val {
			arr[i] = normalizeValue(item)
		}
		return arr
	default:
		return val
	}
}

func normalizeBSON(doc bson.M) map[string]interface{} {
	if doc == nil {
		return nil
	}
	norm := normalizeValue(doc)
	if m, ok := norm.(map[string]interface{}); ok {
		return m
	}
	return make(map[string]interface{})
}

// Preserve index names from prior deployments. Mongo validates uniqueness and
// partial-filter options even when the keys already exist.
func ensureMongoIndexes(ctx context.Context, collection *mongo.Collection, models []mongo.IndexModel) ([]string, error) {
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var specs []struct {
		Key  bson.D `bson:"key"`
		Name string `bson:"name"`
	}
	if err := cursor.All(ctx, &specs); err != nil {
		return nil, err
	}
	for i := range models {
		for _, spec := range specs {
			if sameIndexKeys(spec.Key, models[i].Keys) {
				if models[i].Options == nil {
					models[i].Options = options.Index()
				}
				models[i].Options.SetName(spec.Name)
				break
			}
		}
	}
	return collection.Indexes().CreateMany(ctx, models)
}

func sameIndexKeys(existing bson.D, desired interface{}) bool {
	keys, ok := desired.(bson.D)
	if !ok || len(existing) != len(keys) {
		return false
	}
	for i := range keys {
		if existing[i].Key != keys[i].Key || fmt.Sprint(existing[i].Value) != fmt.Sprint(keys[i].Value) {
			return false
		}
	}
	return true
}
