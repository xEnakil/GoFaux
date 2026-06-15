package mock

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const ConfigVersion = 2

type Definition struct {
	ID             string            `json:"id"`
	Name           string            `json:"name,omitempty"`
	Description    string            `json:"description,omitempty"`
	Method         string            `json:"method"`
	Endpoint       string            `json:"endpoint"`
	Status         int               `json:"status"`
	Headers        map[string]string `json:"headers,omitempty"`
	Query          map[string]string `json:"query,omitempty"`
	RequestHeaders map[string]string `json:"request_headers,omitempty"`
	Body           json.RawMessage   `json:"body"`
	DelayMS        int               `json:"delay_ms,omitempty"`
	Priority       int               `json:"priority,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	Requests       int               `json:"requests,omitempty"`
	CreatedAt      time.Time         `json:"created_at,omitempty"`
	UpdatedAt      time.Time         `json:"updated_at,omitempty"`
}

type Set struct {
	Version int          `json:"version"`
	Name    string       `json:"name"`
	APIs    []Definition `json:"apis"`
}

type Match struct {
	PathParams map[string]string `json:"path_params,omitempty"`
	Score      int               `json:"score"`
}

func NewDefinition(input Definition) (Definition, error) {
	def := CloneDefinition(input)
	if err := def.Normalize(); err != nil {
		return Definition{}, err
	}

	now := time.Now().UTC()
	if def.ID == "" {
		def.ID = generateID()
	}
	if def.CreatedAt.IsZero() {
		def.CreatedAt = now
	}
	def.UpdatedAt = now
	return def, nil
}

func (d *Definition) Normalize() error {
	d.Method = NormalizeMethod(d.Method)
	if d.Method == "" {
		return errors.New("method is required")
	}
	if !IsSupportedMethod(d.Method) {
		return fmt.Errorf("unsupported method %q", d.Method)
	}

	d.Endpoint = NormalizeEndpoint(d.Endpoint)
	if d.Endpoint == "" {
		return errors.New("endpoint is required")
	}

	if d.Status == 0 {
		d.Status = http.StatusOK
	}
	if d.Status < 100 || d.Status > 599 {
		return fmt.Errorf("status code %d is outside the HTTP range", d.Status)
	}

	if d.DelayMS < 0 {
		return errors.New("delay_ms cannot be negative")
	}

	if len(d.Body) == 0 {
		d.Body = json.RawMessage(`{}`)
	}
	if !json.Valid(d.Body) {
		return errors.New("body must be valid JSON")
	}

	d.Headers = normalizeMap(d.Headers)
	d.Query = normalizeMap(d.Query)
	d.RequestHeaders = normalizeHeaderMap(d.RequestHeaders)
	return nil
}

func NormalizeMethod(method string) string {
	return strings.ToUpper(strings.TrimSpace(method))
}

func IsSupportedMethod(method string) bool {
	switch NormalizeMethod(method) {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodOptions, http.MethodHead:
		return true
	default:
		return false
	}
}

func NormalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	if len(endpoint) > 1 {
		endpoint = strings.TrimRight(endpoint, "/")
	}
	return endpoint
}

func (d Definition) Summary(index int) string {
	name := d.Name
	if name == "" {
		name = "unnamed"
	}
	return fmt.Sprintf("[%d] %s %s -> %d (%s, requests: %d, priority: %d)",
		index, d.Method, d.Endpoint, d.Status, name, d.Requests, d.Priority)
}

func CloneDefinition(input Definition) Definition {
	out := input
	out.Headers = cloneStringMap(input.Headers)
	out.Query = cloneStringMap(input.Query)
	out.RequestHeaders = cloneStringMap(input.RequestHeaders)
	out.Body = cloneRaw(input.Body)
	out.Tags = append([]string(nil), input.Tags...)
	return out
}

func CloneDefinitions(input []Definition) []Definition {
	if len(input) == 0 {
		return []Definition{}
	}
	out := make([]Definition, len(input))
	for i, def := range input {
		out[i] = CloneDefinition(def)
	}
	return out
}

func normalizeMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	normalized := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		normalized[key] = strings.TrimSpace(value)
	}
	return normalized
}

func normalizeHeaderMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	normalized := make(map[string]string, len(values))
	for key, value := range values {
		key = http.CanonicalHeaderKey(strings.TrimSpace(key))
		if key == "" {
			continue
		}
		normalized[key] = strings.TrimSpace(value)
	}
	return normalized
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneRaw(input json.RawMessage) json.RawMessage {
	if len(input) == 0 {
		return nil
	}
	out := make([]byte, len(input))
	copy(out, input)
	return out
}

func generateID() string {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return "mock_" + hex.EncodeToString(buf[:])
	}
	return fmt.Sprintf("mock_%d", time.Now().UnixNano())
}
