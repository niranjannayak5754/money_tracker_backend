package goalrepo

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
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/goal"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	mongohelper "github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
)

type MongoRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
}

func New(db *mongo.Database, logger *slog.Logger) goal.Repository {
	return &MongoRepo{
		col:    db.Collection("goals"),
		logger: logger.With("repo", "goal"),
	}
}

type mongoGoal struct {
	ID           primitive.ObjectID   `bson:"_id"`
	UserID       primitive.ObjectID   `bson:"user_id"`
	Name         string               `bson:"name"`
	TargetAmount primitive.Decimal128 `bson:"target_amount"`
	TargetDate   *time.Time           `bson:"target_date,omitempty"`
	LinkedType   string               `bson:"linked_type"`
	LinkedID     string               `bson:"linked_id"`
	Status       string               `bson:"status"`
	CreatedAt    time.Time            `bson:"created_at"`
	UpdatedAt    time.Time            `bson:"updated_at"`
	DeletedAt    *time.Time           `bson:"deleted_at,omitempty"`
	DeletedBy    *primitive.ObjectID  `bson:"deleted_by,omitempty"`
}

func (m *MongoRepo) Create(ctx context.Context, g goal.Model) (common.GoalID, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(g.UserID))
	if err != nil {
		return "", err
	}

	dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", g.TargetAmount))
	if err != nil {
		return "", err
	}

	doc := mongoGoal{
		ID:           primitive.NewObjectID(),
		UserID:       uid,
		Name:         g.Name,
		TargetAmount: dec,
		TargetDate:   g.TargetDate,
		LinkedType:   string(g.LinkedType),
		LinkedID:     g.LinkedID,
		Status:       string(g.Status),
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	_, err = m.col.InsertOne(ctx, doc)
	if err != nil {
		m.logger.Error("mongo insert failed", "request_id", requestctx.RequestID(ctx), "uid", g.UserID, "err", err)
		return "", err
	}

	return common.GoalID(doc.ID.Hex()), nil
}

func (m *MongoRepo) List(ctx context.Context, userID common.UserID) ([]goal.Model, error) {
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

	var out []goal.Model
	for cur.Next(ctx) {
		var mg mongoGoal
		if err := cur.Decode(&mg); err != nil {
			m.logger.Error("mongo decode failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
			return nil, err
		}
		out = append(out, goal.Model{
			ID:           common.GoalID(mg.ID.Hex()),
			UserID:       common.UserID(mg.UserID.Hex()),
			Name:         mg.Name,
			TargetAmount: shared.Decimal128ToFloat(mg.TargetAmount),
			TargetDate:   mg.TargetDate,
			LinkedType:   goal.LinkedType(mg.LinkedType),
			LinkedID:     mg.LinkedID,
			Status:       goal.Status(mg.Status),
			CreatedAt:    mg.CreatedAt,
			UpdatedAt:    mg.UpdatedAt,
		})
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (m *MongoRepo) Update(ctx context.Context, userID common.UserID, id common.GoalID, set map[string]any) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}
	gid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	converted := bson.M{}
	for k, v := range set {
		if k == "target_amount" {
			f, ok := v.(float64)
			if !ok {
				return false, fmt.Errorf("target_amount must be a float64")
			}
			dec, err := primitive.ParseDecimal128(fmt.Sprintf("%.2f", f))
			if err != nil {
				return false, err
			}
			converted["target_amount"] = dec
			continue
		}
		converted[k] = v
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{"_id": gid, "user_id": uid, "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": converted},
	)
	if err != nil {
		m.logger.Error("mongo update failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "goal_id", id, "err", err)
		return false, err
	}

	return res.MatchedCount > 0, nil
}

func (m *MongoRepo) Delete(ctx context.Context, userID common.UserID, id common.GoalID) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}
	gid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{"_id": gid, "user_id": uid, "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"deleted_at": time.Now().UTC(), "deleted_by": uid}},
	)
	if err != nil {
		m.logger.Error("mongo soft-delete failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "goal_id", id, "err", err)
		return false, err
	}

	return res.MatchedCount > 0, nil
}
