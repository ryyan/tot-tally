// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0
// source: query.sql

package totdb

import (
	"context"
	"time"
)

const createBaby = `-- name: CreateBaby :one
INSERT INTO babies (id, name, timezone)
VALUES (?, ?, ?)
RETURNING id, name, timezone
`

type CreateBabyParams struct {
	ID       string
	Name     string
	Timezone string
}

func (q *Queries) CreateBaby(ctx context.Context, arg CreateBabyParams) (Baby, error) {
	row := q.db.QueryRowContext(ctx, createBaby, arg.ID, arg.Name, arg.Timezone)
	var i Baby
	err := row.Scan(&i.ID, &i.Name, &i.Timezone)
	return i, err
}

const createFeed = `-- name: CreateFeed :one
INSERT INTO feeds (id, baby_id, created_at, note, ounces)
VALUES (?, ?, ?, ?, ?)
RETURNING id, baby_id, created_at, note, ounces
`

type CreateFeedParams struct {
	ID        string
	BabyID    string
	CreatedAt time.Time
	Note      string
	Ounces    int64
}

func (q *Queries) CreateFeed(ctx context.Context, arg CreateFeedParams) (Feed, error) {
	row := q.db.QueryRowContext(ctx, createFeed,
		arg.ID,
		arg.BabyID,
		arg.CreatedAt,
		arg.Note,
		arg.Ounces,
	)
	var i Feed
	err := row.Scan(
		&i.ID,
		&i.BabyID,
		&i.CreatedAt,
		&i.Note,
		&i.Ounces,
	)
	return i, err
}

const createSoil = `-- name: CreateSoil :one
INSERT INTO soils (id, baby_id, created_at, note, wet, soil)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, baby_id, created_at, note, wet, soil
`

type CreateSoilParams struct {
	ID        string
	BabyID    string
	CreatedAt time.Time
	Note      string
	Wet       string
	Soil      string
}

func (q *Queries) CreateSoil(ctx context.Context, arg CreateSoilParams) (Soil, error) {
	row := q.db.QueryRowContext(ctx, createSoil,
		arg.ID,
		arg.BabyID,
		arg.CreatedAt,
		arg.Note,
		arg.Wet,
		arg.Soil,
	)
	var i Soil
	err := row.Scan(
		&i.ID,
		&i.BabyID,
		&i.CreatedAt,
		&i.Note,
		&i.Wet,
		&i.Soil,
	)
	return i, err
}

const getBaby = `-- name: GetBaby :one
SELECT id, name, timezone
FROM babies
WHERE id = ?
LIMIT 1
`

func (q *Queries) GetBaby(ctx context.Context, id string) (Baby, error) {
	row := q.db.QueryRowContext(ctx, getBaby, id)
	var i Baby
	err := row.Scan(&i.ID, &i.Name, &i.Timezone)
	return i, err
}

const listFeeds = `-- name: ListFeeds :many
SELECT id, baby_id, created_at, note, ounces
FROM feeds
WHERE baby_id = ?
ORDER BY created_at DESC
LIMIT 10
`

func (q *Queries) ListFeeds(ctx context.Context, babyID string) ([]Feed, error) {
	rows, err := q.db.QueryContext(ctx, listFeeds, babyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Feed
	for rows.Next() {
		var i Feed
		if err := rows.Scan(
			&i.ID,
			&i.BabyID,
			&i.CreatedAt,
			&i.Note,
			&i.Ounces,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listSoils = `-- name: ListSoils :many
SELECT id, baby_id, created_at, note, wet, soil
FROM soils
WHERE baby_id = ?
ORDER BY created_at DESC
LIMIT 10
`

func (q *Queries) ListSoils(ctx context.Context, babyID string) ([]Soil, error) {
	rows, err := q.db.QueryContext(ctx, listSoils, babyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Soil
	for rows.Next() {
		var i Soil
		if err := rows.Scan(
			&i.ID,
			&i.BabyID,
			&i.CreatedAt,
			&i.Note,
			&i.Wet,
			&i.Soil,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateTimezone = `-- name: UpdateTimezone :one
UPDATE babies
SET timezone = ?
WHERE id = ?
RETURNING id, name, timezone
`

type UpdateTimezoneParams struct {
	Timezone string
	ID       string
}

func (q *Queries) UpdateTimezone(ctx context.Context, arg UpdateTimezoneParams) (Baby, error) {
	row := q.db.QueryRowContext(ctx, updateTimezone, arg.Timezone, arg.ID)
	var i Baby
	err := row.Scan(&i.ID, &i.Name, &i.Timezone)
	return i, err
}
