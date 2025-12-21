package userrepo

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/user"
	mongohelper "github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
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

	// map domain model to mongo document
	doc := struct {
		ID        primitive.ObjectID  `bson:"_id"`
		Email     string              `bson:"email"`
		PassHash  []byte              `bson:"pass_hash"`
		CreatedAt primitive.Timestamp `bson:"created_at"` // placeholder, will be ignored
	}{}

	// create a new ObjectID for the user document
	doc.ID = primitive.NewObjectID()
	doc.Email = u.Email
	doc.PassHash = u.PassHash

	_, err := m.col.InsertOne(ctx, bson.M{
		"_id":        doc.ID,
		"email":      doc.Email,
		"pass_hash":  doc.PassHash,
		"created_at": u.CreatedAt,
	})
	if err != nil {
		m.logger.Error(
			"insert user failed",
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
	// read mongo document and map to domain model
	var doc struct {
		ID        primitive.ObjectID `bson:"_id"`
		Email     string             `bson:"email"`
		PassHash  []byte             `bson:"pass_hash"`
		CreatedAt time.Time          `bson:"created_at"`
	}

	err := m.col.FindOne(ctx, bson.M{"email": email}).Decode(&doc)
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

	out = user.Model{
		ID:        common.UserID(doc.ID.Hex()),
		Email:     doc.Email,
		PassHash:  doc.PassHash,
		CreatedAt: doc.CreatedAt,
	}

	return &out, nil
}

func (m *MongoRepo) FindByID(
	ctx context.Context,
	id common.UserID,
) (*user.Model, error) {

	oid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return nil, err
	}

	var doc struct {
		ID        primitive.ObjectID `bson:"_id"`
		Email     string             `bson:"email"`
		PassHash  []byte             `bson:"pass_hash"`
		CreatedAt time.Time          `bson:"created_at"`
	}

	err = m.col.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, repository.ErrNotFound
		}

		m.logger.Warn(
			"mongo find user by id failed",
			"uid", id,
			"err", err,
		)
		return nil, err
	}

	out := user.Model{
		ID:        common.UserID(doc.ID.Hex()),
		Email:     doc.Email,
		PassHash:  doc.PassHash,
		CreatedAt: doc.CreatedAt,
	}

	return &out, nil
}

func (m *MongoRepo) UpdatePasswordHash(
	ctx context.Context,
	id common.UserID,
	hash []byte,
) error {
	oid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return err
	}

	res, err := m.col.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": bson.M{"pass_hash": hash}})
	if err != nil {
		m.logger.Error(
			"mongo update password_hash failed",
			"uid", id,
			"err", err,
		)
		return err
	}

	if res.MatchedCount == 0 {
		return repository.ErrNotFound
	}

	return nil
}
