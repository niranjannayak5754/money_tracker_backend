package user

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Repository interface {
	Create(ctx context.Context, u Model) error
	FindByEmail(ctx context.Context, email string) (*Model, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*Model, error)
	UpdatePasswordHash(ctx context.Context, id primitive.ObjectID, hash []byte) error
}
