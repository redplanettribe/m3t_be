-- Generalize external source identifiers for rooms, sessions, and speakers.
-- Adds a `source` column and renames Sessionize-specific IDs to `source_session_id`.

-- Rooms: already use source_session_id, just add source column and backfill.
ALTER TABLE rooms
    ADD COLUMN IF NOT EXISTS source VARCHAR(20)
        CHECK (source IN ('sessionize', 'admin_app'));

-- Backfill existing rooms imported from Sessionize.
UPDATE rooms
SET source = 'sessionize'
WHERE source_session_id IS NOT NULL
  AND source IS NULL;

-- Sessions: rename sessionize_session_id -> source_session_id, add source column, and update unique constraint.
ALTER TABLE sessions
    DROP CONSTRAINT IF EXISTS sessions_room_id_sessionize_session_id_key;

ALTER TABLE sessions
    RENAME COLUMN sessionize_session_id TO source_session_id;

ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS source VARCHAR(20)
        CHECK (source IN ('sessionize', 'admin_app'));

-- Backfill existing sessions imported from Sessionize.
UPDATE sessions
SET source = 'sessionize'
WHERE source_session_id IS NOT NULL
  AND source IS NULL;

ALTER TABLE sessions
    ADD CONSTRAINT sessions_room_id_source_session_id_key
        UNIQUE (room_id, source_session_id);

-- Speakers: rename sessionize_speaker_id -> source_session_id, add source column, and update unique constraint.
ALTER TABLE speakers
    DROP CONSTRAINT IF EXISTS speakers_event_id_sessionize_speaker_id_key;

ALTER TABLE speakers
    RENAME COLUMN sessionize_speaker_id TO source_session_id;

ALTER TABLE speakers
    ADD COLUMN IF NOT EXISTS source VARCHAR(20)
        CHECK (source IN ('sessionize', 'admin_app'));

-- Backfill existing speakers imported from Sessionize.
UPDATE speakers
SET source = 'sessionize'
WHERE source_session_id IS NOT NULL
  AND source IS NULL;

ALTER TABLE speakers
    ADD CONSTRAINT speakers_event_id_source_session_id_key
        UNIQUE (event_id, source_session_id);

