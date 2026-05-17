package users

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleCustomer Role = "customer"
	RoleManager  Role = "manager"
	RoleAdmin    Role = "admin"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
