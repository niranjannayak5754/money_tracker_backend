package sessionrepo

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/session"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	mongohelper "github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
)

type MongoRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
}

func New(db *mongo.Database, logger *slog.Logger) session.Repository {
	return &MongoRepo{
		col:    db.Collection("refresh_tokens"),
		logger: logger.With("repo", "session"),
	}
}

type mongoRefreshToken struct {
	ID        primitive.ObjectID `bson:"_id"`
	UserID    primitive.ObjectID `bson:"user_id"`
	TokenHash string             `bson:"token_hash"`
	ExpiresAt time.Time          `bson:"expires_at"`
	CreatedAt time.Time          `bson:"created_at"`
	RevokedAt *time.Time         `bson:"revoked_at,omitempty"`
}

func (m *MongoRepo) Create(ctx context.Context, rt session.RefreshToken) error {
	uid, err := mongohelper.ObjectIDFromHex(string(rt.UserID))
	if err != nil {
		return err
	}

	doc := mongoRefreshToken{
		ID:        primitive.NewObjectID(),
		UserID:    uid,
		TokenHash: rt.TokenHash,
		ExpiresAt: rt.ExpiresAt,
		CreatedAt: time.Now().UTC(),
	}

	_, err = m.col.InsertOne(ctx, doc)
	if err != nil {
		m.logger.Error(
			"mongo insert failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", rt.UserID,
			"err", err,
		)
	}
	return err
}

func (m *MongoRepo) FindValidByHash(ctx context.Context, tokenHash string) (*session.RefreshToken, error) {
	filter := bson.M{
		"token_hash": tokenHash,
		"revoked_at": bson.M{"$exists": false},
		"expires_at": bson.M{"$gt": time.Now().UTC()},
	}

	var doc mongoRefreshToken
	err := m.col.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, repository.ErrNotFound
		}
		m.logger.Error(
			"mongo find refresh token failed",
			"request_id", requestctx.RequestID(ctx),
			"err", err,
		)
		return nil, err
	}

	return &session.RefreshToken{
		ID:        common.SessionID(doc.ID.Hex()),
		UserID:    common.UserID(doc.UserID.Hex()),
		TokenHash: doc.TokenHash,
		ExpiresAt: doc.ExpiresAt,
		CreatedAt: doc.CreatedAt,
		RevokedAt: doc.RevokedAt,
	}, nil
}

func (m *MongoRepo) RevokeByHash(ctx context.Context, tokenHash string) error {
	_, err := m.col.UpdateOne(
		ctx,
		bson.M{"token_hash": tokenHash},
		bson.M{"$set": bson.M{"revoked_at": time.Now().UTC()}},
	)
	if err != nil {
		m.logger.Error(
			"mongo revoke refresh token failed",
			"request_id", requestctx.RequestID(ctx),
			"err", err,
		)
	}
	return err
}

func (m *MongoRepo) RevokeAllForUser(ctx context.Context, userID common.UserID) error {
	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return err
	}

	_, err = m.col.UpdateMany(
		ctx,
		bson.M{"user_id": uid, "revoked_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"revoked_at": time.Now().UTC()}},
	)
	if err != nil {
		m.logger.Error(
			"mongo revoke all refresh tokens failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"err", err,
		)
	}
	return err
}
