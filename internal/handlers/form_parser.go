package handlers

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// ParseNestedForm parses URL-encoded query or form data with extended bracket syntax (qs style):
// e.g. payload[entity][type]=matchmaking
//
//	players[0][nickname]=edox
func ParseNestedForm(data string) (map[string]interface{}, error) {
	values, err := url.ParseQuery(data)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for key, values := range values {
		tokens := tokenizeKey(key)
		if len(tokens) > 32 {
			return nil, fmt.Errorf("form nesting exceeds 32 levels")
		}
		for _, value := range values {
			setNestedValue(result, tokens, value)
		}
	}
	for key, child := range result {
		result[key] = convertNumericMapsToSlices(child)
	}
	return result, nil
}

func tokenizeKey(key string) []string {
	var tokens []string
	firstBracket := strings.Index(key, "[")
	if firstBracket == -1 {
		return []string{key}
	}

	tokens = append(tokens, key[:firstBracket])
	remaining := key[firstBracket:]

	for len(remaining) > 0 {
		if remaining[0] == '[' {
			end := strings.Index(remaining, "]")
			if end == -1 {
				tokens = append(tokens, remaining[1:])
				break
			}
			tokens = append(tokens, remaining[1:end])
			remaining = remaining[end+1:]
		} else {
			remaining = remaining[1:]
		}
	}
	return tokens
}

func setNestedValue(current map[string]interface{}, tokens []string, value string) {
	if len(tokens) == 0 {
		return
	}

	head := tokens[0]
	if head == "" {
		// qs merges unindexed object fields into one array item, while scalar
		// array values append. Numeric maps are compacted after parsing.
		head = "0"
		if len(tokens) == 1 {
			for current[head] != nil {
				n, _ := strconv.Atoi(head)
				head = strconv.Itoa(n + 1)
			}
		}
	}
	if len(tokens) == 1 {
		if previous, exists := current[head]; exists {
			if list, ok := previous.([]interface{}); ok {
				current[head] = append(list, value)
			} else {
				current[head] = []interface{}{previous, value}
			}
		} else {
			current[head] = value
		}
		return
	}

	next, exists := current[head]
	if !exists || next == nil {
		child := make(map[string]interface{})
		current[head] = child
		setNestedValue(child, tokens[1:], value)
		return
	}

	childMap, ok := next.(map[string]interface{})
	if !ok {
		childMap = make(map[string]interface{})
		current[head] = childMap
	}
	setNestedValue(childMap, tokens[1:], value)
}

// convertNumericMapsToSlices compacts qs array indices, including sparse arrays.
func convertNumericMapsToSlices(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		if len(val) == 0 {
			return val
		}

		indices := make([]int, 0, len(val))
		allNumeric := true
		for key := range val {
			n, err := strconv.Atoi(key)
			if err != nil || n < 0 || n > 20 || strconv.Itoa(n) != key {
				allNumeric = false
				break
			}
			indices = append(indices, n)
		}
		if allNumeric {
			sort.Ints(indices)
			items := make([]interface{}, 0, len(indices))
			for _, index := range indices {
				items = append(items, convertNumericMapsToSlices(val[strconv.Itoa(index)]))
			}
			return items
		}

		for k, child := range val {
			val[k] = convertNumericMapsToSlices(child)
		}
		return val

	case []interface{}:
		for i, child := range val {
			val[i] = convertNumericMapsToSlices(child)
		}
		return val

	default:
		return v
	}
}

// Helper to format URL error messages
func validateURL(rawURL string) error {
	if len(rawURL) > 512 {
		return fmt.Errorf("url exceeds maximum length of 512 characters")
	}
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url scheme must be http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("url host cannot be empty")
	}
	if u.User != nil {
		return fmt.Errorf("url credentials are not permitted")
	}
	return nil
}
