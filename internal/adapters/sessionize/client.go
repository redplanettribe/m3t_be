package sessionize

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"multitrackticketing/internal/domain"
)

type sessionizeHTTPFetcher struct {
	client *http.Client
}

// NewHTTPFetcher returns a fetcher that calls the Sessionize API.
func NewHTTPFetcher(client *http.Client) domain.SessionFetcher {
	if client == nil {
		client = http.DefaultClient
	}
	return &sessionizeHTTPFetcher{client: client}
}

func (f *sessionizeHTTPFetcher) Fetch(ctx context.Context, sessionizeID string) (domain.SessionFetcherResponse, error) {
	url := fmt.Sprintf("https://sessionize.com/api/v2/%s/view/All", sessionizeID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return domain.SessionFetcherResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return domain.SessionFetcherResponse{}, fmt.Errorf("failed to fetch from sessionize: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.SessionFetcherResponse{}, fmt.Errorf("sessionize api returned status: %d", resp.StatusCode)
	}

	var data domain.SessionFetcherResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return domain.SessionFetcherResponse{}, fmt.Errorf("failed to decode sessionize response: %w", err)
	}
	return data, nil
}
