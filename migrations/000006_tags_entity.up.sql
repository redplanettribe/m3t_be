-- Tags as first-class entity (one row per name globally)
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE
);

-- Event–tag many-to-many
CREATE TABLE event_tags (
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, tag_id)
);
CREATE INDEX idx_event_tags_tag_id ON event_tags(tag_id);
CREATE INDEX idx_event_tags_event_id ON event_tags(event_id);

-- Backfill tags from existing session_tags string values
INSERT INTO tags (name)
SELECT DISTINCT tag FROM session_tags
ON CONFLICT (name) DO NOTHING;

-- New session_tags junction (session_id, tag_id)
CREATE TABLE session_tags_new (
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (session_id, tag_id)
);
INSERT INTO session_tags_new (session_id, tag_id)
SELECT st.session_id, t.id
FROM session_tags st
JOIN tags t ON t.name = st.tag;

-- Backfill event_tags from sessions → rooms → event_id
INSERT INTO event_tags (event_id, tag_id)
SELECT DISTINCT r.event_id, stn.tag_id
FROM session_tags_new stn
JOIN sessions s ON s.id = stn.session_id
JOIN rooms r ON r.id = s.room_id
ON CONFLICT (event_id, tag_id) DO NOTHING;

-- Replace old session_tags with new structure
DROP TABLE session_tags;
ALTER TABLE session_tags_new RENAME TO session_tags;
CREATE INDEX idx_session_tags_session_id ON session_tags(session_id);
CREATE INDEX idx_session_tags_tag_id ON session_tags(tag_id);
