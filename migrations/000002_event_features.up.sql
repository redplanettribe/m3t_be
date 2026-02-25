-- Session tags
CREATE TABLE session_tags (
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    tag VARCHAR(255) NOT NULL,
    PRIMARY KEY (session_id, tag)
);
CREATE INDEX idx_session_tags_session_id ON session_tags(session_id);

-- Event team members
CREATE TABLE event_team_members (
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, user_id)
);
CREATE INDEX idx_event_team_members_event_id ON event_team_members(event_id);
CREATE INDEX idx_event_team_members_user_id ON event_team_members(user_id);

-- Event registrations
CREATE TABLE event_registrations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id uuid NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE UNIQUE INDEX idx_event_registrations_event_user
    ON event_registrations (event_id, user_id);
