// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0

package totdb

import (
	"time"
)

type Baby struct {
	ID       string
	Name     string
	Timezone string
}

type Feed struct {
	ID        string
	BabyID    string
	CreatedAt time.Time
	Note      string
	Ounces    int64
}

type Soil struct {
	ID        string
	BabyID    string
	CreatedAt time.Time
	Note      string
	Wet       int64
	Soil      int64
}
