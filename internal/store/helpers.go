package store

import (
	"crypto/rand"
	"fmt"
)

func hasTeams(payload interface{}) bool {
	p, ok := payload.(map[string]interface{})
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
