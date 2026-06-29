-- Replace the OAuth2 client-secret model with X.509 credential slots.
-- A node authenticates with a client certificate; each credential slot holds the
-- metadata of the certificate currently occupying it. The credential_id is the
-- opaque marker minted per enrollment-code exchange (stable across renewals) and
-- is what revocation acts on.

DROP INDEX nodes_secret_hash;
ALTER TABLE nodes DROP COLUMN secret_hash;

CREATE TABLE credentials (
    id              INTEGER PRIMARY KEY,
    node_id         INTEGER NOT NULL,
    credential_id   TEXT NOT NULL,    -- opaque marker, minted per code exchange, carried in the cert SAN
    slot            TEXT NOT NULL,    -- 'primary' | 'secondary'
    serial          TEXT NOT NULL,    -- current leaf serial number (hex)
    fingerprint     BLOB NOT NULL,    -- sha256 of the current leaf DER
    not_before      DATETIME NOT NULL,
    not_after       DATETIME NOT NULL,
    create_time     DATETIME NOT NULL,

    FOREIGN KEY (node_id) REFERENCES nodes (id) ON DELETE CASCADE,
    CONSTRAINT slot_valid CHECK ( slot IN ('primary', 'secondary') ),
    CONSTRAINT not_before_format CHECK ( not_before IS datetime(not_before, 'subsec') ),
    CONSTRAINT not_after_format CHECK ( not_after IS datetime(not_after, 'subsec') ),
    CONSTRAINT create_time_format CHECK ( create_time IS datetime(create_time, 'subsec') )
);

CREATE UNIQUE INDEX credentials_credential_id ON credentials (credential_id);
CREATE UNIQUE INDEX credentials_node_slot ON credentials (node_id, slot);
CREATE INDEX credentials_node_id ON credentials (node_id);

-- Which credential slot an enrollment code fills when exchanged (default primary;
-- secondary is used for a hardware-replacement overlap).
ALTER TABLE enrollment_codes ADD COLUMN target_slot TEXT NOT NULL DEFAULT 'primary'
    CHECK ( target_slot IN ('primary', 'secondary') );
