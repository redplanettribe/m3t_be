package domain

import (
	"context"
	"time"
)

// SessionFetcher fetches schedule data from Sessionize (or a test double).
type SessionFetcher interface {
	Fetch(ctx context.Context, sessionizeID string) (SessionFetcherResponse, error)
}

// SessionFetcherResponse is the Sessionize All API response shape.
type SessionFetcherResponse struct {
	Sessions   []SessionFetcherSession  `json:"sessions"`
	Speakers   []SessionFetcherSpeaker  `json:"speakers"`
	Rooms      []SessionFetcherRoom     `json:"rooms"`
	Categories []SessionFetcherCategory `json:"categories"`
}

// SessionFetcherRoom is a room in the Sessionize All response (flat list).
type SessionFetcherRoom struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Sort int    `json:"sort"`
}

// SessionFetcherSession is a session in the Sessionize All response.
type SessionFetcherSession struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	StartsAt     time.Time `json:"startsAt"`
	EndsAt       time.Time `json:"endsAt"`
	Speakers     []string  `json:"speakers"`
	CategoryItems []int    `json:"categoryItems"`
	RoomID       int       `json:"roomId"`
}

// SessionFetcherSpeaker is a speaker in the Sessionize All response.
type SessionFetcherSpeaker struct {
	ID             string `json:"id"`
	FirstName      string `json:"firstName"`
	LastName       string `json:"lastName"`
	FullName       string `json:"fullName"`
	Bio            string `json:"bio"`
	TagLine        string `json:"tagLine"`
	ProfilePicture string `json:"profilePicture"`
	IsTopSpeaker   bool   `json:"isTopSpeaker"`
}

// SessionFetcherCategoryItem is a single category item in the Sessionize All response.
type SessionFetcherCategoryItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Sort int    `json:"sort"`
}

// SessionFetcherCategory is a category group in the Sessionize All response.
type SessionFetcherCategory struct {
	ID    int                        `json:"id"`
	Title string                     `json:"title"`
	Items []SessionFetcherCategoryItem `json:"items"`
	Sort  int                        `json:"sort"`
	Type  string                     `json:"type"`
}
