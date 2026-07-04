package session

import (
	"context"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	Create(ctx context.Context, rt RefreshToken) error

	// FindValidByHash returns the token only if it is neither expired nor
	// revoked; repository.ErrNotFound otherwise.
	FindValidByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)

	RevokeByHash(ctx context.Context, tokenHash string) error
	RevokeAllForUser(ctx context.Context, userID common.UserID) error
}
