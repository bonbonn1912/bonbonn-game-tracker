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
	ID      string `json:"id" bson:"_id"`
	Name    string `json:"name" bson:"name"`
	URL     string `json:"url" bson:"url"`
	Enabled bool   `json:"enabled" bson:"enabled"`
}

// RedirectEvent records a redirect HTTP request event for auditing and stats.
type RedirectEvent struct {
	ID           string `json:"id" bson:"_id"`
	StreamerID   string `json:"streamerId" bson:"streamerId"`
	StreamerName string `json:"streamerName" bson:"streamerName"`
	MatchID      string `json:"matchId" bson:"matchId"`
	Target       string `json:"target" bson:"target"`
	Outcome      string `json:"outcome" bson:"outcome"`
	CreatedAt    string `json:"createdAt" bson:"createdAt"`
	RequestID    string `json:"requestId" bson:"requestId"`
	IP           string `json:"ip,omitempty" bson:"ip,omitempty"`
	UserAgent    string `json:"userAgent,omitempty" bson:"userAgent,omitempty"`
}

// AdminStatus represents current system and database status.
type AdminStatus struct {
	Database      string `json:"database"`
	ActiveMatches int    `json:"activeMatches"`
}

// WebhookPlayer represents a player recognized from incoming webhooks.
type WebhookPlayer struct {
	ID            string `json:"id" bson:"_id"`
	Username      string `json:"username" bson:"username"`
	GameCount     int    `json:"gameCount" bson:"gameCount"`
	LastWebhookAt string `json:"lastWebhookAt" bson:"lastWebhookAt"`
}

// WebhookMatch represents a canonical match record for a player.
type WebhookMatch struct {
	ID          string                 `json:"id" bson:"_id"`
	PlayerID    string                 `json:"playerId" bson:"playerId"`
	Username    string                 `json:"username" bson:"username"`
	MatchID     string                 `json:"matchId" bson:"matchId"`
	Status      string                 `json:"status" bson:"status"`
	FirstSeenAt string                 `json:"firstSeenAt" bson:"firstSeenAt"`
	LastSeenAt  string                 `json:"lastSeenAt" bson:"lastSeenAt"`
	Map         interface{}            `json:"map" bson:"map"`
	Matchup     interface{}            `json:"matchup" bson:"matchup"`
	Payload     map[string]interface{} `json:"payload" bson:"payload"`
	Event       string                 `json:"event" bson:"event"`
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
	ID        string                 `json:"id" bson:"_id"`
	PlayerID  string                 `json:"playerId" bson:"playerId"`
	MatchID   string                 `json:"matchId" bson:"matchId"`
	Event     string                 `json:"event" bson:"event"`
	Timestamp string                 `json:"timestamp" bson:"timestamp"`
	RawBody   map[string]interface{} `json:"rawBody" bson:"rawBody"`
}

// FaceitPlayer represents the core Faceit player structure returned from Faceit v4 API.
type FaceitPlayer struct {
	PlayerID     string                 `json:"player_id" bson:"player_id"`
	Nickname     string                 `json:"nickname" bson:"nickname"`
	Avatar       string                 `json:"avatar" bson:"avatar"`
	Country      string                 `json:"country" bson:"country"`
	CoverImage   string                 `json:"cover_image" bson:"cover_image"`
	Platforms    map[string]interface{} `json:"platforms" bson:"platforms"`
	Games        FaceitGames            `json:"games" bson:"games"`
	MatchHistory []string               `json:"matchHistory,omitempty" bson:"matchHistory,omitempty"`
	InsertType   string                 `json:"insertType,omitempty" bson:"insertType,omitempty"`
	Local        *FaceitEloInfo         `json:"local,omitempty" bson:"local,omitempty"`
}

// FaceitGames contains CSGO and CS2 game data.
type FaceitGames struct {
	CSGO *FaceitGameDetail `json:"csgo,omitempty" bson:"csgo,omitempty"`
	CS2  *FaceitGameDetail `json:"cs2,omitempty" bson:"cs2,omitempty"`
}

// FaceitGameDetail holds elo and level info for a specific game.
type FaceitGameDetail struct {
	Region         string `json:"region" bson:"region"`
	GamePlayerID   string `json:"game_player_id" bson:"game_player_id"`
	SkillLevel     int    `json:"skill_level" bson:"skill_level"`
	FaceitElo      int    `json:"faceit_elo" bson:"faceit_elo"`
	GamePlayerName string `json:"game_player_name" bson:"game_player_name"`
}

// FaceitEloInfo contains calculated rank information.
type FaceitEloInfo struct {
	Username       string `json:"username" bson:"username"`
	IsMax          bool   `json:"isMax" bson:"isMax"`
	Level          int    `json:"level" bson:"level"`
	Elo            int    `json:"elo" bson:"elo"`
	CS2Elo         *int   `json:"cs2elo,omitempty" bson:"cs2elo,omitempty"`
	EloToNextLevel int    `json:"eloToNextLevel" bson:"eloToNextLevel"`
	ResponseString string `json:"responseString" bson:"responseString"`
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
