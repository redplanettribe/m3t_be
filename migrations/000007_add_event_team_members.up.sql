CREATE TABLE event_team_members (
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, user_id)
);

CREATE INDEX idx_event_team_members_event_id ON event_team_members(event_id);
CREATE INDEX idx_event_team_members_user_id ON event_team_members(user_id);
