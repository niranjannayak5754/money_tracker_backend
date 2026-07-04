package bankaccountrepo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/bankaccount"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	mongohelper "github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
	auditrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/audit"
)

type MongoRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
	audit  *auditrepo.AuditRepo
}

func New(db *mongo.Database, logger *slog.Logger) bankaccount.Repository {
	return &MongoRepo{
		col:    db.Collection("bank_accounts"),
		logger: logger.With("repo", "bankaccount"),
		audit:  auditrepo.New(db, logger),
	}
}

type mongoBankAccount struct {
	ID           primitive.ObjectID   `bson:"_id"`
	UserID       primitive.ObjectID   `bson:"user_id"`
	Name         string               `bson:"name"`
	Balance      primitive.Decimal128 `bson:"balance"`
	InterestRate *float64             `bson:"interest_rate,omitempty"`
	CreatedAt    time.Time            `bson:"created_at"`
	UpdatedAt    time.Time            `bson:"updated_at"`
	DeletedAt    *time.Time           `bson:"deleted_at,omitempty"`
	DeletedBy    *primitive.ObjectID  `bson:"deleted_by,omitempty"`
}

func (m *MongoRepo) Create(ctx context.Context, acc bankaccount.Model) (common.BankAccountID, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(acc.UserID))
	if err != nil {
		return "", err
	}

	dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", acc.Balance))
	if err != nil {
		return "", err
	}

	doc := mongoBankAccount{
		ID:           primitive.NewObjectID(),
		UserID:       uid,
		Name:         acc.Name,
		Balance:      dec,
		InterestRate: acc.InterestRate,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	_, err = m.col.InsertOne(ctx, doc)
	if err != nil {
		m.logger.Error("mongo insert failed", "request_id", requestctx.RequestID(ctx), "uid", acc.UserID, "err", err)
		return "", err
	}

	_ = m.audit.Write(ctx, auditrepo.Entry{
		Action:   "create",
		Entity:   "bank_account",
		EntityID: doc.ID.Hex(),
		Payload: bson.M{
			"user_id": doc.UserID.Hex(),
			"name":    doc.Name,
			"balance": acc.Balance,
		},
	})

	return common.BankAccountID(doc.ID.Hex()), nil
}

func (m *MongoRepo) List(ctx context.Context, userID common.UserID) ([]bankaccount.Model, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return nil, err
	}

	filter := bson.M{"user_id": uid, "deleted_at": bson.M{"$exists": false}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cur, err := m.col.Find(ctx, filter, opts)
	if err != nil {
		m.logger.Error("mongo find failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
		return nil, err
	}
	defer cur.Close(ctx)

	var out []bankaccount.Model
	for cur.Next(ctx) {
		var ma mongoBankAccount
		if err := cur.Decode(&ma); err != nil {
			m.logger.Error("mongo decode failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
			return nil, err
		}
		out = append(out, bankaccount.Model{
			ID:           common.BankAccountID(ma.ID.Hex()),
			UserID:       common.UserID(ma.UserID.Hex()),
			Name:         ma.Name,
			Balance:      shared.Decimal128ToFloat(ma.Balance),
			InterestRate: ma.InterestRate,
			CreatedAt:    ma.CreatedAt,
			UpdatedAt:    ma.UpdatedAt,
		})
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (m *MongoRepo) Update(ctx context.Context, userID common.UserID, id common.BankAccountID, set map[string]any) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}
	aid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	converted := bson.M{}
	for k, v := range set {
		if k == "balance" {
			f, ok := v.(float64)
			if !ok {
				return false, fmt.Errorf("balance must be a float64")
			}
			dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", f))
			if err != nil {
				return false, err
			}
			converted["balance"] = dec
			continue
		}
		converted[k] = v
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{"_id": aid, "user_id": uid, "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": converted},
	)
	if err != nil {
		m.logger.Error("mongo update failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "bank_account_id", id, "err", err)
		return false, err
	}

	if res.MatchedCount > 0 {
		_ = m.audit.Write(ctx, auditrepo.Entry{Action: "update", Entity: "bank_account", EntityID: string(id), Payload: bson.M{"set": set}})
	}

	return res.MatchedCount > 0, nil
}

func (m *MongoRepo) Delete(ctx context.Context, userID common.UserID, id common.BankAccountID) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}
	aid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{"_id": aid, "user_id": uid, "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"deleted_at": time.Now().UTC(), "deleted_by": uid}},
	)
	if err != nil {
		m.logger.Error("mongo soft-delete failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "bank_account_id", id, "err", err)
		return false, err
	}

	if res.MatchedCount > 0 {
		_ = m.audit.Write(ctx, auditrepo.Entry{Action: "delete", Entity: "bank_account", EntityID: string(id), Payload: bson.M{"deleted_at": time.Now().UTC()}})
	}

	return res.MatchedCount > 0, nil
}

// BalanceTotal sums non-deleted account balances for a user via a
// Mongo-side $sum for precision, matching the summary repo's pattern.
func (m *MongoRepo) BalanceTotal(ctx context.Context, userID common.UserID) (float64, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return 0, err
	}

	cur, err := m.col.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: bson.M{"user_id": uid, "deleted_at": bson.M{"$exists": false}}}},
		bson.D{{Key: "$group", Value: bson.M{"_id": nil, "total": bson.M{"$sum": "$balance"}}}},
	})
	if err != nil {
		m.logger.Error("mongo aggregate balance total failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
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
