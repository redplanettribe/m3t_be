-- Restore session_tags as (session_id, tag varchar)
CREATE TABLE session_tags_old (
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    tag VARCHAR(255) NOT NULL,
    PRIMARY KEY (session_id, tag)
);
INSERT INTO session_tags_old (session_id, tag)
SELECT st.session_id, t.name
FROM session_tags st
JOIN tags t ON t.id = st.tag_id;

DROP TABLE session_tags;
ALTER TABLE session_tags_old RENAME TO session_tags;
CREATE INDEX idx_session_tags_session_id ON session_tags(session_id);

DROP TABLE event_tags;
DROP TABLE tags;
