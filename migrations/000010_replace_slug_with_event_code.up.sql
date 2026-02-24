ALTER TABLE events
    ADD COLUMN event_code CHAR(4);

-- Backfill event_code for existing rows using a simple deterministic code based on id
UPDATE events
SET event_code = SUBSTRING(REPLACE(id::text, '-', ''), 1, 4)
WHERE event_code IS NULL;

ALTER TABLE events
    ADD CONSTRAINT events_event_code_unique UNIQUE (event_code);

ALTER TABLE events
    DROP COLUMN slug;
