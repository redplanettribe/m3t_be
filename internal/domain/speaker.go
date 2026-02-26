package domain

import "time"

// Speaker represents a speaker at an event (imported from Sessionize or created manually).
// swagger:model Speaker
type Speaker struct {
	ID               string    `json:"id"`
	EventID          string    `json:"event_id"`
	SourceSessionID  string    `json:"source_session_id"`
	Source           string    `json:"source"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	FullName         string    `json:"full_name"`
	Bio              string    `json:"bio"`
	TagLine          string    `json:"tag_line"`
	ProfilePicture   string    `json:"profile_picture"`
	IsTopSpeaker     bool      `json:"is_top_speaker"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// NewSpeaker returns a new Speaker with the given fields. ID is typically set by the repository on create.
func NewSpeaker(eventID, sourceSessionID, source, firstName, lastName, fullName, bio, tagLine, profilePicture string, isTopSpeaker bool, createdAt, updatedAt time.Time) *Speaker {
	return &Speaker{
		EventID:         eventID,
		SourceSessionID: sourceSessionID,
		Source:          source,
		FirstName:       firstName,
		LastName:        lastName,
		FullName:        fullName,
		Bio:             bio,
		TagLine:         tagLine,
		ProfilePicture:  profilePicture,
		IsTopSpeaker:    isTopSpeaker,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
}
