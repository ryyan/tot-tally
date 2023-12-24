// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.21.0
// source: query.sql

package totdb

import (
	"context"
	"time"
)

const createTally = `-- name: CreateTally :one
INSERT INTO tallies (tot_id, created_at, kind)
VALUES (?, ?, ?)
RETURNING id, tot_id, created_at, kind
`

type CreateTallyParams struct {
	TotID     string
	CreatedAt time.Time
	Kind      string
}

func (q *Queries) CreateTally(ctx context.Context, arg CreateTallyParams) (Tally, error) {
	row := q.db.QueryRowContext(ctx, createTally, arg.TotID, arg.CreatedAt, arg.Kind)
	var i Tally
	err := row.Scan(
		&i.ID,
		&i.TotID,
		&i.CreatedAt,
		&i.Kind,
	)
	return i, err
}

const createTot = `-- name: CreateTot :one
INSERT INTO tots (id, name, timezone)
VALUES (?, ?, ?)
RETURNING id, name, timezone
`

type CreateTotParams struct {
	ID       string
	Name     string
	Timezone string
}

func (q *Queries) CreateTot(ctx context.Context, arg CreateTotParams) (Tot, error) {
	row := q.db.QueryRowContext(ctx, createTot, arg.ID, arg.Name, arg.Timezone)
	var i Tot
	err := row.Scan(&i.ID, &i.Name, &i.Timezone)
	return i, err
}

const getLastBathTime = `-- name: GetLastBathTime :one
SELECT created_at
FROM tallies
WHERE tot_id = ?
    AND kind = 'Bath'
ORDER BY created_at DESC
LIMIT 1
`

func (q *Queries) GetLastBathTime(ctx context.Context, totID string) (time.Time, error) {
	row := q.db.QueryRowContext(ctx, getLastBathTime, totID)
	var created_at time.Time
	err := row.Scan(&created_at)
	return created_at, err
}

const getLastFoodTime = `-- name: GetLastFoodTime :one
SELECT created_at
FROM tallies
WHERE tot_id = ?
    AND kind LIKE 'Food%'
ORDER BY created_at DESC
LIMIT 1
`

func (q *Queries) GetLastFoodTime(ctx context.Context, totID string) (time.Time, error) {
	row := q.db.QueryRowContext(ctx, getLastFoodTime, totID)
	var created_at time.Time
	err := row.Scan(&created_at)
	return created_at, err
}

const getLastMilkTime = `-- name: GetLastMilkTime :one
SELECT created_at
FROM tallies
WHERE tot_id = ?
    AND kind LIKE 'Milk%'
ORDER BY created_at DESC
LIMIT 1
`

func (q *Queries) GetLastMilkTime(ctx context.Context, totID string) (time.Time, error) {
	row := q.db.QueryRowContext(ctx, getLastMilkTime, totID)
	var created_at time.Time
	err := row.Scan(&created_at)
	return created_at, err
}

const getLastSoilTime = `-- name: GetLastSoilTime :one
SELECT created_at
FROM tallies
WHERE tot_id = ?
    AND (
        kind = 'Wet'
        OR kind = 'Wet & Soil'
    )
ORDER BY created_at DESC
LIMIT 1
`

func (q *Queries) GetLastSoilTime(ctx context.Context, totID string) (time.Time, error) {
	row := q.db.QueryRowContext(ctx, getLastSoilTime, totID)
	var created_at time.Time
	err := row.Scan(&created_at)
	return created_at, err
}

const getLastToothbrushTime = `-- name: GetLastToothbrushTime :one
SELECT created_at
FROM tallies
WHERE tot_id = ?
    AND kind = 'Toothbrush'
ORDER BY created_at DESC
LIMIT 1
`

func (q *Queries) GetLastToothbrushTime(ctx context.Context, totID string) (time.Time, error) {
	row := q.db.QueryRowContext(ctx, getLastToothbrushTime, totID)
	var created_at time.Time
	err := row.Scan(&created_at)
	return created_at, err
}

const getLastWetTime = `-- name: GetLastWetTime :one
SELECT created_at
FROM tallies
WHERE tot_id = ?
    AND (
        kind = 'Soil'
        OR kind = 'Wet & Soil'
    )
ORDER BY created_at DESC
LIMIT 1
`

func (q *Queries) GetLastWetTime(ctx context.Context, totID string) (time.Time, error) {
	row := q.db.QueryRowContext(ctx, getLastWetTime, totID)
	var created_at time.Time
	err := row.Scan(&created_at)
	return created_at, err
}

const getTot = `-- name: GetTot :one
SELECT id, name, timezone
FROM tots
WHERE id = ?
LIMIT 1
`

func (q *Queries) GetTot(ctx context.Context, id string) (Tot, error) {
	row := q.db.QueryRowContext(ctx, getTot, id)
	var i Tot
	err := row.Scan(&i.ID, &i.Name, &i.Timezone)
	return i, err
}

const listTallies = `-- name: ListTallies :many
SELECT created_at, kind
FROM tallies
WHERE tot_id = ?
ORDER BY created_at DESC
LIMIT 100
`

type ListTalliesRow struct {
	CreatedAt time.Time
	Kind      string
}

func (q *Queries) ListTallies(ctx context.Context, totID string) ([]ListTalliesRow, error) {
	rows, err := q.db.QueryContext(ctx, listTallies, totID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListTalliesRow
	for rows.Next() {
		var i ListTalliesRow
		if err := rows.Scan(&i.CreatedAt, &i.Kind); err != nil {
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
UPDATE tots
SET timezone = ?
WHERE id = ?
RETURNING id, name, timezone
`

type UpdateTimezoneParams struct {
	Timezone string
	ID       string
}

func (q *Queries) UpdateTimezone(ctx context.Context, arg UpdateTimezoneParams) (Tot, error) {
	row := q.db.QueryRowContext(ctx, updateTimezone, arg.Timezone, arg.ID)
	var i Tot
	err := row.Scan(&i.ID, &i.Name, &i.Timezone)
	return i, err
}