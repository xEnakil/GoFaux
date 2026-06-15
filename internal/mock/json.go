package mock

import (
	"encoding/json"
	"errors"
	"strings"
)

func ValidateJSON(s string) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil, errors.New("JSON cannot be empty")
	}
	if !json.Valid([]byte(trimmed)) {
		return nil, errors.New("invalid JSON syntax")
	}
	return json.RawMessage(trimmed), nil
}

func PrettyJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return string(raw)
	}

	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(out)
}
