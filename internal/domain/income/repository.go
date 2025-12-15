package income

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Repository interface {
	Create(ctx context.Context, m Model) error
	ListByMonth(
		ctx context.Context,
		userID primitive.ObjectID,
		start, end time.Time,
	) ([]Model, error)
	Update(
		ctx context.Context,
		userID, id primitive.ObjectID,
		set map[string]any,
	) (bool, error)
	Delete(
		ctx context.Context,
		userID, id primitive.ObjectID,
	) (bool, error)
}
