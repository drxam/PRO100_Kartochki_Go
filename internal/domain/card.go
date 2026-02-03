package domain

import "time"

type Card struct {
	ID         int        `json:"id"`
	DeckID     int        `json:"deck_id"`
	Question   string     `json:"question"`
	Answer     string     `json:"answer"`
	CategoryID *int       `json:"category_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Category   *Category  `json:"category,omitempty"`
	Tags       []Tag      `json:"tags,omitempty"`
}
