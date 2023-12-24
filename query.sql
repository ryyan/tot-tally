-- name: CreateTot :one
INSERT INTO tots (id, name, timezone)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetTot :one
SELECT *
FROM tots
WHERE id = ?
LIMIT 1;

-- name: UpdateTimezone :one
UPDATE tots
SET timezone = ?
WHERE id = ?
RETURNING *;

-- name: CreateTally :one
INSERT INTO tallies (tot_id, created_at, kind)
VALUES (?, ?, ?)
RETURNING *;

-- name: ListTallies :many
SELECT created_at, kind
FROM tallies
WHERE tot_id = ?
ORDER BY created_at DESC
LIMIT 100;

-- name: GetLastMilkTime :one
SELECT created_at
FROM tallies
WHERE tot_id = ?
    AND kind LIKE 'Milk%'
ORDER BY created_at DESC
LIMIT 1;

-- name: GetLastFoodTime :one
SELECT created_at
FROM tallies
WHERE tot_id = ?
    AND kind LIKE 'Food%'
ORDER BY created_at DESC
LIMIT 1;

-- name: GetLastWetTime :one
SELECT created_at
FROM tallies
WHERE tot_id = ?
    AND (
        kind = 'Soil'
        OR kind = 'Wet & Soil'
    )
ORDER BY created_at DESC
LIMIT 1;

-- name: GetLastSoilTime :one
SELECT created_at
FROM tallies
WHERE tot_id = ?
    AND (
        kind = 'Wet'
        OR kind = 'Wet & Soil'
    )
ORDER BY created_at DESC
LIMIT 1;

-- name: GetLastBathTime :one
SELECT created_at
FROM tallies
WHERE tot_id = ?
    AND kind = 'Bath'
ORDER BY created_at DESC
LIMIT 1;

-- name: GetLastToothbrushTime :one
SELECT created_at
FROM tallies
WHERE tot_id = ?
    AND kind = 'Toothbrush'
ORDER BY created_at DESC
LIMIT 1;