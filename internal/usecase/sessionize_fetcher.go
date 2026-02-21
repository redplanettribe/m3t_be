package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SessionizeFetcher fetches schedule data from Sessionize (or a test double).
type SessionizeFetcher interface {
	Fetch(ctx context.Context, sessionizeID string) (SessionizeResponse, error)
}

type sessionizeHTTPFetcher struct {
	client *http.Client
}

// NewSessionizeHTTPFetcher returns a fetcher that calls the Sessionize API.
func NewSessionizeHTTPFetcher(client *http.Client) SessionizeFetcher {
	if client == nil {
		client = http.DefaultClient
	}
	return &sessionizeHTTPFetcher{client: client}
}

func (f *sessionizeHTTPFetcher) Fetch(ctx context.Context, sessionizeID string) (SessionizeResponse, error) {
	url := fmt.Sprintf("https://sessionize.com/api/v2/%s/view/GridSmart", sessionizeID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from sessionize: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sessionize api returned status: %d", resp.StatusCode)
	}

	var data SessionizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode sessionize response: %w", err)
	}
	return data, nil
}
