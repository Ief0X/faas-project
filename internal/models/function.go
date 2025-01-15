package models

import "time"

type Function struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	OwnerId string `json:"ownerId"`
	Image   string `json:"image"`
	LastExecution time.Time `db:"last_execution"`
	LastResult    string    `db:"last_result"`
	NextExecution time.Time `db:"next_execution"`
}
