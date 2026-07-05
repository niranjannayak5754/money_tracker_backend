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
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
	auditrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/audit"
)

type MongoRepo struct {
	col       *mongo.Collection
	ledgerCol *mongo.Collection
	logger    *slog.Logger
	audit     *auditrepo.AuditRepo
}

func New(db *mongo.Database, logger *slog.Logger) bankaccount.Repository {
	return &MongoRepo{
		col:       db.Collection("bank_accounts"),
		ledgerCol: db.Collection("bank_account_ledger"),
		logger:    logger.With("repo", "bankaccount"),
		audit:     auditrepo.New(db, logger),
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

type mongoLedgerEntry struct {
	ID            primitive.ObjectID   `bson:"_id"`
	UserID        primitive.ObjectID   `bson:"user_id"`
	BankAccountID primitive.ObjectID   `bson:"bank_account_id"`
	Type          string               `bson:"type"`
	Amount        primitive.Decimal128 `bson:"amount"`
	Note          string               `bson:"note,omitempty"`
	BalanceAfter  primitive.Decimal128 `bson:"balance_after"`
	CreatedAt     time.Time            `bson:"created_at"`
}

// insertLedgerEntry records one dated balance change. Best-effort like the
// audit log — a failure here shouldn't roll back a balance change that
// already succeeded, but is logged loudly since losing a ledger row breaks
// the user-facing history.
func (m *MongoRepo) insertLedgerEntry(ctx context.Context, uid, aid primitive.ObjectID, entryType string, amount, balanceAfter float64, note string, at time.Time) {
	amountDec, err1 := primitive.ParseDecimal128(fmt.Sprintf("%.2f", amount))
	balanceDec, err2 := primitive.ParseDecimal128(fmt.Sprintf("%.2f", balanceAfter))
	if err1 != nil || err2 != nil {
		m.logger.Error("failed to encode ledger entry amounts", "bank_account_id", aid.Hex(), "err1", err1, "err2", err2)
		return
	}

	_, err := m.ledgerCol.InsertOne(ctx, mongoLedgerEntry{
		ID:            primitive.NewObjectID(),
		UserID:        uid,
		BankAccountID: aid,
		Type:          entryType,
		Amount:        amountDec,
		Note:          note,
		BalanceAfter:  balanceDec,
		CreatedAt:     at,
	})
	if err != nil {
		m.logger.Error("mongo insert ledger entry failed", "request_id", requestctx.RequestID(ctx), "bank_account_id", aid.Hex(), "err", err)
	}
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

	m.insertLedgerEntry(ctx, uid, doc.ID, string(bankaccount.LedgerOpening), acc.Balance, acc.Balance, "Initial balance", doc.CreatedAt)

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

// GetByID returns a single non-deleted bank account belonging to the user.
func (m *MongoRepo) GetByID(ctx context.Context, userID common.UserID, id common.BankAccountID) (*bankaccount.Model, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return nil, err
	}
	aid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return nil, err
	}

	var ma mongoBankAccount
	err = m.col.FindOne(ctx, bson.M{"_id": aid, "user_id": uid, "deleted_at": bson.M{"$exists": false}}).Decode(&ma)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, repository.ErrNotFound
		}
		m.logger.Error("mongo find by id failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "bank_account_id", id, "err", err)
		return nil, err
	}

	return &bankaccount.Model{
		ID:           common.BankAccountID(ma.ID.Hex()),
		UserID:       common.UserID(ma.UserID.Hex()),
		Name:         ma.Name,
		Balance:      shared.Decimal128ToFloat(ma.Balance),
		InterestRate: ma.InterestRate,
		CreatedAt:    ma.CreatedAt,
		UpdatedAt:    ma.UpdatedAt,
	}, nil
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

// Adjust atomically applies delta to an account's balance via $inc — for a
// withdrawal (delta < 0), the filter requires balance >= -delta so the
// match (and thus the update) fails atomically rather than racing a
// separate read-then-write against a concurrent withdrawal.
func (m *MongoRepo) Adjust(ctx context.Context, userID common.UserID, id common.BankAccountID, delta float64, note string) (float64, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return 0, err
	}
	aid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return 0, err
	}

	incDec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", delta))
	if err != nil {
		return 0, err
	}

	filter := bson.M{"_id": aid, "user_id": uid, "deleted_at": bson.M{"$exists": false}}
	if delta < 0 {
		minBalance, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", -delta))
		if err != nil {
			return 0, err
		}
		filter["balance"] = bson.M{"$gte": minBalance}
	}

	now := time.Now().UTC()
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var ma mongoBankAccount
	err = m.col.FindOneAndUpdate(ctx, filter,
		bson.M{"$inc": bson.M{"balance": incDec}, "$set": bson.M{"updated_at": now}},
		opts,
	).Decode(&ma)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, repository.ErrNotFound
		}
		m.logger.Error("mongo adjust balance failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "bank_account_id", id, "err", err)
		return 0, err
	}

	newBalance := shared.Decimal128ToFloat(ma.Balance)

	direction := "add"
	entryType := bankaccount.LedgerAdd
	if delta < 0 {
		direction = "withdraw"
		entryType = bankaccount.LedgerWithdraw
	}
	_ = m.audit.Write(ctx, auditrepo.Entry{
		Action:   "adjust",
		Entity:   "bank_account",
		EntityID: string(id),
		Payload: bson.M{
			"direction":     direction,
			"amount":        delta,
			"note":          note,
			"balance_after": newBalance,
		},
	})

	amountAbs := delta
	if amountAbs < 0 {
		amountAbs = -amountAbs
	}
	m.insertLedgerEntry(ctx, uid, aid, string(entryType), amountAbs, newBalance, note, now)

	return newBalance, nil
}

// ListLedger returns an account's ledger entries in ascending created_at
// order (opening entry first, like a bank statement). Zero start/end
// means all-time.
func (m *MongoRepo) ListLedger(ctx context.Context, userID common.UserID, id common.BankAccountID, start, end time.Time) ([]bankaccount.LedgerEntry, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return nil, err
	}
	aid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return nil, err
	}

	filter := bson.M{"user_id": uid, "bank_account_id": aid}
	if !start.IsZero() || !end.IsZero() {
		filter["created_at"] = bson.M{"$gte": start, "$lt": end}
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cur, err := m.ledgerCol.Find(ctx, filter, opts)
	if err != nil {
		m.logger.Error("mongo find ledger failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "bank_account_id", id, "err", err)
		return nil, err
	}
	defer cur.Close(ctx)

	var out []bankaccount.LedgerEntry
	for cur.Next(ctx) {
		var le mongoLedgerEntry
		if err := cur.Decode(&le); err != nil {
			m.logger.Error("mongo decode ledger entry failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
			return nil, err
		}
		out = append(out, bankaccount.LedgerEntry{
			ID:            common.LedgerEntryID(le.ID.Hex()),
			UserID:        common.UserID(le.UserID.Hex()),
			BankAccountID: common.BankAccountID(le.BankAccountID.Hex()),
			Type:          bankaccount.LedgerEntryType(le.Type),
			Amount:        shared.Decimal128ToFloat(le.Amount),
			Note:          le.Note,
			BalanceAfter:  shared.Decimal128ToFloat(le.BalanceAfter),
			CreatedAt:     le.CreatedAt,
		})
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
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
