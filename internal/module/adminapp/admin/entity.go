package admin

import (
	"time"
)

const (
	StatusActive   = "ACTIVE"
	StatusInactive = "INACTIVE"
)

type Administrator struct {
	ID           int64
	Name         string
	Email        string
	Password     string
	PasswordSalt string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
