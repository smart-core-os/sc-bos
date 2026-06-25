-- Sites

-- name: CreateSite :one
INSERT INTO sites (name, create_time)
VALUES (:name, datetime('now', 'subsec'))
RETURNING *;

-- name: GetSite :one
SELECT *
FROM sites
WHERE id = :id;

-- name: ListSites :many
SELECT *
FROM sites
WHERE id > :after_id
ORDER BY id
LIMIT :limit;

-- name: CountSites :one
SELECT COUNT(*) AS count
FROM sites;

-- name: UpdateSite :one
UPDATE sites
SET name = :name
WHERE id = :id
RETURNING *;

-- name: DeleteSite :execrows
DELETE FROM sites
WHERE id = :id;

-- Nodes

-- name: CreateNode :one
INSERT INTO nodes (hostname, site_id, platform, secret_hash, create_time)
VALUES (:hostname, :site_id, :platform, :secret_hash, datetime('now', 'subsec'))
RETURNING *;

-- name: GetNode :one
SELECT *
FROM nodes
WHERE id = :id;

-- name: ListNodes :many
SELECT *
FROM nodes
WHERE id > :after_id
ORDER BY id
LIMIT :limit;

-- name: ListNodesBySite :many
SELECT *
FROM nodes
WHERE site_id = :site_id AND id > :after_id
ORDER BY id
LIMIT :limit;

-- name: CountNodes :one
SELECT COUNT(*) AS count
FROM nodes;

-- name: CountNodesBySite :one
SELECT COUNT(*) AS count
FROM nodes
WHERE site_id = :site_id;

-- name: UpdateNode :one
UPDATE nodes
SET hostname = :hostname, site_id = :site_id, platform = :platform
WHERE id = :id
RETURNING *;

-- name: UpdateNodeSecretHash :exec
UPDATE nodes SET secret_hash = :secret_hash WHERE id = :id;

-- name: DeleteNode :execrows
DELETE FROM nodes
WHERE id = :id;

-- Node Check-Ins

-- name: CreateNodeCheckIn :one
INSERT INTO node_check_ins (
    node_id, check_in_time,
    current_deployment_id, installing_deployment_id, installing_deployment_error, installing_deployment_attempts,
    current_update_deployment_id, installing_update_deployment_id, installing_update_error, installing_update_attempts
)
VALUES (
    :node_id, datetime('now', 'subsec'),
    :current_deployment_id, :installing_deployment_id, :installing_deployment_error, :installing_deployment_attempts,
    :current_update_deployment_id, :installing_update_deployment_id, :installing_update_error, :installing_update_attempts
)
RETURNING *;

-- name: GetNodeCheckIn :one
SELECT *
FROM node_check_ins
WHERE id = :id;

-- name: ListNodeCheckInsByNode :many
SELECT *
FROM node_check_ins
WHERE node_id = :node_id AND id < :before_id
ORDER BY id DESC
LIMIT :limit;

-- name: DeleteNodeCheckIn :execrows
DELETE FROM node_check_ins
WHERE id = :id;

-- Config Versions

-- name: CreateConfigVersion :one
INSERT INTO config_versions (node_id, description, payload, create_time)
VALUES (:node_id, :description, :payload, datetime('now', 'subsec'))
RETURNING *;

-- name: GetConfigVersion :one
SELECT *
FROM config_versions
WHERE id = :id;


-- name: ListConfigVersions :many
SELECT *
FROM config_versions
WHERE id > :after_id
ORDER BY id
LIMIT :limit;

-- name: ListConfigVersionsByNode :many
SELECT *
FROM config_versions
WHERE node_id = :node_id AND id > :after_id
ORDER BY id
LIMIT :limit;

-- name: DeleteConfigVersion :execrows
DELETE FROM config_versions
WHERE id = :id;

-- Config Deployments

-- name: CreateConfigDeployment :one
INSERT INTO config_deployments (config_version_id, status, start_time, finished_time)
VALUES (:config_version_id, :status, datetime('now', 'subsec'), NULL)
RETURNING *;

-- name: GetConfigDeployment :one
SELECT *
FROM config_deployments
WHERE id = :id;

-- name: GetConfigDeploymentWithConfigVersion :one
SELECT d.*, cv.*
FROM config_deployments d
JOIN config_versions cv ON d.config_version_id = cv.id
WHERE d.id = :id;

-- name: ListConfigDeployments :many
SELECT *
FROM config_deployments
WHERE id < :before_ID
ORDER BY id DESC
LIMIT :limit;

-- name: ListConfigDeploymentsByConfigVersion :many
SELECT *
FROM config_deployments
WHERE config_version_id = :config_version_id AND id < :before_id
ORDER BY id DESC
LIMIT :limit;

-- name: ListConfigDeploymentsByNode :many
SELECT d.*
FROM config_deployments d
JOIN config_versions cv ON d.config_version_id = cv.id
WHERE cv.node_id = :node_id AND d.id < :before_id
ORDER BY d.id DESC
LIMIT :limit;

-- name: CountConfigDeployments :one
SELECT COUNT(*) AS count
FROM config_deployments;

