-- Rename deployments -> config_deployments now that update deployments exist alongside it.
-- The unqualified "deployment" is ambiguous; config_deployments pairs with update_deployments.
-- (The deployments table has no secondary indexes: migration 0002 rebuilt it without them.)
ALTER TABLE deployments RENAME TO config_deployments;

-- BOS software update distribution, parallel to config versions/deployments.
-- Update artefacts are site- and platform-scoped (site_id NULL = generic, all sites) and carry a
-- version + sha256 so BOS/the Supervisor can verify the download. Payloads are podman-save tarballs
-- (hundreds of MB), so they are stored as external files on disk (named by artefact id) rather than
-- as BLOBs in the database; the row only records metadata, including the payload size.

-- Nodes gain a platform so a deployment can only target an artefact of the matching platform. No CHECK
-- on the value: this is a simulator, and validating an enum in the DB just makes adding new
-- platforms/kinds harder for no benefit (any validation lives in Go).
ALTER TABLE nodes ADD COLUMN platform TEXT NOT NULL DEFAULT 'podman';

CREATE TABLE update_artefacts (
    id              INTEGER PRIMARY KEY,
    site_id         INTEGER,            -- NULL = generic artefact, available to all sites
    platform        TEXT NOT NULL,
    kind            TEXT NOT NULL DEFAULT 'bos-image', -- 'bos-image' (podman tarball) or 'supervisor-rpm'
    version         TEXT NOT NULL,
    sha256          TEXT,               -- hex, computed server-side while streaming to the file
    description     TEXT,
    size            INTEGER NOT NULL,   -- payload byte length; the payload itself lives as an external file
    create_time     DATETIME NOT NULL,

    FOREIGN KEY (site_id) REFERENCES sites (id) ON DELETE CASCADE,
    CONSTRAINT create_time_format CHECK ( create_time IS datetime(create_time, 'subsec') )
);

CREATE INDEX update_artefacts_site_id ON update_artefacts (site_id);

CREATE TABLE update_deployments (
    id                  INTEGER PRIMARY KEY,
    update_artefact_id  INTEGER NOT NULL,
    node_id             INTEGER NOT NULL,
    status              TEXT NOT NULL,
    start_time          DATETIME NOT NULL,
    finished_time       DATETIME,
    reason              TEXT, -- optional reason for failure or cancellation

    FOREIGN KEY (update_artefact_id) REFERENCES update_artefacts (id) ON DELETE CASCADE,
    FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE,
    CONSTRAINT status_valid CHECK ( status IN ('pending', 'in_progress', 'completed', 'failed', 'cancelled') ),
    CONSTRAINT start_time_format CHECK ( start_time IS datetime(start_time, 'subsec') ),
    CONSTRAINT finished_time_format CHECK ( finished_time IS NULL OR finished_time IS datetime(finished_time, 'subsec') )
);

CREATE INDEX update_deployments_node_id ON update_deployments (node_id);
CREATE INDEX update_deployments_status ON update_deployments (status);

-- node_check_ins records the update channel alongside the existing config channel.
ALTER TABLE node_check_ins ADD COLUMN current_update_deployment_id    INTEGER;
ALTER TABLE node_check_ins ADD COLUMN installing_update_deployment_id INTEGER;
ALTER TABLE node_check_ins ADD COLUMN installing_update_error         TEXT;
ALTER TABLE node_check_ins ADD COLUMN installing_update_attempts      INTEGER;
