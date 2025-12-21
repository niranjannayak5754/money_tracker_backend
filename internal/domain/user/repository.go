package user

import (
	"context"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
)

type Repository interface {
	Create(ctx context.Context, u Model) error
	FindByEmail(ctx context.Context, email string) (*Model, error)
	FindByID(ctx context.Context, id common.UserID) (*Model, error)
	UpdatePasswordHash(ctx context.Context, id common.UserID, hash []byte) error
}
