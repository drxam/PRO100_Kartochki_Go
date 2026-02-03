package domain

import "time"

type Deck struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	CategoryID  *int       `json:"category_id,omitempty"`
	IsPublic    bool       `json:"is_public"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Category    *Category  `json:"category,omitempty"`
	Tags        []Tag      `json:"tags,omitempty"`
	CardsCount  int        `json:"cards_count,omitempty"`
	Cards       []Card     `json:"cards,omitempty"`
}
