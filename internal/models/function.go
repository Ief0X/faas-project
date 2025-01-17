package models

type Function struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	OwnerId string `json:"ownerId"`
	Image   string `json:"image"`
}
