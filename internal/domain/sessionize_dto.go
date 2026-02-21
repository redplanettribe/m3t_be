package domain

import "time"

// SessionizeResponse is the Sessionize GridSmart API response shape.
type SessionizeResponse []SessionizeDateGrid

// SessionizeDateGrid represents one date's grid of rooms/sessions.
type SessionizeDateGrid struct {
	Date  string           `json:"date"`
	Rooms []SessionizeRoom `json:"rooms"`
}

// SessionizeRoom is a room in the Sessionize response.
type SessionizeRoom struct {
	ID       int                 `json:"id"`
	Name     string              `json:"name"`
	Sessions []SessionizeSession `json:"sessions"`
}

// SessionizeSession is a session in the Sessionize response.
type SessionizeSession struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	StartsAt    time.Time `json:"startsAt"`
	EndsAt      time.Time `json:"endsAt"`
	RoomID      int       `json:"roomId"`
}
