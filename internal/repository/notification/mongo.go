package notificationrepo

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/notification"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	mongohelper "github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
)

type MongoRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
}

func New(db *mongo.Database, logger *slog.Logger) notification.Repository {
	return &MongoRepo{
		col:    db.Collection("notifications"),
		logger: logger.With("repo", "notification"),
	}
}

type mongoNotification struct {
	ID                primitive.ObjectID `bson:"_id"`
	UserID            primitive.ObjectID `bson:"user_id"`
	Type              string             `bson:"type"`
	Title             string             `bson:"title"`
	Message           string             `bson:"message,omitempty"`
	RelatedEntityType string             `bson:"related_entity_type,omitempty"`
	RelatedEntityID   string             `bson:"related_entity_id,omitempty"`
	CreatedAt         time.Time          `bson:"created_at"`
	ReadAt            *time.Time         `bson:"read_at,omitempty"`
	DismissedAt       *time.Time         `bson:"dismissed_at,omitempty"`
}

func toModel(mn mongoNotification) notification.Model {
	return notification.Model{
		ID:                common.NotificationID(mn.ID.Hex()),
		UserID:            common.UserID(mn.UserID.Hex()),
		Type:              notification.Type(mn.Type),
		Title:             mn.Title,
		Message:           mn.Message,
		RelatedEntityType: mn.RelatedEntityType,
		RelatedEntityID:   mn.RelatedEntityID,
		CreatedAt:         mn.CreatedAt,
		ReadAt:            mn.ReadAt,
		DismissedAt:       mn.DismissedAt,
	}
}

func (m *MongoRepo) Create(ctx context.Context, n notification.Model) (common.NotificationID, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(n.UserID))
	if err != nil {
		return "", err
	}

	doc := mongoNotification{
		ID:                primitive.NewObjectID(),
		UserID:            uid,
		Type:              string(n.Type),
		Title:             n.Title,
		Message:           n.Message,
		RelatedEntityType: n.RelatedEntityType,
		RelatedEntityID:   n.RelatedEntityID,
		CreatedAt:         time.Now().UTC(),
	}

	_, err = m.col.InsertOne(ctx, doc)
	if err != nil {
		m.logger.Error("mongo insert failed", "request_id", requestctx.RequestID(ctx), "uid", n.UserID, "err", err)
		return "", err
	}

	return common.NotificationID(doc.ID.Hex()), nil
}

func (m *MongoRepo) List(ctx context.Context, userID common.UserID, onlyUnread bool) ([]notification.Model, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return nil, err
	}

	filter := bson.M{"user_id": uid, "dismissed_at": bson.M{"$exists": false}}
	if onlyUnread {
		filter["read_at"] = bson.M{"$exists": false}
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cur, err := m.col.Find(ctx, filter, opts)
	if err != nil {
		m.logger.Error("mongo find failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
		return nil, err
	}
	defer cur.Close(ctx)

	var out []notification.Model
	for cur.Next(ctx) {
		var mn mongoNotification
		if err := cur.Decode(&mn); err != nil {
			m.logger.Error("mongo decode failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
			return nil, err
		}
		out = append(out, toModel(mn))
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (m *MongoRepo) UnreadCount(ctx context.Context, userID common.UserID) (int64, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return 0, err
	}

	count, err := m.col.CountDocuments(ctx, bson.M{
		"user_id":      uid,
		"dismissed_at": bson.M{"$exists": false},
		"read_at":      bson.M{"$exists": false},
	})
	if err != nil {
		m.logger.Error("mongo count failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
		return 0, err
	}
	return count, nil
}

func (m *MongoRepo) MarkRead(ctx context.Context, userID common.UserID, id common.NotificationID) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}
	nid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{"_id": nid, "user_id": uid, "read_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"read_at": time.Now().UTC()}},
	)
	if err != nil {
		m.logger.Error("mongo mark read failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "notification_id", id, "err", err)
		return false, err
	}
	if res.MatchedCount > 0 {
		return true, nil
	}

	// Already read (or doesn't exist) — distinguish so the handler can
	// still report success for an already-read notification rather than
	// a confusing 404.
	count, err := m.col.CountDocuments(ctx, bson.M{"_id": nid, "user_id": uid})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *MongoRepo) Dismiss(ctx context.Context, userID common.UserID, id common.NotificationID) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}
	nid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{"_id": nid, "user_id": uid, "dismissed_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"dismissed_at": time.Now().UTC()}},
	)
	if err != nil {
		m.logger.Error("mongo dismiss failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "notification_id", id, "err", err)
		return false, err
	}
	if res.MatchedCount > 0 {
		return true, nil
	}

	count, err := m.col.CountDocuments(ctx, bson.M{"_id": nid, "user_id": uid})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *MongoRepo) ExistsActive(ctx context.Context, userID common.UserID, notifType notification.Type, relatedEntityType, relatedEntityID string) (bool, error) {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}

	count, err := m.col.CountDocuments(ctx, bson.M{
		"user_id":             uid,
		"type":                string(notifType),
		"related_entity_type": relatedEntityType,
		"related_entity_id":   relatedEntityID,
		"dismissed_at":        bson.M{"$exists": false},
	})
	if err != nil {
		m.logger.Error("mongo exists active failed", "request_id", requestctx.RequestID(ctx), "uid", userID, "err", err)
		return false, err
	}
	return count > 0, nil
}
