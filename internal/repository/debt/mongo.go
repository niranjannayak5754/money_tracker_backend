package debtrepo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/debt"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	mongohelper "github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
	auditrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/audit"
)

type MongoRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
	audit  *auditrepo.AuditRepo
}

func New(db *mongo.Database, logger *slog.Logger) debt.Repository {
	return &MongoRepo{
		col:    db.Collection("debts"),
		logger: logger.With("repo", "debt"),
		audit:  auditrepo.New(db, logger),
	}
}

type mongoDebt struct {
	ID                 primitive.ObjectID   `bson:"_id"`
	UserID             primitive.ObjectID   `bson:"user_id"`
	Name               string               `bson:"name"`
	Principal          primitive.Decimal128 `bson:"principal"`
	InterestRate       float64              `bson:"interest_rate"`
	EMIAmount          primitive.Decimal128 `bson:"emi_amount,omitempty"`
	TenureMonths       int                  `bson:"tenure_months,omitempty"`
	StartDate          time.Time            `bson:"start_date"`
	OutstandingBalance primitive.Decimal128 `bson:"outstanding_balance"`
	TotalInterestPaid  primitive.Decimal128 `bson:"total_interest_paid"`
	Status             string               `bson:"status"`
	Notes              string               `bson:"notes,omitempty"`
	CreatedAt          time.Time            `bson:"created_at"`
	UpdatedAt          time.Time            `bson:"updated_at"`
	DeletedAt          *time.Time           `bson:"deleted_at,omitempty"`
	DeletedBy          *primitive.ObjectID  `bson:"deleted_by,omitempty"`
}

func toModel(md mongoDebt) debt.Model {
	return debt.Model{
		ID:                 common.DebtID(md.ID.Hex()),
		UserID:             common.UserID(md.UserID.Hex()),
		Name:               md.Name,
		Principal:          shared.Decimal128ToFloat(md.Principal),
		InterestRate:       md.InterestRate,
		EMIAmount:          shared.Decimal128ToFloat(md.EMIAmount),
		TenureMonths:       md.TenureMonths,
		StartDate:          md.StartDate,
		OutstandingBalance: shared.Decimal128ToFloat(md.OutstandingBalance),
		TotalInterestPaid:  shared.Decimal128ToFloat(md.TotalInterestPaid),
		Status:             debt.Status(md.Status),
		Notes:              md.Notes,
		CreatedAt:          md.CreatedAt,
		UpdatedAt:          md.UpdatedAt,
	}
}

func (m *MongoRepo) Create(ctx context.Context, d debt.Model) (common.DebtID, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(d.UserID))
	if err != nil {
		return "", err
	}

	principalDec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", d.Principal))
	if err != nil {
		return "", err
	}
	emiDec, _ := primitive.ParseDecimal128(fmt.Sprintf("%.2f", d.EMIAmount))
	outstandingDec, _ := primitive.ParseDecimal128(fmt.Sprintf("%.2f", d.OutstandingBalance))
	interestPaidDec, _ := primitive.ParseDecimal128(fmt.Sprintf("%.2f", d.TotalInterestPaid))

	doc := mongoDebt{
		ID:                 primitive.NewObjectID(),
		UserID:             uid,
		Name:               d.Name,
		Principal:          principalDec,
		InterestRate:       d.InterestRate,
		EMIAmount:          emiDec,
		TenureMonths:       d.TenureMonths,
		StartDate:          d.StartDate,
		OutstandingBalance: outstandingDec,
		TotalInterestPaid:  interestPaidDec,
		Status:             string(d.Status),
		Notes:              d.Notes,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	_, err = m.col.InsertOne(ctx, doc)
	if err != nil {
		m.logger.Error("mongo insert failed", "request_id", requestctx.RequestID(ctx), "uid", d.UserID, "err", err)
		return "", err
	}

	_ = m.audit.Write(ctx, auditrepo.Entry{
		Action:   "create",
		Entity:   "debt",
		EntityID: doc.ID.Hex(),
		Payload: bson.M{
			"user_id":   doc.UserID.Hex(),
			"name":      doc.Name,
			"principal": d.Principal,
		},
	})

	return common.DebtID(doc.ID.Hex()), nil
}

func (m *MongoRepo) List(ctx context.Context, userID common.UserID) ([]debt.Model, error) {
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

	var out []debt.Model
	for cur.Next(ctx) {
		var md mongoDebt
		if err := cur.Decode(&md); err != nil {
			m.logger.Error("mongo decode failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
			return nil, err
		}
		out = append(out, toModel(md))
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (m *MongoRepo) GetByID(ctx context.Context, userID common.UserID, id common.DebtID) (*debt.Model, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return nil, err
	}
	did, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return nil, err
	}

	var md mongoDebt
	err = m.col.FindOne(ctx, bson.M{"_id": did, "user_id": uid, "deleted_at": bson.M{"$exists": false}}).Decode(&md)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, repository.ErrNotFound
		}
		m.logger.Error("mongo find by id failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "debt_id", id, "err", err)
		return nil, err
	}

	model := toModel(md)
	return &model, nil
}

func (m *MongoRepo) Update(ctx context.Context, userID common.UserID, id common.DebtID, set map[string]any) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}
	did, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	decimalFields := map[string]bool{
		"outstanding_balance": true,
		"total_interest_paid": true,
		"emi_amount":          true,
	}

	converted := bson.M{}
	for k, v := range set {
		if decimalFields[k] {
			f, ok := v.(float64)
			if !ok {
				return false, fmt.Errorf("%s must be a float64", k)
			}
			dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", f))
			if err != nil {
				return false, err
			}
			converted[k] = dec
			continue
		}
		converted[k] = v
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{"_id": did, "user_id": uid, "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": converted},
	)
	if err != nil {
		m.logger.Error("mongo update failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "debt_id", id, "err", err)
		return false, err
	}

	if res.MatchedCount > 0 {
		_ = m.audit.Write(ctx, auditrepo.Entry{Action: "update", Entity: "debt", EntityID: string(id), Payload: bson.M{"set": set}})
	}

	return res.MatchedCount > 0, nil
}

func (m *MongoRepo) Delete(ctx context.Context, userID common.UserID, id common.DebtID) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}
	did, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{"_id": did, "user_id": uid, "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"deleted_at": time.Now().UTC(), "deleted_by": uid}},
	)
	if err != nil {
		m.logger.Error("mongo soft-delete failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "debt_id", id, "err", err)
		return false, err
	}

	if res.MatchedCount > 0 {
		_ = m.audit.Write(ctx, auditrepo.Entry{Action: "delete", Entity: "debt", EntityID: string(id), Payload: bson.M{"deleted_at": time.Now().UTC()}})
	}

	return res.MatchedCount > 0, nil
}
