-- Migrate deployment status values to lowercase to match the SCC API.

CREATE TABLE deployments_new (
    id                  INTEGER PRIMARY KEY,
    config_version_id   INTEGER NOT NULL,
    status              TEXT NOT NULL,
    start_time          DATETIME NOT NULL,
    finished_time       DATETIME,
    reason              TEXT, -- optional field to capture reason for failure or cancellation

    FOREIGN KEY (config_version_id) REFERENCES config_versions (id) ON DELETE CASCADE,
    CONSTRAINT status_valid CHECK ( status IN ('pending', 'in_progress', 'completed', 'failed', 'cancelled') ),
    CONSTRAINT start_time_format CHECK ( start_time IS datetime(start_time, 'subsec') ),
    CONSTRAINT finished_time_format CHECK ( finished_time IS NULL OR finished_time IS datetime(finished_time, 'subsec') )
);

INSERT INTO deployments_new (id, config_version_id, status, start_time, finished_time, reason)
SELECT id, config_version_id, LOWER(status), start_time, finished_time, reason FROM deployments;

DROP TABLE deployments;
ALTER TABLE deployments_new RENAME TO deployments;
