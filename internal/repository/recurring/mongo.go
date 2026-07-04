package recurringrepo

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/recurring"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	mongohelper "github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
)

type MongoRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
}

func New(db *mongo.Database, logger *slog.Logger) recurring.Repository {
	return &MongoRepo{
		col:    db.Collection("recurring_templates"),
		logger: logger.With("repo", "recurring"),
	}
}

type mongoRecurring struct {
	ID          primitive.ObjectID  `bson:"_id"`
	UserID      primitive.ObjectID  `bson:"user_id"`
	EntityType  string              `bson:"entity_type"`
	Frequency   string              `bson:"frequency"`
	DayOfMonth  int                 `bson:"day_of_month"`
	Payload     bson.M              `bson:"payload"`
	StartDate   time.Time           `bson:"start_date"`
	EndDate     *time.Time          `bson:"end_date,omitempty"`
	NextRunDate time.Time           `bson:"next_run_date"`
	LastRunDate *time.Time          `bson:"last_run_date,omitempty"`
	Active      bool                `bson:"active"`
	CreatedAt   time.Time           `bson:"created_at"`
	UpdatedAt   time.Time           `bson:"updated_at"`
	DeletedAt   *time.Time          `bson:"deleted_at,omitempty"`
	DeletedBy   *primitive.ObjectID `bson:"deleted_by,omitempty"`
}

func toModel(mr mongoRecurring) recurring.Model {
	return recurring.Model{
		ID:          common.RecurringID(mr.ID.Hex()),
		UserID:      common.UserID(mr.UserID.Hex()),
		EntityType:  recurring.EntityType(mr.EntityType),
		Frequency:   recurring.Frequency(mr.Frequency),
		DayOfMonth:  mr.DayOfMonth,
		Payload:     map[string]any(mr.Payload),
		StartDate:   mr.StartDate,
		EndDate:     mr.EndDate,
		NextRunDate: mr.NextRunDate,
		LastRunDate: mr.LastRunDate,
		Active:      mr.Active,
		CreatedAt:   mr.CreatedAt,
		UpdatedAt:   mr.UpdatedAt,
	}
}

func (m *MongoRepo) Create(ctx context.Context, rec recurring.Model) (common.RecurringID, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(rec.UserID))
	if err != nil {
		return "", err
	}

	doc := mongoRecurring{
		ID:          primitive.NewObjectID(),
		UserID:      uid,
		EntityType:  string(rec.EntityType),
		Frequency:   string(rec.Frequency),
		DayOfMonth:  rec.DayOfMonth,
		Payload:     bson.M(rec.Payload),
		StartDate:   rec.StartDate,
		EndDate:     rec.EndDate,
		NextRunDate: rec.NextRunDate,
		Active:      rec.Active,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	_, err = m.col.InsertOne(ctx, doc)
	if err != nil {
		m.logger.Error("mongo insert failed", "request_id", requestctx.RequestID(ctx), "uid", rec.UserID, "err", err)
		return "", err
	}

	return common.RecurringID(doc.ID.Hex()), nil
}

func (m *MongoRepo) List(ctx context.Context, userID common.UserID) ([]recurring.Model, error) {
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

	var out []recurring.Model
	for cur.Next(ctx) {
		var mr mongoRecurring
		if err := cur.Decode(&mr); err != nil {
			m.logger.Error("mongo decode failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
			return nil, err
		}
		out = append(out, toModel(mr))
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (m *MongoRepo) GetByID(ctx context.Context, userID common.UserID, id common.RecurringID) (*recurring.Model, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return nil, err
	}
	rid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return nil, err
	}

	var mr mongoRecurring
	err = m.col.FindOne(ctx, bson.M{"_id": rid, "user_id": uid, "deleted_at": bson.M{"$exists": false}}).Decode(&mr)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, repository.ErrNotFound
		}
		m.logger.Error("mongo find by id failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "recurring_id", id, "err", err)
		return nil, err
	}

	model := toModel(mr)
	return &model, nil
}

func (m *MongoRepo) Update(ctx context.Context, userID common.UserID, id common.RecurringID, set map[string]any) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}
	rid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{"_id": rid, "user_id": uid, "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": set},
	)
	if err != nil {
		m.logger.Error("mongo update failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "recurring_id", id, "err", err)
		return false, err
	}

	return res.MatchedCount > 0, nil
}

func (m *MongoRepo) Delete(ctx context.Context, userID common.UserID, id common.RecurringID) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}
	rid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{"_id": rid, "user_id": uid, "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"deleted_at": time.Now().UTC(), "deleted_by": uid, "active": false}},
	)
	if err != nil {
		m.logger.Error("mongo soft-delete failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "recurring_id", id, "err", err)
		return false, err
	}

	return res.MatchedCount > 0, nil
}

// ListDue returns active, non-deleted templates across all users whose
// NextRunDate is <= asOf. Not scoped to a single user — this is the
// background scheduler's global query.
func (m *MongoRepo) ListDue(ctx context.Context, asOf time.Time) ([]recurring.Model, error) {
	filter := bson.M{
		"active":        true,
		"deleted_at":    bson.M{"$exists": false},
		"next_run_date": bson.M{"$lte": asOf},
	}

	cur, err := m.col.Find(ctx, filter)
	if err != nil {
		m.logger.Error("mongo find due failed", "request_id", requestctx.RequestID(ctx), "err", err)
		return nil, err
	}
	defer cur.Close(ctx)

	var out []recurring.Model
	for cur.Next(ctx) {
		var mr mongoRecurring
		if err := cur.Decode(&mr); err != nil {
			m.logger.Error("mongo decode due failed", "request_id", requestctx.RequestID(ctx), "err", err)
			return nil, err
		}
		out = append(out, toModel(mr))
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
