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
	incomeCol       *mongo.Collection
	expensesCol     *mongo.Collection
	investmentsCol  *mongo.Collection
	bankAccountsCol *mongo.Collection
	debtsCol        *mongo.Collection
	logger          *slog.Logger
}

func New(
	db *mongo.Database,
	logger *slog.Logger,
) summary.Repository {
	return &MongoRepo{
		incomeCol:       db.Collection("income"),
		expensesCol:     db.Collection("expenses"),
		investmentsCol:  db.Collection("investments"),
		bankAccountsCol: db.Collection("bank_accounts"),
		debtsCol:        db.Collection("debts"),
		logger:          logger.With("repo", "summary"),
	}
}

// BankBalanceTotal sums current non-deleted bank account balances for a user.
func (m *MongoRepo) BankBalanceTotal(ctx context.Context, userID common.UserID) (float64, error) {
	return m.sumField(ctx, m.bankAccountsCol, userID, "balance", "bank balance")
}

// DebtOutstandingTotal sums current non-deleted debts' outstanding balances for a user.
func (m *MongoRepo) DebtOutstandingTotal(ctx context.Context, userID common.UserID) (float64, error) {
	return m.sumField(ctx, m.debtsCol, userID, "outstanding_balance", "debt outstanding")
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

func (m *MongoRepo) IncomeTotal(
	ctx context.Context,
	userId common.UserID,
	start, end time.Time,
) (float64, error) {

	uid, err := mongohelper.ObjectIDFromHex(string(userId))
	if err != nil {
		return 0, err
	}

	match := bson.M{"user_id": uid, "deleted_at": bson.M{"$exists": false}}
	if !start.IsZero() || !end.IsZero() {
		match["date"] = bson.M{"$gte": start, "$lt": end}
	}

	cur, err := m.incomeCol.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: match}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$amount"},
		}}},
	})
	if err != nil {
		m.logger.Error(
			"mongo aggregate income failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userId,
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
				"uid", userId,
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

func (m *MongoRepo) InvestmentTotals(
	ctx context.Context,
	userId common.UserID,
	start, end time.Time,
) (float64, float64, error) {

	uid, err := mongohelper.ObjectIDFromHex(string(userId))
	if err != nil {
		return 0, 0, err
	}

	match := bson.M{"user_id": uid, "deleted_at": bson.M{"$exists": false}}
	if !start.IsZero() || !end.IsZero() {
		match["date"] = bson.M{"$gte": start, "$lt": end}
	}

	// Group both sums in one aggregation: sum(amount) and sum(realized_pnl)
	cur, err := m.investmentsCol.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: match}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":                nil,
			"total_amount":       bson.M{"$sum": "$amount"},
			"total_realized_pnl": bson.M{"$sum": "$realized_pnl"},
		}}},
	})
	if err != nil {
		m.logger.Error(
			"mongo aggregate investments failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userId,
			"err", err,
		)
		return 0, 0, err
	}
	defer cur.Close(ctx)

	var out struct {
		TotalAmount   primitive.Decimal128 `bson:"total_amount"`
		TotalRealized primitive.Decimal128 `bson:"total_realized_pnl"`
	}

	if cur.Next(ctx) {
		if err := cur.Decode(&out); err != nil {
			m.logger.Error(
				"mongo decode investments total failed",
				"request_id", requestctx.RequestID(ctx),
				"uid", userId,
				"err", err,
			)
			return 0, 0, err
		}
		return shared.Decimal128ToFloat(out.TotalAmount), shared.Decimal128ToFloat(out.TotalRealized), nil
	}

	return 0, 0, nil
}
