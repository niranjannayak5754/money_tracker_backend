package incomerepo

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
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/income"
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

func New(
	db *mongo.Database,
	logger *slog.Logger,
) income.Repository {
	return &MongoRepo{
		col:    db.Collection("income"),
		logger: logger.With("repo", "income"),
		audit:  auditrepo.New(db, logger),
	}
}

type mongoIncome struct {
	ID         primitive.ObjectID   `bson:"_id"`
	UserID     primitive.ObjectID   `bson:"user_id"`
	Amount     primitive.Decimal128 `bson:"amount"`
	Date       time.Time            `bson:"date"`
	CategoryID primitive.ObjectID   `bson:"category_id"`
	Source     string               `bson:"source,omitempty"`
	Notes      string               `bson:"notes,omitempty"`
	CreatedAt  time.Time            `bson:"created_at"`
	UpdatedAt  time.Time            `bson:"updated_at"`
	DeletedAt  *time.Time           `bson:"deleted_at,omitempty"`
	DeletedBy  *primitive.ObjectID  `bson:"deleted_by,omitempty"`
}

func (m *MongoRepo) Create(ctx context.Context, rec income.Model) (common.IncomeID, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(rec.UserID))
	if err != nil {
		return "", err
	}

	dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", rec.Amount))
	if err != nil {
		return "", err
	}

	cid, err := mongohelper.ObjectIDFromHex(string(rec.CategoryID))
	if err != nil {
		return "", err
	}

	doc := mongoIncome{
		ID:         primitive.NewObjectID(),
		UserID:     uid,
		Amount:     dec,
		Date:       rec.Date,
		CategoryID: cid,
		Source:     rec.Source,
		Notes:      rec.Notes,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	_, err = m.col.InsertOne(ctx, doc)
	if err != nil {
		m.logger.Error(
			"mongo insert failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", rec.UserID,
			"err", err,
		)
	}
	// write audit entry (best-effort)
	_ = m.audit.Write(ctx, auditrepo.Entry{
		Action:   "create",
		Entity:   "income",
		EntityID: doc.ID.Hex(),
		Payload: bson.M{
			"user_id":     doc.UserID.Hex(),
			"amount":      shared.Decimal128ToFloat(doc.Amount),
			"date":        doc.Date,
			"category_id": doc.CategoryID.Hex(),
			"source":      doc.Source,
			"notes":       doc.Notes,
		},
	})
	if err != nil {
		return "", err
	}
	return common.IncomeID(doc.ID.Hex()), nil
}

func (m *MongoRepo) ListByMonth(
	ctx context.Context,
	userId common.UserID,
	start, end time.Time,
) ([]income.Model, error) {

	uid, err := mongohelper.ObjectIDFromHex(string(userId))
	if err != nil {
		return nil, err
	}

	filter := bson.M{"user_id": uid, "deleted_at": bson.M{"$exists": false}}
	if !start.IsZero() || !end.IsZero() {
		filter["date"] = bson.M{"$gte": start, "$lt": end}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "date", Value: -1}})

	cur, err := m.col.Find(ctx, filter, opts)
	if err != nil {
		m.logger.Error(
			"mongo find failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userId,
			"err", err,
		)
		return nil, err
	}
	defer cur.Close(ctx)

	var out []income.Model
	for cur.Next(ctx) {
		var mi mongoIncome
		if err := cur.Decode(&mi); err != nil {
			m.logger.Error(
				"mongo decode failed",
				"request_id", requestctx.RequestID(ctx),
				"uid", userId,
				"err", err,
			)
			return nil, err
		}

		out = append(out, income.Model{
			ID:         common.IncomeID(mi.ID.Hex()),
			UserID:     common.UserID(mi.UserID.Hex()),
			Amount:     shared.Decimal128ToFloat(mi.Amount),
			Date:       mi.Date,
			CategoryID: common.CategoryID(mi.CategoryID.Hex()),
			Source:     mi.Source,
			Notes:      mi.Notes,
			CreatedAt:  mi.CreatedAt,
			UpdatedAt:  mi.UpdatedAt,
		})
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (m *MongoRepo) Update(
	ctx context.Context,
	userId common.UserID,
	id common.IncomeID,
	set map[string]any,
) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userId))
	if err != nil {
		return false, err
	}
	incomeId, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	converted := bson.M{}
	for k, v := range set {
		switch k {
		case "amount":
			f, ok := v.(float64)
			if !ok {
				return false, fmt.Errorf("amount must be a float64")
			}
			dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", f))
			if err != nil {
				return false, err
			}
			converted["amount"] = dec
		case "category_id":
			cid, ok := v.(common.CategoryID)
			if !ok {
				s, ok := v.(string)
				if !ok {
					return false, fmt.Errorf("category_id must be a string")
				}
				cid = common.CategoryID(s)
			}
			oid, err := mongohelper.ObjectIDFromHex(string(cid))
			if err != nil {
				return false, err
			}
			converted["category_id"] = oid
		default:
			converted[k] = v
		}
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{"_id": incomeId, "user_id": uid, "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": converted},
	)

	if err != nil {
		m.logger.Error(
			"mongo update failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userId,
			"income_id", id,
			"err", err,
		)
		return false, err
	}

	if res.MatchedCount > 0 {
		_ = m.audit.Write(ctx, auditrepo.Entry{
			Action:   "update",
			Entity:   "income",
			EntityID: string(id),
			Payload:  bson.M{"set": set},
		})
	}

	return res.MatchedCount > 0, nil
}

