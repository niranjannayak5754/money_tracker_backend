package expenserepo

import (
	"context"
	"time"

	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
)

type mongoRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
}

func New(db *mongo.Database, logger *slog.Logger) expense.Repository {
	return &mongoRepo{
		col:    db.Collection("expenses"),
		logger: logger.With("repo", "expense"),
	}
}

// Create inserts a new expense document.
func (m *mongoRepo) Create(
	ctx context.Context,
	exp expense.Model,
) error {
	_, err := m.col.InsertOne(ctx, exp)
	if err != nil {
		m.logger.Error(
			"mongo insert failed",
			"request_id", requestctx.RequestID(ctx),
			"user_id", exp.UserID.Hex(),
			"err", err,
		)
	}
	return err
}

// ListByMonth returns expenses for a user within a month and optional category.
func (m *mongoRepo) ListByMonth(
	ctx context.Context,
	userID primitive.ObjectID,
	start, end time.Time,
	categoryID *primitive.ObjectID,
) ([]expense.Model, error) {

	filter := bson.M{
		"user_id": userID,
		"date": bson.M{
			"$gte": start,
			"$lt":  end,
		},
	}

	if categoryID != nil {
		filter["category_id"] = *categoryID
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "date", Value: -1}})

	cur, err := m.col.Find(ctx, filter, opts)
	if err != nil {
		m.logger.Error(
			"mongo find failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID.Hex(),
			"err", err,
		)
		return nil, err
	}
	defer cur.Close(ctx)

	var out []expense.Model
	for cur.Next(ctx) {
		var exp expense.Model
		if err := cur.Decode(&exp); err != nil {
			m.logger.Error(
				"mongo decode failed",
				"request_id", requestctx.RequestID(ctx),
				"uid", userID.Hex(),
				"err", err,
			)
			return nil, err
		}
		out = append(out, exp)
	}

	if err := cur.Err(); err != nil {
		m.logger.Error(
			"mongo cursor error",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID.Hex(),
			"err", err,
		)
		return nil, err
	}

	return out, nil
}

// Update applies partial updates to an expense.
func (m *mongoRepo) Update(
	ctx context.Context,
	userID, id primitive.ObjectID,
	set map[string]any,
) (bool, error) {
	res, err := m.col.UpdateOne(
		ctx,
		bson.M{
			"_id":     id,
			"user_id": userID,
		},
		bson.M{
			"$set": set,
		},
	)
	if err != nil {
		m.logger.Error(
			"mongo update failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID.Hex(),
			"expense_id", id.Hex(),
			"err", err,
		)
		return false, err
	}

	return res.MatchedCount > 0, nil
}

// Delete removes an expense.
func (m *mongoRepo) Delete(
	ctx context.Context,
	userID, id primitive.ObjectID,
) (bool, error) {

	res, err := m.col.DeleteOne(
		ctx,
		bson.M{
			"_id":     id,
			"user_id": userID,
		},
	)
	if err != nil {
		m.logger.Error(
			"mongo delete failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID.Hex(),
			"expense_id", id.Hex(),
			"err", err,
		)
		return false, err
	}

	return res.DeletedCount > 0, nil
}
