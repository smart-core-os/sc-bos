-- Used to normalise identifiers for health checks.
-- Each check id is only unique per device,
-- so we need to combine it with the device name to get a unique key.
CREATE TABLE health_check_ids
(
    id       INTEGER PRIMARY KEY,
    name     TEXT NOT NULL,
    check_id TEXT NOT NULL,
    UNIQUE (name, check_id)
);

-- Used to store infrequently changing properties of health checks.
CREATE TABLE health_check_aux
(
    id      INTEGER PRIMARY KEY,
    -- A binary proto message representing infrequently changing data.
    -- It's important that this does not include fields like timestamps.
    -- The data should also not contain the check id or other identifying
    -- information that would limit reuse.
    payload BLOB NOT NULL UNIQUE
);

CREATE TABLE health_check_history
(
    id       INTEGER PRIMARY KEY, -- encodes both the timestamp and a sequence number
    check_id INTEGER NOT NULL REFERENCES health_check_ids (id) ON DELETE CASCADE,
    aux_id   INTEGER NOT NULL REFERENCES health_check_aux (id) ON DELETE RESTRICT,
    -- A binary proto message representing a health check excluding the payload in aux_id.
    payload  BLOB    NOT NULL
);

CREATE INDEX health_check_history_check_id_idx ON health_check_history (check_id, id);
