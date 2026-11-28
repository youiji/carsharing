package models

import "time"

const (
	RoleClient  = "client"
	RoleManager = "manager"
	RoleAdmin   = "admin"
)

type User struct {
	ID            int64
	Email         string
	PasswordHash  string
	FullName      string
	DriverLicense *string
	Role          string
	CreatedAt     time.Time
}

var UserRoles = []string{RoleClient, RoleManager, RoleAdmin}

type UserFilter struct {
	Role   *string
	Search string // email и ФИО
	Sort   string
	Desc   bool
}
