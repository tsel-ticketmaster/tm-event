package admin

import (
	"time"
)

type SignInResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type CreateResponse struct {
	ID int64 `json:"id"`
}
