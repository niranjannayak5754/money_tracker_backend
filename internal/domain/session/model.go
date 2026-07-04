package session

import (
	"time"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

// RefreshToken represents an issued refresh token. Only TokenHash is ever
// persisted; the plain token value is handed to the client once at issuance.
type RefreshToken struct {
	ID        common.SessionID
	UserID    common.UserID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}
