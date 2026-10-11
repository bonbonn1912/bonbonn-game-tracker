package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/bonbonn1912/bonbonn-game-tracker/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const seedMigrationID = "seed_streamers_v1"

// schemaLockID serializes schema setup when several instances start at once.
const schemaLockID = 7_311_202_610

const schema = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	id         text PRIMARY KEY,
	applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS streamers (
	id      text PRIMARY KEY,
	name    text NOT NULL,
	url     text NOT NULL,
	enabled boolean NOT NULL
);
CREATE INDEX IF NOT EXISTS streamers_name_idx ON streamers (name);

CREATE TABLE IF NOT EXISTS redirect_logs (
	id            text PRIMARY KEY,
	streamer_id   text NOT NULL,
	streamer_name text NOT NULL,
	match_id      text NOT NULL,
	target        text NOT NULL,
	outcome       text NOT NULL,
	created_at    timestamptz NOT NULL,
	request_id    text NOT NULL,
	ip            text NOT NULL,
	user_agent    text NOT NULL
);
CREATE INDEX IF NOT EXISTS redirect_logs_created_idx ON redirect_logs (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS redirect_logs_streamer_idx ON redirect_logs (streamer_id, created_at DESC);

CREATE TABLE IF NOT EXISTS matchrooms (
	id          text PRIMARY KEY,
	streamer    text NOT NULL,
	is_running  boolean NOT NULL,
	inserted_at timestamptz NOT NULL,
	updated_at  timestamptz NOT NULL,
	document    jsonb NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS matchrooms_running_streamer_idx ON matchrooms (streamer) WHERE is_running;

CREATE TABLE IF NOT EXISTS player_lookups (
	id          text PRIMARY KEY,
	insert_type text NOT NULL,
	inserted_at timestamptz NOT NULL,
	document    jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS webhook_players (
	id              text PRIMARY KEY,
	username        text NOT NULL,
	game_count      integer NOT NULL DEFAULT 0,
	last_webhook_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS webhook_players_last_webhook_idx ON webhook_players (last_webhook_at DESC);

CREATE TABLE IF NOT EXISTS webhook_matches (
	id            text PRIMARY KEY,
	player_id     text NOT NULL,
	username      text NOT NULL,
	match_id      text NOT NULL,
	status        text NOT NULL,
	first_seen_at timestamptz NOT NULL,
	last_seen_at  timestamptz NOT NULL,
	map           jsonb,
	matchup       jsonb,
	payload       jsonb,
	event         text NOT NULL,
	UNIQUE (player_id, match_id)
);
CREATE INDEX IF NOT EXISTS webhook_matches_player_idx ON webhook_matches (player_id, last_seen_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS webhook_matches_status_idx ON webhook_matches (status, last_seen_at DESC);

CREATE TABLE IF NOT EXISTS webhook_events (
	id          text PRIMARY KEY,
	player_id   text NOT NULL,
	match_id    text NOT NULL,
	event       text NOT NULL,
	received_at timestamptz NOT NULL,
	raw_body    jsonb
);
CREATE INDEX IF NOT EXISTS webhook_events_player_idx ON webhook_events (player_id, received_at DESC);
CREATE INDEX IF NOT EXISTS webhook_events_match_idx ON webhook_events (match_id);

CREATE TABLE IF NOT EXISTS webhook_requests (
	id          text PRIMARY KEY,
	received_at timestamptz NOT NULL,
	method      text NOT NULL,
	path        text NOT NULL,
	headers     jsonb NOT NULL,
	body        text NOT NULL
);
CREATE INDEX IF NOT EXISTS webhook_requests_received_idx ON webhook_requests (received_at DESC, id DESC);
`

const webhookMatchColumns = `id, player_id, username, match_id, status, first_seen_at, last_seen_at, map, matchup, payload, event`

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid database url: %w", err)
	}
	cfg.MaxConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	ps := &PostgresStore{pool: pool}

	if err := ps.initSchema(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to initialize postgres schema: %w", err)
	}

	if err := ps.seedStreamers(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to run seed streamers migration: %w", err)
	}

	return ps, nil
}

func (p *PostgresStore) initSchema(ctx context.Context) error {
	return pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, schemaLockID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, schema)
		return err
	})
}

func (p *PostgresStore) seedStreamers(ctx context.Context) error {
	return pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `INSERT INTO schema_migrations (id) VALUES ($1) ON CONFLICT DO NOTHING`, seedMigrationID)
		if err != nil {
			return fmt.Errorf("failed recording migration marker: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return nil
		}
		for _, s := range DefaultSeedStreamers {
			_, err := tx.Exec(ctx,
				`INSERT INTO streamers (id, name, url, enabled) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
				s.ID, s.Name, s.URL, s.Enabled)
			if err != nil {
				return fmt.Errorf("failed seeding streamer %s: %w", s.ID, err)
			}
		}
		return nil
	})
}

