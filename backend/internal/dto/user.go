package dto

import (
	"time"

	"carsharing/internal/models"
)

type RegisterRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	FullName      string `json:"full_name"`
	DriverLicense string `json:"driver_license"`
}

type User struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	FullName      string    `json:"full_name"`
	DriverLicense *string   `json:"driver_license"`
	Role          string    `json:"role"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewUser(u models.User) User {
	return User{
		ID:            u.ID,
		Email:         u.Email,
		FullName:      u.FullName,
		DriverLicense: u.DriverLicense,
		Role:          u.Role,
		CreatedAt:     u.CreatedAt,
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
	ExpiresIn int    `json:"expires_in"`
	User      User   `json:"user"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type UserRoleRequest struct {
	Role string `json:"role"`
}

type UpdateUserRequest struct {
	FullName      *string `json:"full_name"`
	DriverLicense *string `json:"driver_license"`
	Email         *string `json:"email"`
}
