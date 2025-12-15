package category

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Repository interface {
	Create(ctx context.Context, m Model) error
	Update(
		ctx context.Context,
		userID, id primitive.ObjectID,
		set map[string]any,
	) (bool, error)
	ListActive(ctx context.Context, userID primitive.ObjectID) ([]Model, error)
	ExistsForUser(
		ctx context.Context,
		userID, categoryID primitive.ObjectID,
	) (bool, error)
}
