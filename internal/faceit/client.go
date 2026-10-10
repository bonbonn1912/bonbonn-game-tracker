package faceit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bonbonn1912/bonbonn-game-tracker/internal/models"
)

var (
	ErrPlayerNotFound = errors.New("player not found")
	ErrMatchNotFound  = errors.New("match not found")
	ErrFaceitAPI      = errors.New("faceit api error")
)

// Client is a client for the FACEIT v4 API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new FACEIT API client.
func NewClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "https://open.faceit.com/data/v4"
	}
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// GetPlayerByNickname fetches player info by username.
func (c *Client) GetPlayerByNickname(ctx context.Context, nickname string) (*models.FaceitPlayer, error) {
	endpoint := fmt.Sprintf("%s/players?nickname=%s", c.baseURL, url.QueryEscape(nickname))
	return c.fetchPlayer(ctx, endpoint)
}

// GetPlayerByID fetches player info by faceitId or steam64Id.
func (c *Client) GetPlayerByID(ctx context.Context, id string) (*models.FaceitPlayer, error) {
	var endpoint string
	if models.RegexFaceitID.MatchString(id) {
		endpoint = fmt.Sprintf("%s/players/%s", c.baseURL, url.PathEscape(id))
	} else {
		endpoint = fmt.Sprintf("%s/players?game=csgo&game_player_id=%s", c.baseURL, url.QueryEscape(id))
	}
	return c.fetchPlayer(ctx, endpoint)
}

func (c *Client) fetchPlayer(ctx context.Context, endpoint string) (*models.FaceitPlayer, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFaceitAPI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrPlayerNotFound
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var player models.FaceitPlayer
	if err := json.Unmarshal(body, &player); err != nil {
		return nil, err
	}

	// Verify required player fields
	if player.PlayerID == "" || player.Nickname == "" {
		return nil, ErrPlayerNotFound
	}

	return &player, nil
}

// GetPlayerHistoryCS2 fetches recent match history for CS2.
func (c *Client) GetPlayerHistoryCS2(ctx context.Context, playerID string, limit string) (*models.MatchHistoryResponse, error) {
	endpoint := fmt.Sprintf("%s/players/%s/history?game=cs2&offset=0&limit=%s", c.baseURL, url.PathEscape(playerID), url.QueryEscape(limit))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFaceitAPI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrFaceitAPI
	}

	var history models.MatchHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return nil, err
	}

	return &history, nil
}

// GetMatchDetails fetches match details (used to determine selected map).
func (c *Client) GetMatchDetails(ctx context.Context, matchID string) (*models.MatchDetails, error) {
	endpoint := fmt.Sprintf("%s/matches/%s", c.baseURL, url.PathEscape(matchID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFaceitAPI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrMatchNotFound
	}

	var match models.MatchDetails
	if err := json.NewDecoder(resp.Body).Decode(&match); err != nil {
		return nil, err
	}

	return &match, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}
