package domain

import "context"

// Tag represents a named tag shared across events and sessions.
// swagger:model Tag
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TagRepository defines storage for tags and event/session–tag links.
type TagRepository interface {
	// EnsureTagForEvent resolves a tag by name (creating it if missing), ensures the event has the tag in event_tags, and returns the tag ID.
	EnsureTagForEvent(ctx context.Context, eventID, tagName string) (tagID string, err error)
	// SetSessionTags replaces all tag links for the given session with the given tag IDs.
	SetSessionTags(ctx context.Context, sessionID string, tagIDs []string) error
	// ListTagsByEventID returns all tags associated with the given event via event_tags.
	ListTagsByEventID(ctx context.Context, eventID string) ([]*Tag, error)
	// AddSessionTag links a tag to a session (idempotent; no-op if already linked).
	AddSessionTag(ctx context.Context, sessionID, tagID string) error
	// RemoveSessionTag unlinks a tag from a session.
	RemoveSessionTag(ctx context.Context, sessionID, tagID string) error
	// UpdateTagName updates the tag name by ID. Returns ErrNotFound if tag does not exist.
	UpdateTagName(ctx context.Context, tagID, name string) error
	// GetTagByID returns the tag by ID, or ErrNotFound if not found.
	GetTagByID(ctx context.Context, tagID string) (*Tag, error)
}
