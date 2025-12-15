package userrepo

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/user"
)

type MongoRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
}

func New(
	db *mongo.Database,
	logger *slog.Logger,
) user.Repository {
	return &MongoRepo{
		col:    db.Collection("users"),
		logger: logger.With("repo", "user"),
	}
}

func (m *MongoRepo) Create(
	ctx context.Context,
	u user.Model,
) error {

	_, err := m.col.InsertOne(ctx, u)
	if err != nil {
		m.logger.Error(
			"insert user failed",
			"user_id", u.ID.Hex(),
			"email", u.Email,
			"err", err,
		)
		return err
	}

	return nil
}

func (m *MongoRepo) FindByEmail(
	ctx context.Context,
	email string,
) (*user.Model, error) {

	var out user.Model
	err := m.col.FindOne(ctx, bson.M{"email": email}).Decode(&out)
	if err != nil {
		m.logger.Warn(
			"user not found by email",
			"email", email,
			"err", err,
		)
		return nil, err
	}

	return &out, nil
}

func (m *MongoRepo) FindByID(
	ctx context.Context,
	id primitive.ObjectID,
) (*user.Model, error) {

	var out user.Model
	err := m.col.FindOne(ctx, bson.M{"_id": id}).Decode(&out)
	if err != nil {
		m.logger.Warn(
			"user not found by id",
			"user_id", id.Hex(),
			"err", err,
		)
		return nil, err
	}

	return &out, nil
}
