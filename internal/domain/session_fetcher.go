package domain

import (
	"context"
	"time"
)

// SessionFetcher fetches schedule data from Session (or a test double).
type SessionFetcher interface {
	Fetch(ctx context.Context, sessionizeID string) (SessionFetcherResponse, error)
}

// SessionFetcherResponse is the Session GridSmart API response shape.
type SessionFetcherResponse []ScheduleDateGrid

// ScheduleDateGrid represents one date's grid of rooms/sessions.
type ScheduleDateGrid struct {
	Date  string               `json:"date"`
	Rooms []SessionFetcherRoom `json:"rooms"`
}

// SessionFetcherRoom is a room in the Session response.
type SessionFetcherRoom struct {
	ID       int                     `json:"id"`
	Name     string                  `json:"name"`
	Sessions []SessionFetcherSession `json:"sessions"`
}

// TagItem is a single tag/category item in Session.
type TagItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// SessionCategory is a category group in the Session response (e.g. "Event tag", "Tipo de sesión").
type SessionCategory struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	CategoryItems []TagItem `json:"categoryItems"`
	Sort          int       `json:"sort"`
}

// SessionFetcherSession is a session in the Session response.
type SessionFetcherSession struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description *string           `json:"description"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
	RoomID      int               `json:"roomId"`
	Categories  []SessionCategory `json:"categories"`
}