-- name: UpdateConfigDeploymentStatus :one
UPDATE config_deployments
SET status = :status,
    reason = :reason,
    finished_time = CASE
        WHEN :status = 'completed' OR :status = 'failed' OR :status = 'cancelled' THEN datetime('now', 'subsec')
        ELSE finished_time
    END
WHERE id = :id
RETURNING *;

-- name: DeleteConfigDeployment :execrows
DELETE FROM config_deployments
WHERE id = :id;

-- name: CancelPendingConfigDeploymentsByNode :execrows
UPDATE config_deployments
SET status = 'cancelled',
    finished_time = datetime('now', 'subsec')
WHERE config_version_id IN (
    SELECT id FROM config_versions WHERE node_id = :node_id
) AND status = 'pending';

-- name: GetNodeBySecretHash :one
SELECT * FROM nodes WHERE secret_hash = :secret_hash;

-- name: GetActiveConfigDeploymentByNode :one
SELECT d.*
FROM config_deployments d
JOIN config_versions cv ON d.config_version_id = cv.id
WHERE cv.node_id = :node_id AND d.status IN ('pending', 'in_progress')
ORDER BY d.id DESC
LIMIT 1;

-- Enrollment Codes

-- name: CreateEnrollmentCode :one
INSERT INTO enrollment_codes (node_id, code, expires_at)
VALUES (:node_id, :code, :expires_at)
RETURNING *;

-- name: GetActiveEnrollmentCode :one
SELECT * FROM enrollment_codes
WHERE code = :code AND used_at IS NULL AND expires_at > datetime('now', 'subsec');

-- name: MarkEnrollmentCodeUsed :exec
UPDATE enrollment_codes SET used_at = datetime('now', 'subsec') WHERE id = :id;

-- Update Artefacts
-- The payload itself is stored as an external file on disk (named by artefact id) and is never held
-- in the database, so it is never selected here; the size column records its byte length.

-- name: CreateUpdateArtefact :one
INSERT INTO update_artefacts (site_id, platform, kind, version, sha256, description, size, create_time)
VALUES (:site_id, :platform, :kind, :version, :sha256, :description, :size, datetime('now', 'subsec'))
RETURNING *;

-- name: GetUpdateArtefact :one
SELECT id, site_id, platform, kind, version, sha256, description, size, create_time
FROM update_artefacts
WHERE id = :id;

-- name: ListUpdateArtefacts :many
SELECT id, site_id, platform, kind, version, sha256, description, size, create_time
FROM update_artefacts
WHERE id > :after_id
  AND (sqlc.arg(platform) = '' OR platform = sqlc.arg(platform))
  AND (sqlc.arg(site_id) = 0 OR site_id = sqlc.arg(site_id) OR site_id IS NULL)
ORDER BY id
LIMIT :limit;

-- name: ListUpdateArtefactIDs :many
SELECT id FROM update_artefacts ORDER BY id;

-- name: DeleteUpdateArtefact :execrows
DELETE FROM update_artefacts
WHERE id = :id;

-- Update Deployments

-- name: CreateUpdateDeployment :one
INSERT INTO update_deployments (update_artefact_id, node_id, status, start_time, finished_time)
VALUES (:update_artefact_id, :node_id, :status, datetime('now', 'subsec'), NULL)
RETURNING *;

-- name: GetUpdateDeployment :one
SELECT *
FROM update_deployments
WHERE id = :id;

-- name: ListUpdateDeployments :many
SELECT *
FROM update_deployments
WHERE id < :before_id
ORDER BY id DESC
LIMIT :limit;

-- name: ListUpdateDeploymentsByNode :many
SELECT *
FROM update_deployments
WHERE node_id = :node_id AND id < :before_id
ORDER BY id DESC
LIMIT :limit;

-- name: SetUpdateDeploymentStatus :one
UPDATE update_deployments
SET status = :status,
    reason = :reason,
    finished_time = CASE
        WHEN :status = 'completed' OR :status = 'failed' OR :status = 'cancelled' THEN datetime('now', 'subsec')
        ELSE finished_time
    END
WHERE id = :id
RETURNING *;

-- name: DeleteUpdateDeployment :execrows
DELETE FROM update_deployments
WHERE id = :id;

-- name: CancelPendingUpdateDeploymentsByNodeAndKind :execrows
-- Scoped to one artefact kind so cancelling a pending BOS-image deployment leaves a pending
-- supervisor-rpm deployment (and vice versa) untouched: the channels are independent.
UPDATE update_deployments
SET status = 'cancelled',
    finished_time = datetime('now', 'subsec')
WHERE node_id = :node_id AND status = 'pending'
  AND update_artefact_id IN (SELECT id FROM update_artefacts WHERE kind = :kind);

-- name: GetActiveUpdateDeploymentByNodeAndKind :one
-- The active deployment for one channel (artefact kind), so a node can have a BOS-image update and a
-- supervisor-rpm update in flight at the same time.
SELECT d.*
FROM update_deployments d
JOIN update_artefacts a ON a.id = d.update_artefact_id
WHERE d.node_id = :node_id AND a.kind = :kind AND d.status IN ('pending', 'in_progress')
ORDER BY d.id DESC
LIMIT 1;
