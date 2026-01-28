-- Sites

-- name: CreateSite :one
INSERT INTO sites (name, create_time)
VALUES (:name, datetime('now', 'subsec'))
RETURNING *;

-- name: GetSite :one
SELECT *
FROM sites
WHERE id = :id;

-- name: GetSiteByName :one
SELECT *
FROM sites
WHERE name = :name;

-- name: ListSites :many
SELECT *
FROM sites
ORDER BY create_time DESC
LIMIT :limit OFFSET :offset;

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
INSERT INTO nodes (hostname, site_id, create_time)
VALUES (:hostname, :site_id, datetime('now', 'subsec'))
RETURNING *;

-- name: GetNode :one
SELECT *
FROM nodes
WHERE id = :id;

-- name: GetNodeByHostname :one
SELECT *
FROM nodes
WHERE hostname = :hostname;

-- name: ListNodes :many
SELECT *
FROM nodes
ORDER BY create_time DESC
LIMIT :limit OFFSET :offset;

-- name: ListNodesBySite :many
SELECT *
FROM nodes
WHERE site_id = :site_id
ORDER BY create_time DESC;

-- name: CountNodes :one
SELECT COUNT(*) AS count
FROM nodes;

-- name: CountNodesBySite :one
SELECT COUNT(*) AS count
FROM nodes
WHERE site_id = :site_id;

-- name: UpdateNode :one
UPDATE nodes
SET hostname = :hostname, site_id = :site_id
WHERE id = :id
RETURNING *;

-- name: DeleteNode :execrows
DELETE FROM nodes
WHERE id = :id;

-- Config Versions

-- name: CreateConfigVersion :one
INSERT INTO config_versions (node_id, version_number, payload, create_time)
VALUES (:node_id, :version_number, :payload, datetime('now', 'subsec'))
RETURNING *;

-- name: GetConfigVersion :one
SELECT *
FROM config_versions
WHERE id = :id;

-- name: GetConfigVersionByNodeAndVersion :one
SELECT *
FROM config_versions
WHERE node_id = :node_id AND version_number = :version_number;

-- name: ListConfigVersions :many
SELECT *
FROM config_versions
ORDER BY create_time DESC
LIMIT :limit OFFSET :offset;

-- name: ListConfigVersionsByNode :many
SELECT *
FROM config_versions
WHERE node_id = :node_id
ORDER BY version_number DESC;

-- name: GetLatestConfigVersionByNode :one
SELECT *
FROM config_versions
WHERE node_id = :node_id
ORDER BY version_number DESC
LIMIT 1;

-- name: CountConfigVersions :one
SELECT COUNT(*) AS count
FROM config_versions;

-- name: CountConfigVersionsByNode :one
SELECT COUNT(*) AS count
FROM config_versions
WHERE node_id = :node_id;

-- name: DeleteConfigVersion :execrows
DELETE FROM config_versions
WHERE id = :id;

-- Deployments

-- name: CreateDeployment :one
INSERT INTO deployments (config_version_id, status, start_time, finished_time)
VALUES (:config_version_id, :status, datetime('now', 'subsec'), NULL)
RETURNING *;

-- name: GetDeployment :one
SELECT *
FROM deployments
WHERE id = :id;

-- name: ListDeployments :many
SELECT *
FROM deployments
ORDER BY start_time DESC
LIMIT :limit OFFSET :offset;

-- name: ListDeploymentsByConfigVersion :many
SELECT *
FROM deployments
WHERE config_version_id = :config_version_id
ORDER BY start_time DESC;

-- name: ListDeploymentsByStatus :many
SELECT *
FROM deployments
WHERE status = :status
ORDER BY start_time DESC
LIMIT :limit OFFSET :offset;

-- name: CountDeployments :one
SELECT COUNT(*) AS count
FROM deployments;

-- name: CountDeploymentsByStatus :one
SELECT COUNT(*) AS count
FROM deployments
WHERE status = :status;

-- name: UpdateDeploymentStatus :one
UPDATE deployments
SET status = :status,
    finished_time = CASE
        WHEN :status = 'COMPLETED' OR :status = 'FAILED' THEN datetime('now', 'subsec')
        ELSE finished_time
    END
WHERE id = :id
RETURNING *;

-- name: DeleteDeployment :execrows
DELETE FROM deployments
WHERE id = :id;
