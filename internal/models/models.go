package models

import (
	"regexp"
)

var (
	RegexSteam64        = regexp.MustCompile(`^[0-9]{17}$`)
	RegexFaceitID       = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	RegexFaceitUsername = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,12}$`)
)

// Streamer represents a persisted streamer configuration mapping.
type Streamer struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// RedirectEvent records a redirect HTTP request event for auditing and stats.
type RedirectEvent struct {
	ID           string `json:"id"`
	StreamerID   string `json:"streamerId"`
	StreamerName string `json:"streamerName"`
	MatchID      string `json:"matchId"`
	Target       string `json:"target"`
	Outcome      string `json:"outcome"`
	CreatedAt    string `json:"createdAt"`
	RequestID    string `json:"requestId"`
	IP           string `json:"ip,omitempty"`
	UserAgent    string `json:"userAgent,omitempty"`
}

// AdminStatus represents current system and database status.
type AdminStatus struct {
	Database      string `json:"database"`
	ActiveMatches int    `json:"activeMatches"`
}

// WebhookPlayer represents a player recognized from incoming webhooks.
type WebhookPlayer struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	GameCount     int    `json:"gameCount"`
	LastWebhookAt string `json:"lastWebhookAt"`
}

// WebhookMatch represents a canonical match record for a player.
type WebhookMatch struct {
	ID          string                 `json:"id"`
	PlayerID    string                 `json:"playerId"`
	Username    string                 `json:"username"`
	MatchID     string                 `json:"matchId"`
	Status      string                 `json:"status"`
	FirstSeenAt string                 `json:"firstSeenAt"`
	LastSeenAt  string                 `json:"lastSeenAt"`
	Map         interface{}            `json:"map"`
	Matchup     interface{}            `json:"matchup"`
	Payload     map[string]interface{} `json:"payload"`
	Event       string                 `json:"event"`
}

// PlayerMatchesResponse represents the paginated match history for a player.
type PlayerMatchesResponse struct {
	Matches []WebhookMatch `json:"matches"`
	Total   int            `json:"total"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
}

// MatchesResponse represents live and recent past matches.
type MatchesResponse struct {
	Live []WebhookMatch `json:"live"`
	Past []WebhookMatch `json:"past"`
}

// WebhookEvent represents an audit record of a received webhook.
type WebhookEvent struct {
	ID        string                 `json:"id"`
	PlayerID  string                 `json:"playerId"`
	MatchID   string                 `json:"matchId"`
	Event     string                 `json:"event"`
	Timestamp string                 `json:"timestamp"`
	RawBody   map[string]interface{} `json:"rawBody"`
}

// WebhookRequest stores the HTTP details of a request received by the webhook endpoint.
type WebhookRequest struct {
	ID        string              `json:"id"`
	Timestamp string              `json:"timestamp"`
	Method    string              `json:"method"`
	Path      string              `json:"path"`
	Headers   map[string][]string `json:"headers"`
	Body      string              `json:"body"`
}

// FaceitPlayer represents the core Faceit player structure returned from Faceit v4 API.
type FaceitPlayer struct {
	PlayerID     string                 `json:"player_id"`
	Nickname     string                 `json:"nickname"`
	Avatar       string                 `json:"avatar"`
	Country      string                 `json:"country"`
	CoverImage   string                 `json:"cover_image"`
	Platforms    map[string]interface{} `json:"platforms"`
	Games        FaceitGames            `json:"games"`
	MatchHistory []string               `json:"matchHistory,omitempty"`
	InsertType   string                 `json:"insertType,omitempty"`
	Local        *FaceitEloInfo         `json:"local,omitempty"`
}

// FaceitGames contains CSGO and CS2 game data.
type FaceitGames struct {
	CSGO *FaceitGameDetail `json:"csgo,omitempty"`
	CS2  *FaceitGameDetail `json:"cs2,omitempty"`
}

// FaceitGameDetail holds elo and level info for a specific game.
type FaceitGameDetail struct {
	Region         string `json:"region"`
	GamePlayerID   string `json:"game_player_id"`
	SkillLevel     int    `json:"skill_level"`
	FaceitElo      int    `json:"faceit_elo"`
	GamePlayerName string `json:"game_player_name"`
}

// FaceitEloInfo contains calculated rank information.
type FaceitEloInfo struct {
	Username       string `json:"username"`
	IsMax          bool   `json:"isMax"`
	Level          int    `json:"level"`
	Elo            int    `json:"elo"`
	CS2Elo         *int   `json:"cs2elo,omitempty"`
	EloToNextLevel int    `json:"eloToNextLevel"`
	ResponseString string `json:"responseString"`
}

// MatchHistoryResponse represents Faceit history items response.
type MatchHistoryResponse struct {
	Items []MatchHistoryItem `json:"items"`
}

// MatchHistoryItem represents one match in history.
type MatchHistoryItem struct {
	MatchID string               `json:"match_id"`
	Results MatchHistoryResult   `json:"results"`
	Teams   MatchHistoryTeamsMap `json:"teams"`
}

type MatchHistoryResult struct {
	Winner string `json:"winner"`
}

type MatchHistoryTeamsMap struct {
	Faction1 MatchHistoryFaction `json:"faction1"`
	Faction2 MatchHistoryFaction `json:"faction2"`
}

type MatchHistoryFaction struct {
	Players []MatchHistoryPlayer `json:"players"`
}

type MatchHistoryPlayer struct {
	PlayerID string `json:"player_id"`
	Nickname string `json:"nickname"`
}

// MatchDetails represents the Faceit match response from /matches/{id}.
type MatchDetails struct {
	Voting struct {
		Map struct {
			Pick []string `json:"pick"`
		} `json:"map"`
	} `json:"voting"`
}

// ConsolePlayerInput represents input item for /internal/api/getMultiplePlayers.
type ConsolePlayerInput struct {
	Nickname  string `json:"nickname"`
	Steam64ID string `json:"steam64ID"`
	Elo       int    `json:"elo"`
}
