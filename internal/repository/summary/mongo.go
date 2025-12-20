package summaryrepo

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/summary"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
)

type MongoRepo struct {
	incomeCol   *mongo.Collection
	expensesCol *mongo.Collection
	logger      *slog.Logger
}

func New(
	db *mongo.Database,
	logger *slog.Logger,
) summary.Repository {
	return &MongoRepo{
		incomeCol:   db.Collection("income"),
		expensesCol: db.Collection("expenses"),
		logger:      logger.With("repo", "summary"),
	}
}

func (m *MongoRepo) IncomeTotal(
	ctx context.Context,
	uid primitive.ObjectID,
	start, end time.Time,
) (float64, error) {

	cur, err := m.incomeCol.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: bson.M{
			"user_id": uid,
			"date":    bson.M{"$gte": start, "$lt": end},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$amount"},
		}}},
	})
	if err != nil {
		m.logger.Error(
			"mongo aggregate income failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", uid.Hex(),
			"err", err,
		)
		return 0, err
	}
	defer cur.Close(ctx)

	var out struct {
		Total primitive.Decimal128 `bson:"total"`
	}

	if cur.Next(ctx) {
		if err := cur.Decode(&out); err != nil {
			m.logger.Error(
				"mongo decode income total failed",
				"request_id", requestctx.RequestID(ctx),
				"uid", uid.Hex(),
				"err", err,
			)
			return 0, err
		}
		return shared.Decimal128ToFloat(out.Total), nil
	}

	return 0, nil
}

func (m *MongoRepo) ExpenseTotals(
	ctx context.Context,
	uid primitive.ObjectID,
	start, end time.Time,
) (float64, []summary.CategoryBreakdown, error) {

	cur, err := m.expensesCol.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: bson.M{
			"user_id": uid,
			"date":    bson.M{"$gte": start, "$lt": end},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   "$category_id",
			"total": bson.M{"$sum": "$amount"},
		}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "categories",
			"localField":   "_id",
			"foreignField": "_id",
			"as":           "cat",
		}}},
		bson.D{{Key: "$unwind", Value: bson.M{
			"path":                       "$cat",
			"preserveNullAndEmptyArrays": true,
		}}},
	})
	if err != nil {
		m.logger.Error(
			"mongo aggregate expenses failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", uid.Hex(),
			"err", err,
		)
		return 0, nil, err
	}
	defer cur.Close(ctx)

	var total float64
	var cats []summary.CategoryBreakdown

	for cur.Next(ctx) {
		var x struct {
			CatID primitive.ObjectID   `bson:"_id"`
			Total primitive.Decimal128 `bson:"total"`
			Cat   struct {
				Name string `bson:"name"`
			} `bson:"cat"`
		}

		if err := cur.Decode(&x); err != nil {
			m.logger.Error(
				"mongo decode expense breakdown failed",
				"request_id", requestctx.RequestID(ctx),
				"uid", uid.Hex(),
				"err", err,
			)
			return 0, nil, err
		}

		f := shared.Decimal128ToFloat(x.Total)
		total += f

		cats = append(cats, summary.CategoryBreakdown{
			CategoryID:   x.CatID.Hex(),
			CategoryName: x.Cat.Name,
			Total:        f,
		})
	}

	return total, cats, nil
}
