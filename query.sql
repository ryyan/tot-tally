-- name: CreateBaby :one
INSERT INTO babies (id, name)
VALUES (?, ?)
RETURNING *;

-- name: GetBaby :one
SELECT *
FROM babies
WHERE id = ?
LIMIT 1;

-- name: CreateFeed :one
INSERT INTO feeds (id, baby_id, created_at, note, ounces)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListFeeds :many
SELECT *
FROM feeds
WHERE baby_id = ?
ORDER BY created_at DESC
LIMIT 30;

-- name: CreateSoil :one
INSERT INTO soils (id, baby_id, created_at, note, wet, soil)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListSoils :many
SELECT *
FROM soils
WHERE baby_id = ?
ORDER BY created_at DESC
LIMIT 30;