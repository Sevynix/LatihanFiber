package model

import "time"

const DefaultRole = "student"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

type AssignRoleRequest struct {
	Role string `json:"role"`
}

type UpdateUserRequest struct {
	Email *string `json:"email"`
}