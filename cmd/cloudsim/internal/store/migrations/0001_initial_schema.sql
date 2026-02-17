CREATE TABLE sites (
    id              INTEGER PRIMARY KEY,
    name            TEXT NOT NULL,
    create_time     DATETIME NOT NULL,

    CONSTRAINT create_time_format CHECK ( create_time IS datetime(create_time, 'subsec') )
);

CREATE TABLE nodes (
    id              INTEGER PRIMARY KEY,
    hostname        TEXT NOT NULL,
    site_id         INTEGER NOT NULL,
    create_time     DATETIME NOT NULL,

    FOREIGN KEY (site_id) REFERENCES sites (id) ON DELETE CASCADE,
    CONSTRAINT create_time_format CHECK ( create_time IS datetime(create_time, 'subsec') )
);

CREATE INDEX nodes_site_id ON nodes (site_id);

CREATE TABLE node_check_ins (
    id              INTEGER PRIMARY KEY,
    node_id         INTEGER NOT NULL,
    check_in_time   DATETIME NOT NULL,

    -- can add additional fields to reflect node status at the time of check in

    FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE,
    CONSTRAINT check_in_time_format CHECK ( check_in_time IS datetime(check_in_time, 'subsec') )
);

CREATE INDEX node_check_ins_node_id ON node_check_ins (node_id);

CREATE TABLE config_versions (
    id              INTEGER PRIMARY KEY,
    node_id         INTEGER NOT NULL,
    description     TEXT,
    payload         BLOB NOT NULL,
    create_time     DATETIME NOT NULL,

    FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE,
    CONSTRAINT create_time_format CHECK ( create_time IS datetime(create_time, 'subsec') )
);

CREATE INDEX config_versions_node_id ON config_versions (node_id);

CREATE TABLE deployments (
    id                  INTEGER PRIMARY KEY,
    config_version_id   INTEGER NOT NULL,
    status              TEXT NOT NULL,
    start_time          DATETIME NOT NULL,
    finished_time       DATETIME,

    FOREIGN KEY (config_version_id) REFERENCES config_versions (id) ON DELETE CASCADE,
    CONSTRAINT status_valid CHECK ( status IN ('PENDING', 'IN_PROGRESS', 'COMPLETED', 'FAILED') ),
    CONSTRAINT start_time_format CHECK ( start_time IS datetime(start_time, 'subsec') ),
    CONSTRAINT finished_time_format CHECK ( finished_time IS NULL OR finished_time IS datetime(finished_time, 'subsec') )
);

CREATE INDEX deployments_config_version_id ON deployments (config_version_id);
CREATE INDEX deployments_status ON deployments (status);
