package user

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Model struct {
	ID        common.UserID `json:"id"`
	Email     string        `json:"email"`
	PassHash  []byte        `json:"-"`
	CreatedAt time.Time     `json:"created_at"`
}
