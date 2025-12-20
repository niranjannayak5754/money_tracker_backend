package userrepo

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/user"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
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
			"uid", u.ID.Hex(),
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
		if err == mongo.ErrNoDocuments {
			return nil, repository.ErrNotFound
		}

		m.logger.Error(
			"mongo find user by email failed",
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
		if err == mongo.ErrNoDocuments {
			return nil, repository.ErrNotFound
		}

		m.logger.Warn(
			"mongo find user by id failed",
			"uid", id.Hex(),
			"err", err,
		)
		return nil, err
	}

	return &out, nil
}

func (m *MongoRepo) UpdatePasswordHash(
	ctx context.Context,
	id primitive.ObjectID,
	hash []byte,
) error {
	res, err := m.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"pass_hash": hash}})
	if err != nil {
		m.logger.Error(
			"mongo update password_hash failed",
			"uid", id.Hex(),
			"err", err,
		)
		return err
	}

	if res.MatchedCount == 0 {
		return repository.ErrNotFound
	}

	return nil
}
