-- Revert generalized external source identifiers for rooms, sessions, and speakers.

-- Speakers: drop new unique constraint, remove source column, and rename source_session_id back.
ALTER TABLE speakers
    DROP CONSTRAINT IF EXISTS speakers_event_id_source_session_id_key;

ALTER TABLE speakers
    DROP COLUMN IF EXISTS source;

ALTER TABLE speakers
    RENAME COLUMN source_session_id TO sessionize_speaker_id;

ALTER TABLE speakers
    ADD CONSTRAINT speakers_event_id_sessionize_speaker_id_key
        UNIQUE (event_id, sessionize_speaker_id);

-- Sessions: drop new unique constraint, remove source column, and rename source_session_id back.
ALTER TABLE sessions
    DROP CONSTRAINT IF EXISTS sessions_room_id_source_session_id_key;

ALTER TABLE sessions
    DROP COLUMN IF EXISTS source;

ALTER TABLE sessions
    RENAME COLUMN source_session_id TO sessionize_session_id;

ALTER TABLE sessions
    ADD CONSTRAINT sessions_room_id_sessionize_session_id_key
        UNIQUE (room_id, sessionize_session_id);

-- Rooms: remove source column (rooms already used source_session_id before this migration).
ALTER TABLE rooms
    DROP COLUMN IF EXISTS source;

