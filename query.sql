-- name: CreateBaby :one
INSERT INTO babies (id, name, timezone)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetBaby :one
SELECT *
FROM babies
WHERE id = ?
LIMIT 1;

-- name: UpdateTimezone :one
UPDATE babies
SET timezone = ?
WHERE id = ?
RETURNING *;

-- name: CreateFeed :one
INSERT INTO feeds (baby_id, created_at, ounces, feed_type)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: ListFeeds :many
SELECT *
FROM feeds
WHERE baby_id = ?
ORDER BY created_at DESC
LIMIT 10;

-- name: CreateSoil :one
INSERT INTO soils (baby_id, created_at, wet, soil)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: ListSoils :many
SELECT *
FROM soils
WHERE baby_id = ?
ORDER BY created_at DESC
LIMIT 10;

-- name: GetLastMilkTime :one
SELECT created_at
FROM feeds
WHERE baby_id = ? AND feed_type = 0 AND ounces > 0
ORDER BY created_at DESC
LIMIT 1;

-- name: GetLastFoodTime :one
SELECT created_at
FROM feeds
WHERE baby_id = ? AND feed_type = 1 AND ounces > 0
ORDER BY created_at DESC
LIMIT 1;

-- name: GetLastWetTime :one
SELECT created_at
FROM soils
WHERE baby_id = ? AND wet = 1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetLastSoilTime :one
SELECT created_at
FROM soils
WHERE baby_id = ? AND soil = 1
ORDER BY created_at DESC
LIMIT 1;