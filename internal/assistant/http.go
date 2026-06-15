package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func postJSON(ctx context.Context, client httpClient, endpoint string, apiKey string, payload any, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("local AI endpoint returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	if err := json.Unmarshal(respBody, target); err != nil {
		return fmt.Errorf("parse local AI response: %w", err)
	}
	return nil
}

func getJSON(ctx context.Context, client httpClient, endpoint string, apiKey string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("local AI endpoint returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	if err := json.Unmarshal(respBody, target); err != nil {
		return fmt.Errorf("parse local AI response: %w", err)
	}
	return nil
}

func newDefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 90 * time.Second}
}
