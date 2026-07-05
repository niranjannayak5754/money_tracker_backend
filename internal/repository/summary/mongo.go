package summaryrepo

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/summary"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	mongohelper "github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
)

type MongoRepo struct {
	expensesCol     *mongo.Collection
	bankAccountsCol *mongo.Collection
	logger          *slog.Logger
}

func New(
	db *mongo.Database,
	logger *slog.Logger,
) summary.Repository {
	return &MongoRepo{
		expensesCol:     db.Collection("expenses"),
		bankAccountsCol: db.Collection("bank_accounts"),
		logger:          logger.With("repo", "summary"),
	}
}

// BankBalanceTotal sums current non-deleted bank (savings) account
// balances for a user.
func (m *MongoRepo) BankBalanceTotal(ctx context.Context, userID common.UserID) (float64, error) {
	return m.sumField(ctx, m.bankAccountsCol, userID, "balance", "bank balance")
}

func (m *MongoRepo) sumField(ctx context.Context, col *mongo.Collection, userID common.UserID, field, label string) (float64, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return 0, err
	}

	cur, err := col.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: bson.M{"user_id": uid, "deleted_at": bson.M{"$exists": false}}}},
		bson.D{{Key: "$group", Value: bson.M{"_id": nil, "total": bson.M{"$sum": "$" + field}}}},
	})
	if err != nil {
		m.logger.Error("mongo aggregate "+label+" failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
		return 0, err
	}
	defer cur.Close(ctx)

	var out struct {
		Total primitive.Decimal128 `bson:"total"`
	}
	if cur.Next(ctx) {
		if err := cur.Decode(&out); err != nil {
			return 0, err
		}
		return shared.Decimal128ToFloat(out.Total), nil
	}

	return 0, nil
}

func (m *MongoRepo) ExpenseTotals(
	ctx context.Context,
	userID common.UserID,
	start, end time.Time,
) (float64, []summary.CategoryBreakdown, error) {

	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return 0, nil, err
	}

	match := bson.M{"user_id": uid, "deleted_at": bson.M{"$exists": false}}
	if !start.IsZero() || !end.IsZero() {
		match["date"] = bson.M{"$gte": start, "$lt": end}
	}

	cur, err := m.expensesCol.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: match}},
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
			"uid", userID,
			"err", err,
		)
		return 0, nil, err
	}
	defer cur.Close(ctx)

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
				"uid", userID,
				"err", err,
			)
			return 0, nil, err
		}

		cats = append(cats, summary.CategoryBreakdown{
			CategoryID:   x.CatID.Hex(),
			CategoryName: x.Cat.Name,
			Total:        shared.Decimal128ToFloat(x.Total),
		})
	}

	if err := cur.Err(); err != nil {
		return 0, nil, err
	}

	// Grand total computed by Mongo ($sum over Decimal128), not by summing
	// already-rounded per-category floats in Go — avoids float drift.
	totalCur, err := m.expensesCol.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: match}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$amount"},
		}}},
	})
	if err != nil {
		m.logger.Error(
			"mongo aggregate expense total failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"err", err,
		)
		return 0, nil, err
	}
	defer totalCur.Close(ctx)

	var total float64
	var out struct {
		Total primitive.Decimal128 `bson:"total"`
	}
	if totalCur.Next(ctx) {
		if err := totalCur.Decode(&out); err != nil {
			m.logger.Error(
				"mongo decode expense total failed",
				"request_id", requestctx.RequestID(ctx),
				"uid", userID,
				"err", err,
			)
			return 0, nil, err
		}
		total = shared.Decimal128ToFloat(out.Total)
	}

	return total, cats, nil
}
