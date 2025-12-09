package usecase

import "time"

type SessionizeResponse []SessionizeDateGrid

type SessionizeDateGrid struct {
	Date  string           `json:"date"`
	Rooms []SessionizeRoom `json:"rooms"`
}

type SessionizeRoom struct {
	ID       int                 `json:"id"`
	Name     string              `json:"name"`
	Sessions []SessionizeSession `json:"sessions"`
}

type SessionizeSession struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	StartsAt    time.Time `json:"startsAt"`
	EndsAt      time.Time `json:"endsAt"`
	RoomID      int       `json:"roomId"`
}
