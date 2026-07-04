package expenserepo

import (
	"context"
	"fmt"
	"time"

	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/expense"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	mongohelper "github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
	auditrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/audit"
)

type mongoRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
	audit  *auditrepo.AuditRepo
}

func New(db *mongo.Database, logger *slog.Logger) expense.Repository {
	return &mongoRepo{
		col:    db.Collection("expenses"),
		logger: logger.With("repo", "expense"),
		audit:  auditrepo.New(db, logger),
	}
}

type mongoExpense struct {
	ID         primitive.ObjectID   `bson:"_id"`
	UserID     primitive.ObjectID   `bson:"user_id"`
	Amount     primitive.Decimal128 `bson:"amount"`
	Date       time.Time            `bson:"date"`
	CategoryID primitive.ObjectID   `bson:"category_id"`
	Merchant   string               `bson:"merchant,omitempty"`
	Notes      string               `bson:"notes,omitempty"`
	Tags       []string             `bson:"tags,omitempty"`
	CreatedAt  time.Time            `bson:"created_at"`
	UpdatedAt  time.Time            `bson:"updated_at"`
}

// Create inserts a new expense document.
func (m *mongoRepo) Create(
	ctx context.Context,
	exp expense.Model,
) (common.ExpenseID, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(exp.UserID))
	if err != nil {
		return "", err
	}

	// convert amount to Decimal128
	dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", exp.Amount))
	if err != nil {
		return "", err
	}

	cid, err := mongohelper.ObjectIDFromHex(string(exp.CategoryID))
	if err != nil {
		return "", err
	}

	doc := mongoExpense{
		ID:         primitive.NewObjectID(),
		UserID:     uid,
		Amount:     dec,
		Date:       exp.Date,
		CategoryID: cid,
		Merchant:   exp.Merchant,
		Notes:      exp.Notes,
		Tags:       exp.Tags,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	_, err = m.col.InsertOne(ctx, doc)
	if err != nil {
		m.logger.Error(
			"mongo insert failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", exp.UserID,
			"err", err,
		)
	}

	// write audit entry
	_ = m.audit.Write(ctx, auditrepo.Entry{
		Action:   "create",
		Entity:   "expense",
		EntityID: doc.ID.Hex(),
		Payload: bson.M{
			"user_id":     doc.UserID.Hex(),
			"amount":      shared.Decimal128ToFloat(doc.Amount),
			"date":        doc.Date,
			"category_id": doc.CategoryID.Hex(),
			"merchant":    doc.Merchant,
			"notes":       doc.Notes,
			"tags":        doc.Tags,
		},
	})

	if err != nil {
		return "", err
	}
	return common.ExpenseID(doc.ID.Hex()), nil
}

// ListByMonth returns expenses for a user within a month and optional category.
func (m *mongoRepo) ListByMonth(
	ctx context.Context,
	userID common.UserID,
	start, end time.Time,
	categoryID *common.CategoryID,
) ([]expense.Model, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return nil, err
	}

	filter := bson.M{"user_id": uid, "deleted_at": bson.M{"$exists": false}}
	if !start.IsZero() || !end.IsZero() {
		filter["date"] = bson.M{"$gte": start, "$lt": end}
	}

	if categoryID != nil {
		cid, err := mongohelper.ObjectIDFromHex(string(*categoryID))
		if err != nil {
			return nil, err
		}
		filter["category_id"] = cid
	}
	m.logger.Info(`checking filter`, slog.Any("filter", filter))

	opts := options.Find().
		SetSort(bson.D{{Key: "date", Value: -1}})

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

	var out []expense.Model
	for cur.Next(ctx) {
		var me mongoExpense
		if err := cur.Decode(&me); err != nil {
			m.logger.Error(
				"mongo decode failed",
				"request_id", requestctx.RequestID(ctx),
				"uid", userID,
				"err", err,
			)
			return nil, err
		}

		out = append(out, expense.Model{
			ID:         common.ExpenseID(me.ID.Hex()),
			UserID:     common.UserID(me.UserID.Hex()),
			Amount:     shared.Decimal128ToFloat(me.Amount),
			Date:       me.Date,
			CategoryID: common.CategoryID(me.CategoryID.Hex()),
			Merchant:   me.Merchant,
			Notes:      me.Notes,
			Tags:       me.Tags,
			CreatedAt:  me.CreatedAt,
			UpdatedAt:  me.UpdatedAt,
		})
	}

	if err := cur.Err(); err != nil {
		m.logger.Error(
			"mongo cursor error",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"err", err,
		)
		return nil, err
	}

	return out, nil
}

// Update applies partial updates to an expense.
func (m *mongoRepo) Update(
	ctx context.Context,
	userID common.UserID,
	id common.ExpenseID,
	set map[string]any,
) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}

	eid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}
	// convert domain-friendly set values into mongo types
	converted := bson.M{}
	for k, v := range set {
		switch k {
		case "amount":
			if f, ok := v.(float64); ok {
				dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", f))
				if err != nil {
					return false, err
				}
				converted["amount"] = dec
			} else {
				converted[k] = v
			}
		case "category_id":
			if cid, ok := v.(common.CategoryID); ok {
				oid, err := mongohelper.ObjectIDFromHex(string(cid))
				if err != nil {
					return false, err
				}
				converted["category_id"] = oid
			} else if s, ok := v.(string); ok {
				oid, err := mongohelper.ObjectIDFromHex(s)
				if err != nil {
					return false, err
				}
				converted["category_id"] = oid
			} else {
				converted[k] = v
			}
		default:
			converted[k] = v
		}
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{
			"_id":        eid,
			"user_id":    uid,
			"deleted_at": bson.M{"$exists": false},
		},
		bson.M{
			"$set": converted,
		},
	)

	if err != nil {
		m.logger.Error(
			"mongo update failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"expense_id", id,
			"err", err,
		)
		return false, err
	}

	// write audit entry
	if res.MatchedCount > 0 {
		_ = m.audit.Write(ctx, auditrepo.Entry{
			Action:   "update",
			Entity:   "expense",
			EntityID: string(id),
			Payload:  bson.M{"set": set},
		})
	}

	return res.MatchedCount > 0, nil
}

// Delete removes an expense.
func (m *mongoRepo) Delete(
	ctx context.Context,
	userID common.UserID,
	id common.ExpenseID,
) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}

	eid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{
			"_id":     eid,
			"user_id": uid,
		},
		bson.M{"$set": bson.M{"deleted_at": time.Now().UTC(), "deleted_by": uid}},
	)
	if err != nil {
		m.logger.Error(
			"mongo soft-delete failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"expense_id", id,
			"err", err,
		)
		return false, err
	}

	if res.MatchedCount > 0 {
		_ = m.audit.Write(ctx, auditrepo.Entry{
			Action:   "delete",
			Entity:   "expense",
			EntityID: string(id),
			Payload: bson.M{
				"deleted_at": time.Now().UTC(),
			},
		})
	}

	return res.MatchedCount > 0, nil
}
