package domain

// DTO для запросов/ответов API

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,password"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// AuthRegisterResponse (201)
type AuthRegisterResponse struct {
	User         AuthUserBrief `json:"user"`
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
}

type AuthUserBrief struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// AuthLoginResponse (200)
type AuthLoginResponse struct {
	User         AuthUserFull `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

type AuthUserFull struct {
	ID        int     `json:"id"`
	Email     string  `json:"email"`
	Username  *string `json:"username,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Role      string  `json:"role"`
}

// AuthRefreshResponse (200)
type AuthRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Внутреннее использование (issueTokens)
type TokenResponse struct {
	AccessToken  string
	RefreshToken string
}

// UserProfileResponse (200) — GET /users/me
type UserProfileResponse struct {
	ID        int            `json:"id"`
	Email     string         `json:"email"`
	Username  *string        `json:"username,omitempty"`
	AvatarURL *string        `json:"avatar_url,omitempty"`
	Role      string         `json:"role"`
	Stats     UserStats      `json:"stats"`
	CreatedAt string         `json:"created_at"`
}

type UserStats struct {
	DecksCount int `json:"decks_count"`
	CardsCount int `json:"cards_count"`
}

type UpdateProfileRequest struct {
	Username *string `json:"username,omitempty"`
}

// DecksListResponse (200) — GET /decks
type DecksListResponse struct {
	Decks      []DeckListItem `json:"decks"`
	Pagination Pagination     `json:"pagination"`
}

type DeckListItem struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Category    *Category `json:"category,omitempty"`
	Tags        []Tag      `json:"tags,omitempty"`
	IsPublic    bool       `json:"is_public"`
	CardsCount  int        `json:"cards_count"`
	CreatedAt   string     `json:"created_at"`
}

type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

type CreateDeckRequest struct {
	Title       string  `json:"title" binding:"required,max=255"`
	Description *string `json:"description,omitempty"`
	CategoryID  *int    `json:"category_id,omitempty"`
	IsPublic    bool    `json:"is_public"`
	TagIDs      []int   `json:"tag_ids,omitempty"`
}

type UpdateDeckRequest struct {
	Title       *string `json:"title,omitempty" binding:"omitempty,max=255"`
	Description *string `json:"description,omitempty"`
	CategoryID  *int    `json:"category_id,omitempty"`
	IsPublic    *bool   `json:"is_public,omitempty"`
	TagIDs      []int   `json:"tag_ids,omitempty"`
}

type CreateCardRequest struct {
	DeckID     *int   `json:"deck_id,omitempty"` // обязателен для POST /api/cards
	Question   string `json:"question" binding:"required"`
	Answer     string `json:"answer" binding:"required"`
	CategoryID *int   `json:"category_id,omitempty"`
	TagIDs     []int  `json:"tag_ids,omitempty"`
}

// CardListItem — элемент списка GET /api/cards
type CardListItem struct {
	ID        int        `json:"id"`
	Question  string     `json:"question"`
	Answer    string     `json:"answer"`
	Deck      DeckBrief  `json:"deck"`
	Category  *Category  `json:"category,omitempty"`
	Tags      []Tag      `json:"tags,omitempty"`
	CreatedAt string     `json:"created_at"`
}

type DeckBrief struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// CardsListResponse (200) — GET /api/cards
type CardsListResponse struct {
	Cards      []CardListItem `json:"cards"`
	Pagination Pagination     `json:"pagination"`
}

type UpdateCardRequest struct {
	Question   *string `json:"question,omitempty"`
	Answer     *string `json:"answer,omitempty"`
	CategoryID *int    `json:"category_id,omitempty"`
	TagIDs     []int   `json:"tag_ids,omitempty"`
}

// CategoriesResponse (200) — GET /api/categories
type CategoriesResponse struct {
	Categories []Category `json:"categories"`
}

type CreateCategoryRequest struct {
	Name string `json:"name" binding:"required,max=100"`
}

// TagsResponse (200) — GET /api/tags
type TagsResponse struct {
	Tags []Tag `json:"tags"`
}

type CreateTagRequest struct {
	Name string `json:"name" binding:"required,max=100"`
}

// PublicDeckListItem — элемент GET /api/public/decks
type PublicDeckListItem struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Category    *Category  `json:"category,omitempty"`
	Tags        []Tag      `json:"tags,omitempty"`
	CardsCount  int        `json:"cards_count"`
	Author      DeckAuthor `json:"author"`
	CreatedAt   string     `json:"created_at"`
}

type DeckAuthor struct {
	ID        int     `json:"id"`
	Username  *string `json:"username,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// PublicDeckDetail — GET /api/public/decks/:id
type PublicDeckDetail struct {
	ID          int             `json:"id"`
	Title       string          `json:"title"`
	Description *string         `json:"description,omitempty"`
	Category    *Category       `json:"category,omitempty"`
	Tags        []Tag           `json:"tags,omitempty"`
	CardsCount  int             `json:"cards_count"`
	Author      DeckAuthor      `json:"author"`
	Cards       []PublicCardItem `json:"cards"`
}

type PublicCardItem struct {
	ID       int    `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// PublicDecksListResponse (200)
type PublicDecksListResponse struct {
	Decks      []PublicDeckListItem `json:"decks"`
	Pagination Pagination           `json:"pagination"`
}
