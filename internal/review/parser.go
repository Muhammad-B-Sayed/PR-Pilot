package review

import (
	"encoding/json"
	"errors"
	"strings"
)

var ErrInvalidModelJSON = errors.New("model returned invalid JSON")

func ParseJSON[T any](raw string) (T, error) {
	var out T
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "```") {
		cleaned = trimFence(cleaned)
	}
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		return out, errors.Join(ErrInvalidModelJSON, err)
	}
	return out, nil
}

func trimFence(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "```json")
	value = strings.TrimPrefix(value, "```")
	value = strings.TrimSuffix(value, "```")
	return strings.TrimSpace(value)
}
