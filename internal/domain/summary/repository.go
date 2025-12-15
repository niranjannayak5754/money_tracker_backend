package summary

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Repository interface {
	IncomeTotal(
		ctx context.Context,
		userID primitive.ObjectID,
		start, end time.Time,
	) (float64, error)

	ExpenseTotals(
		ctx context.Context,
		userID primitive.ObjectID,
		start, end time.Time,
	) (float64, []CategoryBreakdown, error)
}
