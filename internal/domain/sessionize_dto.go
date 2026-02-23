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

// SessionizeCategoryItem is a single tag/category item in Sessionize.
type SessionizeCategoryItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// SessionizeCategory is a category group in the Sessionize response (e.g. "Event tag", "Tipo de sesión").
type SessionizeCategory struct {
	ID            int                     `json:"id"`
	Name          string                  `json:"name"`
	CategoryItems []SessionizeCategoryItem `json:"categoryItems"`
	Sort          int                     `json:"sort"`
}

// SessionizeSession is a session in the Sessionize response.
type SessionizeSession struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description *string             `json:"description"`
	StartsAt    time.Time           `json:"startsAt"`
	EndsAt      time.Time           `json:"endsAt"`
	RoomID      int                 `json:"roomId"`
	Categories  []SessionizeCategory `json:"categories"`
}
