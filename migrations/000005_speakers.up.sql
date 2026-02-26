-- Speakers (per event, imported from Sessionize or created manually)
CREATE TABLE IF NOT EXISTS speakers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    sessionize_speaker_id VARCHAR(100) NOT NULL,
    first_name VARCHAR(255) NOT NULL DEFAULT '',
    last_name VARCHAR(255) NOT NULL DEFAULT '',
    full_name VARCHAR(512) NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    tag_line VARCHAR(512) NOT NULL DEFAULT '',
    profile_picture TEXT NOT NULL DEFAULT '',
    is_top_speaker BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(event_id, sessionize_speaker_id)
);

CREATE INDEX idx_speakers_event_id ON speakers(event_id);

-- Session-speaker many-to-many
CREATE TABLE IF NOT EXISTS session_speakers (
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    speaker_id UUID NOT NULL REFERENCES speakers(id) ON DELETE CASCADE,
    PRIMARY KEY (session_id, speaker_id)
);

CREATE INDEX idx_session_speakers_session_id ON session_speakers(session_id);
CREATE INDEX idx_session_speakers_speaker_id ON session_speakers(speaker_id);
