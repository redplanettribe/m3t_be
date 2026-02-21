package domain

import "context"

// SessionizeFetcher fetches schedule data from Sessionize (or a test double).
type SessionizeFetcher interface {
	Fetch(ctx context.Context, sessionizeID string) (SessionizeResponse, error)
}
