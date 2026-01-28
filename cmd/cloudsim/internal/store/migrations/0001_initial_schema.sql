CREATE TABLE sites (
    id              TEXT PRIMARY KEY, -- UUID
    name            TEXT NOT NULL,
    create_time     DATETIME NOT NULL,

    CONSTRAINT create_time_format CHECK ( create_time IS datetime(create_time, 'subsec') )
);

CREATE TABLE nodes (
    id              TEXT PRIMARY KEY, -- UUID
    hostname        TEXT NOT NULL,
    site_id         TEXT NOT NULL,
    create_time     DATETIME NOT NULL,

    FOREIGN KEY (site_id) REFERENCES sites (id) ON DELETE CASCADE,
    CONSTRAINT create_time_format CHECK ( create_time IS datetime(create_time, 'subsec') )
);

CREATE INDEX nodes_site_id ON nodes (site_id);

CREATE TABLE config_versions (
    id              TEXT PRIMARY KEY, -- UUID
    node_id         TEXT NOT NULL,
    version_number  INTEGER NOT NULL,
    payload         BLOB NOT NULL,
    create_time     DATETIME NOT NULL,

    FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE,
    CONSTRAINT create_time_format CHECK ( create_time IS datetime(create_time, 'subsec') )
);

CREATE UNIQUE INDEX config_versions_node_version ON config_versions (node_id, version_number);
CREATE INDEX config_versions_node_id ON config_versions (node_id);

CREATE TABLE deployments (
    id                  TEXT PRIMARY KEY, -- UUID
    config_version_id   TEXT NOT NULL,
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
