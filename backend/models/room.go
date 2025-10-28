package models

type Room struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	OwnerID string          `json:"owner_id"`
	Members map[string]bool `json:"members"`
}
