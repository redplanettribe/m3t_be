CREATE TABLE session_tags (
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    tag VARCHAR(255) NOT NULL,
    PRIMARY KEY (session_id, tag)
);

CREATE INDEX idx_session_tags_session_id ON session_tags(session_id);
