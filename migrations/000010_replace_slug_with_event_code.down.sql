ALTER TABLE events
    ADD COLUMN slug VARCHAR(255);

-- Reconstruct slug from event_code for existing rows (best-effort)
UPDATE events
SET slug = event_code
WHERE slug IS NULL;

ALTER TABLE events
    ADD CONSTRAINT events_slug_unique UNIQUE (slug);

ALTER TABLE events
    DROP CONSTRAINT IF EXISTS events_event_code_unique;

ALTER TABLE events
    DROP COLUMN event_code;

