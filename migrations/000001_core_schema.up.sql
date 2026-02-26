-- Users and roles (first: events.owner_id references users)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255),
    last_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(64) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);

INSERT INTO roles (id, code) VALUES
    (gen_random_uuid(), 'attendee'),
    (gen_random_uuid(), 'admin')
ON CONFLICT (code) DO NOTHING;

-- Events (event_code from start; owner_id references users)
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    event_code CHAR(4) NOT NULL UNIQUE,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    date TIMESTAMP WITH TIME ZONE,
    description TEXT,
    location_lat DOUBLE PRECISION,
    location_lng DOUBLE PRECISION,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_events_owner_id ON events(owner_id);
CREATE INDEX idx_events_event_code ON events(event_code);

-- Rooms (with generic external source identifiers)
CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    source_session_id INTEGER,
    source VARCHAR(20)
        CHECK (source IN ('sessionize', 'admin_app')),
    not_bookable BOOLEAN NOT NULL DEFAULT false,
    capacity INTEGER,
    description TEXT,
    how_to_get_there TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(event_id, source_session_id)
);

CREATE INDEX idx_rooms_event_id ON rooms(event_id);

-- Sessions (with generic external source identifiers)
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    source_session_id VARCHAR(50),
    source VARCHAR(20)
        CHECK (source IN ('sessionize', 'admin_app')),
    title VARCHAR(255) NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(room_id, source_session_id)
);

CREATE INDEX idx_sessions_room_id ON sessions(room_id);

-- Tags as first-class entity (no string backfill needed in early dev)
CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE
);

-- Event–tag many-to-many
CREATE TABLE IF NOT EXISTS event_tags (
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, tag_id)
);

CREATE INDEX idx_event_tags_tag_id ON event_tags(tag_id);
CREATE INDEX idx_event_tags_event_id ON event_tags(event_id);

-- Session–tag many-to-many (using tag ids directly)
CREATE TABLE IF NOT EXISTS session_tags (
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (session_id, tag_id)
);

CREATE INDEX idx_session_tags_session_id ON session_tags(session_id);
CREATE INDEX idx_session_tags_tag_id ON session_tags(tag_id);

-- Event team members
CREATE TABLE IF NOT EXISTS event_team_members (
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, user_id)
);

CREATE INDEX idx_event_team_members_event_id ON event_team_members(event_id);
CREATE INDEX idx_event_team_members_user_id ON event_team_members(user_id);

-- Event registrations
CREATE TABLE IF NOT EXISTS event_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE UNIQUE INDEX idx_event_registrations_event_user
    ON event_registrations (event_id, user_id);

-- Speakers (per event, imported from multiple sources)
CREATE TABLE IF NOT EXISTS speakers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    source_session_id VARCHAR(100) NOT NULL,
    source VARCHAR(20)
        CHECK (source IN ('sessionize', 'admin_app')),
    first_name VARCHAR(255) NOT NULL DEFAULT '',
    last_name VARCHAR(255) NOT NULL DEFAULT '',
    full_name VARCHAR(512) NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    tag_line VARCHAR(512) NOT NULL DEFAULT '',
    profile_picture TEXT NOT NULL DEFAULT '',
    is_top_speaker BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(event_id, source_session_id)
);

CREATE INDEX idx_speakers_event_id ON speakers(event_id);

-- Session–speaker many-to-many
CREATE TABLE IF NOT EXISTS session_speakers (
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    speaker_id UUID NOT NULL REFERENCES speakers(id) ON DELETE CASCADE,
    PRIMARY KEY (session_id, speaker_id)
);

CREATE INDEX idx_session_speakers_session_id ON session_speakers(session_id);
CREATE INDEX idx_session_speakers_speaker_id ON session_speakers(speaker_id);

-- Login codes for passwordless auth
CREATE TABLE IF NOT EXISTS login_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    code_hash TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_login_codes_email ON login_codes(email);
CREATE INDEX idx_login_codes_expires_at ON login_codes(expires_at);

-- Event invitations: track emails invited to register for an event
CREATE TABLE IF NOT EXISTS event_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_event_invitations_event_email ON event_invitations(event_id, email);
CREATE INDEX idx_event_invitations_event_id ON event_invitations(event_id);
