package categoryrepo

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/category"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	mongohelper "github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
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

type mongoCategory struct {
	ID        primitive.ObjectID `bson:"_id"`
	UserID    primitive.ObjectID `bson:"user_id"`
	Name      string             `bson:"name"`
	Type      string             `bson:"type"`
	Archived  bool               `bson:"archived"`
	CreatedAt time.Time          `bson:"created_at"`
}

// Create inserts a new category document.
func (m *MongoRepo) Create(
	ctx context.Context,
	cat category.Model,
) error {

	uid, err := mongohelper.ObjectIDFromHex(string(cat.UserID))
	if err != nil {
		return err
	}

	doc := mongoCategory{
		ID:        primitive.NewObjectID(),
		UserID:    uid,
		Name:      cat.Name,
		Type:      cat.Type,
		Archived:  cat.Archived,
		CreatedAt: time.Now().UTC(),
	}

	_, err = m.col.InsertOne(ctx, doc)
	if err != nil {
		m.logger.Error(
			"mongo insert failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", cat.UserID,
			"err", err,
		)
		return err
	}

	return nil
}

// List returns all categories for a user.
func (m *MongoRepo) List(
	ctx context.Context,
	userID common.UserID,
	archived *bool,
) ([]category.Model, error) {

	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"user_id": uid,
	}

	if archived != nil {
		filter["archived"] = *archived
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cur, err := m.col.Find(ctx, filter, opts)
	if err != nil {
		m.logger.Error(
			"mongo find failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"err", err,
		)
		return nil, err
	}
	defer cur.Close(ctx)

	var out []category.Model
	for cur.Next(ctx) {
		var mc mongoCategory
		if err := cur.Decode(&mc); err != nil {
			m.logger.Error(
				"mongo decode failed",
				"request_id", requestctx.RequestID(ctx),
				"uid", userID,
				"err", err,
			)
			return nil, err
		}

		out = append(out, category.Model{
			ID:        common.CategoryID(mc.ID.Hex()),
			UserID:    common.UserID(mc.UserID.Hex()),
			Name:      mc.Name,
			Type:      mc.Type,
			Archived:  mc.Archived,
			CreatedAt: mc.CreatedAt,
		})
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

// Update modifies category fields.
func (m *MongoRepo) Update(
	ctx context.Context,
	userID common.UserID,
	id common.CategoryID,
	set map[string]any,
) (bool, error) {

	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}

	cid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{
			"_id":     cid,
			"user_id": uid,
		},
		bson.M{"$set": set},
	)

	if err != nil {
		m.logger.Error(
			"mongo update failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"category_id", id,
			"err", err,
		)
		return false, err
	}

	return res.MatchedCount > 0, nil
}

// ExistsForUser checks if a category belongs to the user and is not archived.
func (m *MongoRepo) ExistsForUser(
	ctx context.Context,
	userID common.UserID,
	categoryID common.CategoryID,
) (bool, error) {

	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}

	cid, err := mongohelper.ObjectIDFromHex(string(categoryID))
	if err != nil {
		return false, err
	}

	filter := bson.M{
		"_id":      cid,
		"user_id":  uid,
		"archived": bson.M{"$ne": true},
	}

	count, err := m.col.CountDocuments(ctx, filter)
	if err != nil {
		m.logger.Error(
			"mongo count failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"category_id", categoryID,
			"err", err,
		)
		return false, err
	}

	return count > 0, nil
}
