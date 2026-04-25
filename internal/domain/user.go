package domain

import "time"

type User struct {
	ID           int        `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Username     *string    `json:"username,omitempty"`
	AvatarURL    *string    `json:"avatar_url,omitempty"`
	Role         string     `json:"role"`
	IsBlocked    bool       `json:"is_blocked"`
	BlockedAt    *time.Time `json:"blocked_at,omitempty"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	TokenVersion int        `json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type UserRole string

const (
	RoleUser      UserRole = "user"
	RoleModerator UserRole = "moderator"
	RoleAdmin     UserRole = "admin"
)

func IsValidRole(r string) bool {
	switch UserRole(r) {
	case RoleUser, RoleModerator, RoleAdmin:
		return true
	}
	return false
}
