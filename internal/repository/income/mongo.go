package incomerepo

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/income"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
)

type MongoRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
}

func New(
	db *mongo.Database,
	logger *slog.Logger,
) income.Repository {
	return &MongoRepo{
		col:    db.Collection("income"),
		logger: logger.With("repo", "income"),
	}
}

func (m *MongoRepo) Create(ctx context.Context, rec income.Model) error {
	_, err := m.col.InsertOne(ctx, rec)
	if err != nil {
		m.logger.Error(
			"mongo insert failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", rec.UserID.Hex(),
			"err", err,
		)
	}
	return err
}

func (m *MongoRepo) ListByMonth(
	ctx context.Context,
	uid primitive.ObjectID,
	start, end time.Time,
) ([]income.Model, error) {

	filter := bson.M{
		"user_id": uid,
		"date": bson.M{
			"$gte": start,
			"$lt":  end,
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "date", Value: -1}})

	cur, err := m.col.Find(ctx, filter, opts)
	if err != nil {
		m.logger.Error(
			"mongo find failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", uid.Hex(),
			"err", err,
		)
		return nil, err
	}
	defer cur.Close(ctx)

	var out []income.Model
	for cur.Next(ctx) {
		var v income.Model
		if err := cur.Decode(&v); err != nil {
			m.logger.Error(
				"mongo decode failed",
				"request_id", requestctx.RequestID(ctx),
				"uid", uid.Hex(),
				"err", err,
			)
			return nil, err
		}
		out = append(out, v)
	}

	return out, cur.Err()
}

func (m *MongoRepo) Update(
	ctx context.Context,
	uid, id primitive.ObjectID,
	set map[string]any,
) (bool, error) {

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{"_id": id, "user_id": uid},
		bson.M{"$set": set},
	)

	if err != nil {
		m.logger.Error(
			"mongo update failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", uid.Hex(),
			"income_id", id.Hex(),
			"err", err,
		)
		return false, err
	}

	return res.MatchedCount > 0, nil
}

func (m *MongoRepo) Delete(
	ctx context.Context,
	uid, id primitive.ObjectID,
) (bool, error) {

	res, err := m.col.DeleteOne(
		ctx,
		bson.M{"_id": id, "user_id": uid},
	)

	if err != nil {
		m.logger.Error(
			"mongo delete failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", uid.Hex(),
			"income_id", id.Hex(),
			"err", err,
		)
		return false, err
	}

	return res.DeletedCount > 0, nil
}
