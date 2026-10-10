package handlers

import (
	"testing"
)

func TestParseNestedForm(t *testing.T) {
	input := "event=match_status_configuring&payload%5Bentity%5D%5Btype%5D=matchmaking&payload%5Bid%5D=match123&teams%5B0%5D%5Bname%5D=TeamA&teams%5B1%5D%5Bname%5D=TeamB"
	parsed, err := ParseNestedForm(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed["event"] != "match_status_configuring" {
		t.Errorf("expected event match_status_configuring, got %v", parsed["event"])
	}

	payload, ok := parsed["payload"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected payload map, got %T", parsed["payload"])
	}
	entity, ok := payload["entity"].(map[string]interface{})
	if !ok || entity["type"] != "matchmaking" {
		t.Errorf("expected entity.type matchmaking, got %v", entity)
	}
	if payload["id"] != "match123" {
		t.Errorf("expected payload.id match123, got %v", payload["id"])
	}

	teams, ok := parsed["teams"].([]interface{})
	if !ok || len(teams) != 2 {
		t.Fatalf("expected 2 teams, got %v", parsed["teams"])
	}
	team0, _ := teams[0].(map[string]interface{})
	if team0["name"] != "TeamA" {
		t.Errorf("expected TeamA, got %v", team0["name"])
	}
}

func TestParseNestedPlayers(t *testing.T) {
	input := "players[0][nickname]=player1&players[0][steam64ID]=76561198000000001&players[0][customTag]=alpha&players[1][nickname]=player2&players[1][steam64ID]=76561198000000002"
	parsed, err := ParseNestedForm(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	players, ok := parsed["players"].([]interface{})
	if !ok || len(players) != 2 {
		t.Fatalf("expected 2 players, got %v", parsed["players"])
	}

	p0, ok := players[0].(map[string]interface{})
	if !ok || p0["nickname"] != "player1" || p0["customTag"] != "alpha" {
		t.Errorf("unexpected player 0: %v", p0)
	}
}

func TestValidateURL(t *testing.T) {
	valid := []string{
		"http://localhost:12000",
		"https://edox.faceitlobby.com",
		"http://127.0.0.1:8080/lobby",
	}
	for _, u := range valid {
		if err := validateURL(u); err != nil {
			t.Errorf("expected %s to be valid, got %v", u, err)
		}
	}

	invalid := []string{
		"ftp://example.com",
		"http://user:pass@example.com",
		"not-a-url",
		"",
	}
	for _, u := range invalid {
		if err := validateURL(u); err == nil {
			t.Errorf("expected %s to be invalid", u)
		}
	}
}

func TestFormQSArrays(t *testing.T) {
	for _, input := range []string{"players[][nickname]=one&players[][steam64ID]=123", "players[0][nickname]=one&players[2][nickname]=two"} {
		result, err := ParseNestedForm(input)
		if err != nil {
			t.Fatal(err)
		}
		players, ok := result["players"].([]interface{})
		if !ok || len(players) == 0 {
			t.Fatalf("expected compact array: %#v", result)
		}
		first := players[0].(map[string]interface{})
		if first["nickname"] != "one" {
			t.Fatalf("wrong first player: %#v", first)
		}
	}
	result, err := ParseNestedForm("roles[]=a&roles[]=b&name=one&name=two")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"roles", "name"} {
		values, ok := result[key].([]interface{})
		if !ok || len(values) != 2 {
			t.Fatalf("expected repeated values for %s: %#v", key, result)
		}
	}
}
