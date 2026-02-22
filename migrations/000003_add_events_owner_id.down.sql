DROP INDEX IF EXISTS idx_events_owner_id;
ALTER TABLE events DROP COLUMN owner_id;
