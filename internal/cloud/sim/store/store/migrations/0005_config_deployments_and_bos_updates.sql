-- Rename deployments -> config_deployments now that binary deployments exist alongside it.
-- The unqualified "deployment" is ambiguous; config_deployments pairs with binary_deployments.
ALTER TABLE deployments RENAME TO config_deployments;

-- Config versions gain a checksum so the BOS client can verify the config payload it downloads,
-- mirroring binary_artefacts.sha256. The digest is the raw 32-byte SHA-256 of the payload, computed
-- server-side on upload. Existing rows keep a NULL checksum (populated only for versions created from
-- now on).
ALTER TABLE config_versions ADD COLUMN sha256 BLOB;

-- Config versions gain an optional human-readable version string, reported to the node and echoed back
-- as running.config.version. NULL when unset.
ALTER TABLE config_versions ADD COLUMN version TEXT;

-- BOS software binary distribution, parallel to config versions/deployments.
-- Binary artefacts are site- and platform-scoped (site_id NULL = generic, all sites) and carry a
-- version + sha256 so BOS/the Supervisor can verify the download. Payloads are large tarballs
-- (hundreds of MB), so they are stored as external files on disk (named by artefact id) rather than
-- as BLOBs in the database; the row records only metadata. The payload byte length is not stored - it
-- is the size of the external file, read with stat when needed.

-- Nodes gain a platform (os + arch, GOOS x GOARCH) so a deployment can only target an artefact of the
-- matching platform. Empty until a node first reports its platform on check-in.
ALTER TABLE nodes ADD COLUMN os   TEXT NOT NULL DEFAULT '';
ALTER TABLE nodes ADD COLUMN arch TEXT NOT NULL DEFAULT '';

CREATE TABLE binary_artefacts (
    id              INTEGER PRIMARY KEY,
    site_id         INTEGER,            -- NULL = generic artefact, available to all sites
    os              TEXT NOT NULL,      -- GOOS, e.g. linux; free-text, validated in Go
    arch            TEXT NOT NULL,      -- GOARCH, e.g. arm64; free-text, validated in Go
    version         TEXT NOT NULL,
    sha256          BLOB,               -- raw 32-byte digest, computed server-side while streaming to the file
    description     TEXT,
    create_time     DATETIME NOT NULL,

    FOREIGN KEY (site_id) REFERENCES sites (id) ON DELETE CASCADE,
    CONSTRAINT sha256_length CHECK ( sha256 IS NULL OR length(sha256) = 32 ),
    CONSTRAINT create_time_format CHECK ( create_time IS datetime(create_time, 'subsec') )
);

CREATE INDEX binary_artefacts_site_id ON binary_artefacts (site_id);

CREATE TABLE binary_deployments (
    id                  INTEGER PRIMARY KEY,
    binary_artefact_id  INTEGER NOT NULL,
    node_id             INTEGER NOT NULL,
    status              TEXT NOT NULL,
    start_time          DATETIME NOT NULL,
    finished_time       DATETIME,
    reason              TEXT, -- optional reason for failure or cancellation

    FOREIGN KEY (binary_artefact_id) REFERENCES binary_artefacts (id) ON DELETE CASCADE,
    FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE,
    CONSTRAINT status_valid CHECK ( status IN ('pending', 'in_progress', 'completed', 'failed', 'cancelled') ),
    CONSTRAINT start_time_format CHECK ( start_time IS datetime(start_time, 'subsec') ),
    CONSTRAINT finished_time_format CHECK ( finished_time IS NULL OR finished_time IS datetime(finished_time, 'subsec') )
);

CREATE INDEX binary_deployments_node_id ON binary_deployments (node_id);
CREATE INDEX binary_deployments_status ON binary_deployments (status);

-- node_check_ins records the update channel alongside the existing config channel.
ALTER TABLE node_check_ins ADD COLUMN current_binary_deployment_id    INTEGER;
ALTER TABLE node_check_ins ADD COLUMN installing_binary_deployment_id INTEGER;
ALTER TABLE node_check_ins ADD COLUMN installing_binary_error         TEXT;
ALTER TABLE node_check_ins ADD COLUMN installing_binary_attempts      INTEGER;
