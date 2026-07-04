package budgetrepo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/budget"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	mongohelper "github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
)

type MongoRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
}

func New(db *mongo.Database, logger *slog.Logger) budget.Repository {
	return &MongoRepo{
		col:    db.Collection("budgets"),
		logger: logger.With("repo", "budget"),
	}
}

type mongoBudget struct {
	ID            primitive.ObjectID   `bson:"_id"`
	UserID        primitive.ObjectID   `bson:"user_id"`
	CategoryID    *primitive.ObjectID  `bson:"category_id"`
	Amount        primitive.Decimal128 `bson:"amount"`
	EffectiveFrom string               `bson:"effective_from"`
	CreatedAt     time.Time            `bson:"created_at"`
	UpdatedAt     time.Time            `bson:"updated_at"`
}

func (m *MongoRepo) Set(ctx context.Context, b budget.Model) (common.BudgetID, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(b.UserID))
	if err != nil {
		return "", err
	}

	var catID *primitive.ObjectID
	if b.CategoryID != nil {
		cid, err := mongohelper.ObjectIDFromHex(string(*b.CategoryID))
		if err != nil {
			return "", err
		}
		catID = &cid
	}

	dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", b.Amount))
	if err != nil {
		return "", err
	}

	filter := bson.M{"user_id": uid, "category_id": catID, "effective_from": b.EffectiveFrom}
	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{"amount": dec, "updated_at": now},
		"$setOnInsert": bson.M{
			"_id":            primitive.NewObjectID(),
			"user_id":        uid,
			"category_id":    catID,
			"effective_from": b.EffectiveFrom,
			"created_at":     now,
		},
	}

	res, err := m.col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		m.logger.Error("mongo upsert budget failed", "request_id", requestctx.RequestID(ctx), "uid", b.UserID, "err", err)
		return "", err
	}

	if res.UpsertedID != nil {
		if oid, ok := res.UpsertedID.(primitive.ObjectID); ok {
			return common.BudgetID(oid.Hex()), nil
		}
	}

	var existing struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	if err := m.col.FindOne(ctx, filter).Decode(&existing); err != nil {
		return "", err
	}
	return common.BudgetID(existing.ID.Hex()), nil
}

func (m *MongoRepo) List(ctx context.Context, userID common.UserID) ([]budget.Model, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return nil, err
	}

	cur, err := m.col.Find(ctx, bson.M{"user_id": uid})
	if err != nil {
		m.logger.Error("mongo find failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
		return nil, err
	}
	defer cur.Close(ctx)

	var out []budget.Model
	for cur.Next(ctx) {
		var mb mongoBudget
		if err := cur.Decode(&mb); err != nil {
			m.logger.Error("mongo decode failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
			return nil, err
		}

		var catID *common.CategoryID
		if mb.CategoryID != nil {
			c := common.CategoryID(mb.CategoryID.Hex())
			catID = &c
		}

		out = append(out, budget.Model{
			ID:            common.BudgetID(mb.ID.Hex()),
			UserID:        common.UserID(mb.UserID.Hex()),
			CategoryID:    catID,
			Amount:        shared.Decimal128ToFloat(mb.Amount),
			EffectiveFrom: mb.EffectiveFrom,
			CreatedAt:     mb.CreatedAt,
			UpdatedAt:     mb.UpdatedAt,
		})
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (m *MongoRepo) Delete(ctx context.Context, userID common.UserID, id common.BudgetID) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}
	bid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	res, err := m.col.DeleteOne(ctx, bson.M{"_id": bid, "user_id": uid})
	if err != nil {
		m.logger.Error("mongo delete failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "budget_id", id, "err", err)
		return false, err
	}

	return res.DeletedCount > 0, nil
}
