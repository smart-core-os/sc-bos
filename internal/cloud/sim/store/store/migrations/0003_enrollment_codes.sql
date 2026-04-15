CREATE TABLE enrollment_codes (
    id          INTEGER PRIMARY KEY,
    node_id     INTEGER NOT NULL,
    code        TEXT NOT NULL,
    expires_at  DATETIME NOT NULL,
    used_at     DATETIME,

    FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE,
    CONSTRAINT expires_at_format CHECK ( expires_at IS datetime(expires_at, 'subsec') ),
    CONSTRAINT used_at_format CHECK ( used_at IS NULL OR used_at IS datetime(used_at, 'subsec') )
);

CREATE UNIQUE INDEX enrollment_codes_code ON enrollment_codes (code);
CREATE INDEX enrollment_codes_node_id ON enrollment_codes (node_id);