func (p *PostgresStore) RecordWebhookRequest(ctx context.Context, request models.WebhookRequest) error {
	if request.ID == "" {
		request.ID = newUUID()
	}
	headers := request.Headers
	if headers == nil {
		headers = map[string][]string{}
	}
	headersJSON, err := jsonParam(headers)
	if err != nil {
		return err
	}
	_, err = p.pool.Exec(ctx,
		`INSERT INTO webhook_requests (id, received_at, method, path, headers, body) VALUES ($1, $2, $3, $4, $5, $6)`,
		request.ID, parseTimestamp(request.Timestamp), request.Method, request.Path, headersJSON, sanitizeText(request.Body))
	return err
}

func (p *PostgresStore) GetWebhookRequests(ctx context.Context, limit int) ([]models.WebhookRequest, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := p.pool.Query(ctx,
		`SELECT id, received_at, method, path, headers, body FROM webhook_requests ORDER BY received_at DESC, id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]models.WebhookRequest, 0)
	for rows.Next() {
		var (
			r          models.WebhookRequest
			receivedAt time.Time
			headers    []byte
		)
		if err := rows.Scan(&r.ID, &receivedAt, &r.Method, &r.Path, &headers, &r.Body); err != nil {
			return nil, err
		}
		r.Timestamp = formatTimestamp(receivedAt)
		if err := json.Unmarshal(headers, &r.Headers); err != nil {
			return nil, err
		}
		requests = append(requests, r)
	}
	return requests, rows.Err()
}

func (p *PostgresStore) GetStreamers(ctx context.Context) ([]models.Streamer, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, name, url, enabled FROM streamers ORDER BY name, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	streamers := make([]models.Streamer, 0)
	for rows.Next() {
		var s models.Streamer
		if err := rows.Scan(&s.ID, &s.Name, &s.URL, &s.Enabled); err != nil {
			return nil, err
		}
		streamers = append(streamers, s)
	}
	return streamers, rows.Err()
}

func (p *PostgresStore) GetStreamer(ctx context.Context, id string) (*models.Streamer, error) {
	var s models.Streamer
	err := p.pool.QueryRow(ctx, `SELECT id, name, url, enabled FROM streamers WHERE id = $1`, id).
		Scan(&s.ID, &s.Name, &s.URL, &s.Enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (p *PostgresStore) CreateStreamer(ctx context.Context, streamer models.Streamer) (*models.Streamer, error) {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO streamers (id, name, url, enabled) VALUES ($1, $2, $3, $4)`,
		streamer.ID, streamer.Name, streamer.URL, streamer.Enabled)
	if isUniqueViolation(err) {
		return nil, ErrDuplicateKey
	}
	if err != nil {
		return nil, err
	}
	return &streamer, nil
}

func (p *PostgresStore) UpdateStreamer(ctx context.Context, id string, streamer models.Streamer) (*models.Streamer, error) {
	tag, err := p.pool.Exec(ctx,
		`UPDATE streamers SET name = $2, url = $3, enabled = $4 WHERE id = $1`,
		id, streamer.Name, streamer.URL, streamer.Enabled)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	streamer.ID = id
	return &streamer, nil
}