func (m *MongoRepo) Delete(
	ctx context.Context,
	userId common.UserID,
	id common.IncomeID,
) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userId))
	if err != nil {
		return false, err
	}
	incomeId, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{"_id": incomeId, "user_id": uid},
		bson.M{"$set": bson.M{"deleted_at": time.Now().UTC(), "deleted_by": uid}},
	)

	if err != nil {
		m.logger.Error(
			"mongo soft-delete failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userId,
			"income_id", id,
			"err", err,
		)
		return false, err
	}

	if res.MatchedCount > 0 {
		_ = m.audit.Write(ctx, auditrepo.Entry{
			Action:   "delete",
			Entity:   "income",
			EntityID: string(id),
			Payload:  bson.M{"deleted_at": time.Now().UTC()},
		})
	}

	return res.MatchedCount > 0, nil
}

// CountByCategory counts a user's non-deleted income entries referencing a
// category — used to guard category archiving.
func (m *MongoRepo) CountByCategory(
	ctx context.Context,
	userID common.UserID,
	categoryID common.CategoryID,
) (int64, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return 0, err
	}
	cid, err := mongohelper.ObjectIDFromHex(string(categoryID))
	if err != nil {
		return 0, err
	}

	count, err := m.col.CountDocuments(ctx, bson.M{
		"user_id":     uid,
		"category_id": cid,
		"deleted_at":  bson.M{"$exists": false},
	})
	if err != nil {
		m.logger.Error(
			"mongo count by category failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"category_id", categoryID,
			"err", err,
		)
		return 0, err
	}

	return count, nil
}

// ReassignCategory bulk-moves a user's non-deleted income entries from one
// category to another — used when archiving a category with reassignment.
func (m *MongoRepo) ReassignCategory(
	ctx context.Context,
	userID common.UserID,
	fromCategoryID, toCategoryID common.CategoryID,
) (int64, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return 0, err
	}
	fromID, err := mongohelper.ObjectIDFromHex(string(fromCategoryID))
	if err != nil {
		return 0, err
	}
	toID, err := mongohelper.ObjectIDFromHex(string(toCategoryID))
	if err != nil {
		return 0, err
	}

	res, err := m.col.UpdateMany(
		ctx,
		bson.M{
			"user_id":     uid,
			"category_id": fromID,
			"deleted_at":  bson.M{"$exists": false},
		},
		bson.M{"$set": bson.M{"category_id": toID, "updated_at": time.Now().UTC()}},
	)
	if err != nil {
		m.logger.Error(
			"mongo reassign category failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"err", err,
		)
		return 0, err
	}

	if res.ModifiedCount > 0 {
		_ = m.audit.Write(ctx, auditrepo.Entry{
			Action: "reassign_category",
			Entity: "income",
			Payload: bson.M{
				"user_id":          uid.Hex(),
				"from_category_id": string(fromCategoryID),
				"to_category_id":   string(toCategoryID),
				"count":            res.ModifiedCount,
			},
		})
	}

	return res.ModifiedCount, nil
}
