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
}
