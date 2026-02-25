-- Event invitations: track emails invited to register for an event
CREATE TABLE IF NOT EXISTS event_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_event_invitations_event_email ON event_invitations(event_id, email);
CREATE INDEX idx_event_invitations_event_id ON event_invitations(event_id);