func (p *PostgresStore) DeleteStreamer(ctx context.Context, id string) error {
	tag, err := p.pool.Exec(ctx, `DELETE FROM streamers WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *PostgresStore) LogRedirectEvent(ctx context.Context, event models.RedirectEvent) error {
	if event.ID == "" {
		event.ID = newUUID()
	}
	_, err := p.pool.Exec(ctx,
		`INSERT INTO redirect_logs (id, streamer_id, streamer_name, match_id, target, outcome, created_at, request_id, ip, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		event.ID, sanitizeText(event.StreamerID), event.StreamerName, event.MatchID, event.Target, event.Outcome,
		parseTimestamp(event.CreatedAt), event.RequestID, event.IP, sanitizeText(event.UserAgent))
	if err != nil {
		slog.Error("failed to persist redirect event in postgres",
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

func (p *PostgresStore) GetRedirectEvents(ctx context.Context, limit int) ([]models.RedirectEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := p.pool.Query(ctx,
		`SELECT id, streamer_id, streamer_name, match_id, target, outcome, created_at, request_id, ip, user_agent
		 FROM redirect_logs ORDER BY created_at DESC, id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]models.RedirectEvent, 0)
	for rows.Next() {
		var (
			e         models.RedirectEvent
			createdAt time.Time
		)
		if err := rows.Scan(&e.ID, &e.StreamerID, &e.StreamerName, &e.MatchID, &e.Target, &e.Outcome,
			&createdAt, &e.RequestID, &e.IP, &e.UserAgent); err != nil {
			return nil, err
		}
		e.CreatedAt = formatTimestamp(createdAt)
		events = append(events, e)
	}
	return events, rows.Err()
}

// AddLiveMatch stores the match document as running and enriches the canonical
// record in webhook_matches. The caller's matchData is never mutated.
func (p *PostgresStore) AddLiveMatch(ctx context.Context, streamerID string, matchData map[string]interface{}) error {
	var running bool
	err := p.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM matchrooms WHERE streamer = $1 AND is_running)`, streamerID).Scan(&running)
	if err != nil {
		return err
	}
	if running {
		return ErrAlreadyLive
	}

	docCopy := cloneMap(matchData)
	if docCopy == nil {
		docCopy = map[string]interface{}{}
	}
	delete(docCopy, "_id")
	delete(docCopy, "meta")
	docCopy["streamer"] = streamerID
	document, err := jsonParam(docCopy)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	_, err = p.pool.Exec(ctx,
		`INSERT INTO matchrooms (id, streamer, is_running, inserted_at, updated_at, document) VALUES ($1, $2, true, $3, $3, $4)`,
		newUUID(), streamerID, now, document)
	if isUniqueViolation(err) {
		return ErrAlreadyLive
	}
	if err != nil {
		slog.Error("failed to persist active match in postgres", "streamer", streamerID, "error", err)
		return err
	}

	matchID := getNestedString(matchData, "payload", "id")
	if matchID == "" {
		return nil
	}
	payload, _ := matchData["payload"].(map[string]interface{})
	mapVal := matchData["map"]
	matchupVal := matchData["matchup"]

	username := streamerID
	if s, sErr := p.GetStreamer(ctx, streamerID); sErr == nil && s.Name != "" {
		username = s.Name
	}
	fresh := models.WebhookMatch{
		PlayerID: streamerID,
		Username: username,
		MatchID:  matchID,
		Status:   "live",
		Map:      mapVal,
		Matchup:  matchupVal,
		Payload:  cloneMap(payload),
		Event:    "match_status_configuring",
	}
	inserted, err := p.upsertWebhookMatch(ctx, fresh, now, func(existing *models.WebhookMatch) {
		// Enrichment must not reset a closed canonical match back to live.
		if existing.Status != "closed" {
			existing.Status = "live"
		}
		mergeMatchDetails(existing, mapVal, matchupVal, payload)
	})
	if err != nil {
		slog.Warn("could not enrich canonical match", "error", err, "streamer", streamerID, "matchId", matchID)
		return nil
	}
	if inserted {
		return p.syncPlayerGameCount(ctx, streamerID)
	}
	return nil
}

// GetLiveMatch returns the running match document for streamerID.
func (p *PostgresStore) GetLiveMatch(ctx context.Context, streamerID string) (map[string]interface{}, bool, error) {
	var document []byte
	err := p.pool.QueryRow(ctx, `SELECT document FROM matchrooms WHERE streamer = $1 AND is_running`, streamerID).Scan(&document)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	doc, err := decodeJSONMap(document)
	if err != nil {
		return nil, false, err
	}
	return doc, true, nil
}

// CloseLiveMatch closes the running match of streamerID and marks its canonical records closed.
func (p *PostgresStore) CloseLiveMatch(ctx context.Context, streamerID string) error {
	now := time.Now().UTC()
	tag, err := p.pool.Exec(ctx,
		`UPDATE matchrooms SET is_running = false, updated_at = $2 WHERE streamer = $1 AND is_running`, streamerID, now)
	if err != nil {
		slog.Error("failed to close live match in postgres", "streamer", streamerID, "error", err)
		return err
	}
	if tag.RowsAffected() == 0 {
		slog.Info("no active match found in db to close for streamer", "streamer", streamerID)
	}

	_, err = p.pool.Exec(ctx,
		`UPDATE webhook_matches SET status = 'closed', last_seen_at = $2 WHERE player_id = $1 AND status = 'live'`, streamerID, now)
	return err
}

func (p *PostgresStore) GetAllLiveMatches(ctx context.Context) (map[string]map[string]interface{}, error) {
	rows, err := p.pool.Query(ctx, `SELECT streamer, document FROM matchrooms WHERE is_running`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]map[string]interface{})
	for rows.Next() {
		var (
			streamer string
			document []byte
		)
		if err := rows.Scan(&streamer, &document); err != nil {
			return nil, err
		}
		doc, err := decodeJSONMap(document)
		if err != nil {
			return nil, err
		}
		result[streamer] = doc
	}
	return result, rows.Err()
}

func (p *PostgresStore) CountActiveMatches(ctx context.Context) (int, error) {
	var count int
	err := p.pool.QueryRow(ctx, `SELECT count(*) FROM matchrooms WHERE is_running`).Scan(&count)
	return count, err
}

func (p *PostgresStore) SavePlayer(ctx context.Context, insertType string, playerData map[string]interface{}) error {
	document, err := jsonParam(playerData)
	if err != nil {
		return err
	}
	if document == nil {
		document = "{}"
	}
	_, err = p.pool.Exec(ctx,
		`INSERT INTO player_lookups (id, insert_type, inserted_at, document) VALUES ($1, $2, $3, $4)`,
		newUUID(), insertType, time.Now().UTC(), document)
	if err != nil {
		slog.Error("failed to save player in postgres", "error", err)
		return err
	}
	return nil
}

// RecordWebhook stores the raw event, upserts the tracked player and maintains
// the canonical match record of that player.
func (p *PostgresStore) RecordWebhook(ctx context.Context, playerID string, resolvedUsername string, body map[string]interface{}) error {
	now := time.Now().UTC()
	event := getNestedString(body, "event")
	matchID := getNestedString(body, "payload", "id")

	rawBody, err := jsonParam(body)
	if err == nil {
		_, err = p.pool.Exec(ctx,
			`INSERT INTO webhook_events (id, player_id, match_id, event, received_at, raw_body) VALUES ($1, $2, $3, $4, $5, $6)`,
			newUUID(), playerID, matchID, event, now, rawBody)
	}
	if err != nil {
		slog.Warn("could not persist raw webhook event", "error", err, "player", playerID)
	}

	knownUsername := resolvedUsername != "" && resolvedUsername != playerID
	_, err = p.pool.Exec(ctx,
		`INSERT INTO webhook_players (id, username, game_count, last_webhook_at) VALUES ($1, $2, 0, $3)
		 ON CONFLICT (id) DO UPDATE SET
			last_webhook_at = EXCLUDED.last_webhook_at,
			username = CASE WHEN $4::boolean THEN EXCLUDED.username ELSE webhook_players.username END`,
		playerID, resolvedUsername, now, knownUsername)
	if err != nil {
		return fmt.Errorf("failed to upsert webhook player: %w", err)
	}

	if matchID != "" {
		status := "live"
		if event != "match_status_configuring" && event != "match_object_created" && event != "" {
			status = "closed"
		}
		payload, _ := body["payload"].(map[string]interface{})
		mapVal := body["map"]
		matchupVal := body["matchup"]

		fresh := models.WebhookMatch{
			PlayerID: playerID,
			Username: resolvedUsername,
			MatchID:  matchID,
			Status:   status,
			Map:      mapVal,
			Matchup:  matchupVal,
			Payload:  cloneMap(payload),
			Event:    event,
		}
		_, err := p.upsertWebhookMatch(ctx, fresh, now, func(existing *models.WebhookMatch) {
			existing.Event = event
			// A closed match stays closed on delayed configuring retries.
			if existing.Status != "closed" {
				existing.Status = status
			}
			if knownUsername {
				existing.Username = resolvedUsername
			}
			mergeMatchDetails(existing, mapVal, matchupVal, payload)
		})
		if err != nil {
			return fmt.Errorf("failed to store canonical match: %w", err)
		}
	}

	return p.syncPlayerGameCount(ctx, playerID)
}

// upsertWebhookMatch inserts fresh if the player has no record of the match yet.
// Otherwise merge is applied to the locked existing record. Reports whether a row was inserted.
func (p *PostgresStore) upsertWebhookMatch(ctx context.Context, fresh models.WebhookMatch, now time.Time, merge func(*models.WebhookMatch)) (bool, error) {
	mapParam, err := jsonParam(fresh.Map)
	if err != nil {
		return false, err
	}
	matchupParam, err := jsonParam(fresh.Matchup)
	if err != nil {
		return false, err
	}
	payloadParam, err := jsonParam(fresh.Payload)
	if err != nil {
		return false, err
	}

	inserted := false
	err = pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`INSERT INTO webhook_matches (`+webhookMatchColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $8, $9, $10)
			 ON CONFLICT DO NOTHING`,
			fmt.Sprintf("%s:%s", fresh.PlayerID, fresh.MatchID), fresh.PlayerID, fresh.Username, fresh.MatchID, fresh.Status,
			now, mapParam, matchupParam, payloadParam, fresh.Event)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 1 {
			inserted = true
			return nil
		}

		existing, err := scanWebhookMatch(tx.QueryRow(ctx,
			`SELECT `+webhookMatchColumns+` FROM webhook_matches WHERE player_id = $1 AND match_id = $2 FOR UPDATE`,
			fresh.PlayerID, fresh.MatchID))
		if err != nil {
			return err
		}
		merge(&existing)

		mapParam, err := jsonParam(existing.Map)
		if err != nil {
			return err
		}
		matchupParam, err := jsonParam(existing.Matchup)
		if err != nil {
			return err
		}
		payloadParam, err := jsonParam(existing.Payload)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`UPDATE webhook_matches SET username = $2, status = $3, last_seen_at = $4, map = $5, matchup = $6, payload = $7, event = $8
			 WHERE id = $1`,
			existing.ID, existing.Username, existing.Status, now, mapParam, matchupParam, payloadParam, existing.Event)
		return err
	})
	return inserted, err
}

func mergeMatchDetails(existing *models.WebhookMatch, mapVal, matchupVal interface{}, payload map[string]interface{}) {
	if mapVal != nil && mapVal != "" && mapVal != "unknown" {
		existing.Map = mapVal
	}
	if matchupVal != nil {
		existing.Matchup = matchupVal
	}
	// Sparse closing events must not replace a payload that already has team details.
	if hasTeams(payload) || !hasTeams(existing.Payload) {
		existing.Payload = cloneMap(payload)
	}
}

func (p *PostgresStore) syncPlayerGameCount(ctx context.Context, playerID string) error {
	// The history only grows. Concurrent deliveries must not replace a newer
	// count with an older snapshot, and retries repair interrupted count updates.
	_, err := p.pool.Exec(ctx,
		`UPDATE webhook_players SET game_count = GREATEST(game_count,
			(SELECT count(*) FROM webhook_matches WHERE player_id = $1 AND match_id <> ''))
		 WHERE id = $1`, playerID)
	return err
}

// GetWebhookPlayer returns a single tracked player by ID.
func (p *PostgresStore) GetWebhookPlayer(ctx context.Context, playerID string) (*models.WebhookPlayer, error) {
	player, err := scanWebhookPlayer(p.pool.QueryRow(ctx,
		`SELECT id, username, game_count, last_webhook_at FROM webhook_players WHERE id = $1`, playerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &player, nil
}

// GetWebhookPlayers returns all tracked players, most recent webhook first.
func (p *PostgresStore) GetWebhookPlayers(ctx context.Context) ([]models.WebhookPlayer, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT id, username, game_count, last_webhook_at FROM webhook_players ORDER BY last_webhook_at DESC, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	players := make([]models.WebhookPlayer, 0)
	for rows.Next() {
		player, err := scanWebhookPlayer(rows)
		if err != nil {
			return nil, err
		}
		players = append(players, player)
	}
	return players, rows.Err()
}

// GetPlayerMatches returns paginated match history for a given player, newest first.
func (p *PostgresStore) GetPlayerMatches(ctx context.Context, playerID string, limit, offset int) ([]models.WebhookMatch, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	err := p.pool.QueryRow(ctx,
		`SELECT count(*) FROM webhook_matches WHERE player_id = $1 AND match_id <> ''`, playerID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	matches, err := p.queryWebhookMatches(ctx,
		`SELECT `+webhookMatchColumns+` FROM webhook_matches WHERE player_id = $1 AND match_id <> ''
		 ORDER BY last_seen_at DESC, id DESC LIMIT $2 OFFSET $3`, playerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return matches, total, nil
}

// GetAdminMatches returns running matches and the last 20 distinct completed matches.
func (p *PostgresStore) GetAdminMatches(ctx context.Context) (live []models.WebhookMatch, past []models.WebhookMatch, err error) {
	live = make([]models.WebhookMatch, 0)

	allLive, err := p.GetAllLiveMatches(ctx)
	if err != nil {
		return nil, nil, err
	}

	for streamerID, gameMap := range allLive {
		matchID := getNestedString(gameMap, "payload", "id")
		canonical, findErr := scanWebhookMatch(p.pool.QueryRow(ctx,
			`SELECT `+webhookMatchColumns+` FROM webhook_matches WHERE player_id = $1 AND match_id = $2`, streamerID, matchID))
		if findErr == nil {
			canonical.Status = "live"
			if canonical.Map == nil || canonical.Map == "" {
				canonical.Map = gameMap["map"]
			}
			if canonical.Matchup == nil {
				canonical.Matchup = gameMap["matchup"]
			}
			live = append(live, canonical)
			continue
		}
		if !errors.Is(findErr, pgx.ErrNoRows) {
			return nil, nil, findErr
		}

		streamerName := streamerID
		if s, sErr := p.GetStreamer(ctx, streamerID); sErr == nil && s.Name != "" {
			streamerName = s.Name
		}
		now := formatTimestamp(time.Now())
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

	sort.Slice(live, func(i, j int) bool {
		if live[i].LastSeenAt != live[j].LastSeenAt {
			return live[i].LastSeenAt > live[j].LastSeenAt
		}
		return live[i].ID < live[j].ID
	})

	past, err = p.queryWebhookMatches(ctx,
		`SELECT `+webhookMatchColumns+` FROM (
			SELECT DISTINCT ON (match_id) `+webhookMatchColumns+` FROM webhook_matches
			WHERE status = 'closed' AND match_id <> ''
			ORDER BY match_id, last_seen_at DESC, id DESC
		 ) latest
		 ORDER BY last_seen_at DESC, id DESC LIMIT 20`)
	if err != nil {
		return live, nil, err
	}
	return live, past, nil
}

func (p *PostgresStore) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func (p *PostgresStore) Close(ctx context.Context) error {
	p.pool.Close()
	return nil
}

func (p *PostgresStore) queryWebhookMatches(ctx context.Context, sql string, args ...interface{}) ([]models.WebhookMatch, error) {
	rows, err := p.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := make([]models.WebhookMatch, 0)
	for rows.Next() {
		match, err := scanWebhookMatch(rows)
		if err != nil {
			return nil, err
		}
		matches = append(matches, match)
	}
	return matches, rows.Err()
}

func scanWebhookMatch(row pgx.Row) (models.WebhookMatch, error) {
	var (
		m                       models.WebhookMatch
		firstSeenAt, lastSeenAt time.Time
		mapVal, matchup, body   []byte
	)
	if err := row.Scan(&m.ID, &m.PlayerID, &m.Username, &m.MatchID, &m.Status, &firstSeenAt, &lastSeenAt,
		&mapVal, &matchup, &body, &m.Event); err != nil {
		return m, err
	}
	m.FirstSeenAt = formatTimestamp(firstSeenAt)
	m.LastSeenAt = formatTimestamp(lastSeenAt)

	var err error
	if m.Map, err = decodeJSON(mapVal); err != nil {
		return m, err
	}
	if m.Matchup, err = decodeJSON(matchup); err != nil {
		return m, err
	}
	if m.Payload, err = decodeJSONMap(body); err != nil {
		return m, err
	}
	if m.Payload == nil {
		m.Payload = map[string]interface{}{}
	}
	return m, nil
}

func scanWebhookPlayer(row pgx.Row) (models.WebhookPlayer, error) {
	var (
		player        models.WebhookPlayer
		lastWebhookAt time.Time
	)
	if err := row.Scan(&player.ID, &player.Username, &player.GameCount, &lastWebhookAt); err != nil {
		return player, err
	}
	player.LastWebhookAt = formatTimestamp(lastWebhookAt)
	return player, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// jsonParam encodes v for a jsonb column; nil becomes SQL NULL.
func jsonParam(v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	if m, ok := v.(map[string]interface{}); ok && m == nil {
		return nil, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	// jsonb cannot store NUL characters, so strip them from all strings.
	if bytes.Contains(data, []byte(`\u0000`)) {
		decoded, err := decodeJSON(data)
		if err != nil {
			return nil, err
		}
		if data, err = json.Marshal(stripNUL(decoded)); err != nil {
			return nil, err
		}
	}
	return string(data), nil
}

func stripNUL(v interface{}) interface{} {
	switch val := v.(type) {
	case string:
		return strings.ReplaceAll(val, "\x00", "")
	case map[string]interface{}:
		clean := make(map[string]interface{}, len(val))
		for k, item := range val {
			clean[strings.ReplaceAll(k, "\x00", "")] = stripNUL(item)
		}
		return clean
	case []interface{}:
		clean := make([]interface{}, len(val))
		for i, item := range val {
			clean[i] = stripNUL(item)
		}
		return clean
	default:
		return val
	}
}

// decodeJSON decodes jsonb keeping numbers exact so values are echoed unchanged.
func decodeJSON(data []byte) (interface{}, error) {
	if data == nil {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var v interface{}
	if err := decoder.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

func decodeJSONMap(data []byte) (map[string]interface{}, error) {
	v, err := decodeJSON(data)
	if err != nil || v == nil {
		return nil, err
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected json object, got %T", v)
	}
	return m, nil
}

// cloneMap deep-copies a JSON-like map so stored documents never alias caller data.
func cloneMap(m map[string]interface{}) map[string]interface{} {
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
	clone, err := decodeJSONMap(data)
	if err != nil {
		return m
	}
	return clone
}

func parseTimestamp(value string) time.Time {
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t.UTC()
	}
	return time.Now().UTC()
}

func formatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// sanitizeText makes arbitrary request data storable in a text column.
func sanitizeText(value string) string {
	return strings.ToValidUTF8(strings.ReplaceAll(value, "\x00", ""), "\uFFFD")
}
