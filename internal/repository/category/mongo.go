package categoryrepo

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/category"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
)

type MongoRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
}

func New(
	db *mongo.Database,
	logger *slog.Logger,
) category.Repository {
	return &MongoRepo{
		col:    db.Collection("categories"),
		logger: logger.With("repo", "category"),
	}
}

// Create inserts a new category document.
func (m *MongoRepo) Create(
	ctx context.Context,
	cat category.Model,
) error {

	_, err := m.col.InsertOne(ctx, cat)
	if err != nil {
		m.logger.Error(
			"mongo insert failed",
			"request_id", requestctx.UID(ctx),
			"user_id", cat.UserID.Hex(),
			"err", err,
		)
		return err
	}

	return nil
}

// ListActive returns all non-archived categories for a user.
func (m *MongoRepo) ListActive(
	ctx context.Context,
	uid primitive.ObjectID,
) ([]category.Model, error) {

	filter := bson.M{
		"user_id":  uid,
		"archived": bson.M{"$ne": true},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cur, err := m.col.Find(ctx, filter, opts)
	if err != nil {
		m.logger.Error(
			"mongo find failed",
			"request_id", requestctx.UID(ctx),
			"user_id", uid.Hex(),
			"err", err,
		)
		return nil, err
	}
	defer cur.Close(ctx)

	var out []category.Model
	for cur.Next(ctx) {
		var v category.Model
		if err := cur.Decode(&v); err != nil {
			m.logger.Error(
				"mongo decode failed",
				"request_id", requestctx.UID(ctx),
				"user_id", uid.Hex(),
				"err", err,
			)
			return nil, err
		}
		out = append(out, v)
	}

	if err := cur.Err(); err != nil {
		m.logger.Error(
			"mongo cursor error",
			"request_id", requestctx.UID(ctx),
			"user_id", uid.Hex(),
			"err", err,
		)
		return nil, err
	}

	return out, nil
}

// Update modifies category fields.
func (m *MongoRepo) Update(
	ctx context.Context,
	uid, id primitive.ObjectID,
	set map[string]any,
) (bool, error) {

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{
			"_id":     id,
			"user_id": uid,
		},
		bson.M{"$set": set},
	)

	if err != nil {
		m.logger.Error(
			"mongo update failed",
			"request_id", requestctx.UID(ctx),
			"user_id", uid.Hex(),
			"category_id", id.Hex(),
			"err", err,
		)
		return false, err
	}

	return res.MatchedCount > 0, nil
}

// ExistsForUser checks if a category belongs to the user and is not archived.
func (m *MongoRepo) ExistsForUser(
	ctx context.Context,
	uid, categoryID primitive.ObjectID,
) (bool, error) {

	filter := bson.M{
		"_id":      categoryID,
		"user_id":  uid,
		"archived": bson.M{"$ne": true},
	}

	count, err := m.col.CountDocuments(ctx, filter)
	if err != nil {
		m.logger.Error(
			"mongo count failed",
			"request_id", requestctx.UID(ctx),
			"user_id", uid.Hex(),
			"category_id", categoryID.Hex(),
			"err", err,
		)
		return false, err
	}

	return count > 0, nil
}
