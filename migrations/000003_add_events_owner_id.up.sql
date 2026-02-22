-- Add owner_id to events; existing rows get the first user as owner (if any).
ALTER TABLE events ADD COLUMN owner_id UUID REFERENCES users(id) ON DELETE RESTRICT;

UPDATE events SET owner_id = (SELECT id FROM users LIMIT 1) WHERE owner_id IS NULL;

ALTER TABLE events ALTER COLUMN owner_id SET NOT NULL;

CREATE INDEX idx_events_owner_id ON events(owner_id);
