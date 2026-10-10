package store

import (
	"context"
	"errors"

	"github.com/bonbonn1912/bonbonn-game-tracker/internal/models"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrAlreadyLive    = errors.New("match already live")
	ErrDuplicateKey   = errors.New("duplicate key")
	ErrInvalidPayload = errors.New("invalid payload")
)

// DefaultSeedStreamers defines the initial 8 streamers from legacy hardcoded config.
var DefaultSeedStreamers = []models.Streamer{
	{ID: "4967963b-566f-4e8a-908b-02c8315d12f1", Name: "edox", URL: "http://edox.faceitlobby.com", Enabled: true},
	{ID: "afce1ec2-1b60-47bd-88c1-bfadce96876a", Name: "danielpello", URL: "http://danielpello.faceitlobby.com", Enabled: true},
	{ID: "e3cefc97-36b1-4615-91d6-cdcca13e4e58", Name: "megalopolik", URL: "http://megalopolik.faceitlobby.com", Enabled: true},
	{ID: "f335cfd1-3a92-4365-9b25-2f0e82a6052f", Name: "mrc9cs", URL: "http://mrc9cs.faceitlobby.com", Enabled: true},
	{ID: "e64b6d73-6313-4527-b313-a852e6368cf2", Name: "redcs", URL: "http://redcs.faceitlobby.com", Enabled: true},
	{ID: "c77a24f9-4126-4321-9cc5-4dfcf69b50ee", Name: "bulmi", URL: "http://bulmi.faceitlobby.com", Enabled: true},
	{ID: "f8bd2fd4-1860-441e-8851-3469dd3a1a55", Name: "tiziaN", URL: "http://tiziaN.faceitlobby.com", Enabled: true},
	{ID: "0deeb9bd-c453-4a02-94fe-d8ca83565f0f", Name: "syrinxx", URL: "http://syrinxx.faceitlobby.com", Enabled: true},
}

// Store defines persistence methods for streamers, live games, redirect history, and players.
type Store interface {
	// Streamer configuration CRUD
	GetStreamers(ctx context.Context) ([]models.Streamer, error)
	GetStreamer(ctx context.Context, id string) (*models.Streamer, error)
	CreateStreamer(ctx context.Context, s models.Streamer) (*models.Streamer, error)
	UpdateStreamer(ctx context.Context, id string, s models.Streamer) (*models.Streamer, error)
	DeleteStreamer(ctx context.Context, id string) error

	// Redirect event / click logging
	LogRedirectEvent(ctx context.Context, event models.RedirectEvent) error
	GetRedirectEvents(ctx context.Context, limit int) ([]models.RedirectEvent, error)

	// Live matches
	AddLiveMatch(ctx context.Context, streamerID string, matchData map[string]interface{}) error
	GetLiveMatch(ctx context.Context, streamerID string) (map[string]interface{}, bool, error)
	CloseLiveMatch(ctx context.Context, streamerID string) error
	GetAllLiveMatches(ctx context.Context) (map[string]map[string]interface{}, error)
	CountActiveMatches(ctx context.Context) (int, error)

	// Player history / ELO lookup persistence
	SavePlayer(ctx context.Context, insertType string, playerData map[string]interface{}) error

	// Webhook capture, canonical matches, and admin queries
	RecordWebhook(ctx context.Context, playerID string, resolvedUsername string, body map[string]interface{}) error
	GetWebhookPlayer(ctx context.Context, playerID string) (*models.WebhookPlayer, error)
	GetWebhookPlayers(ctx context.Context) ([]models.WebhookPlayer, error)
	GetPlayerMatches(ctx context.Context, playerID string, limit, offset int) ([]models.WebhookMatch, int, error)
	GetAdminMatches(ctx context.Context) (live []models.WebhookMatch, past []models.WebhookMatch, err error)

	// Health and lifecycle
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
}
