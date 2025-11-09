package user

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type mongoRepo struct {
	col *mongo.Collection
}

func NewMongoRepo(db *mongo.Database) Repository {
	return &mongoRepo{col: db.Collection("users")}
}

// Insert stores a new user document.
func (m *mongoRepo) Insert(rctx any, u Model) error {
	_, err := m.col.InsertOne(toCtx(rctx), u)
	return err
}

// FindByEmail fetches a user by email. Returns (nil, err) if not found.
func (m *mongoRepo) FindByEmail(rctx any, email string) (*Model, error) {
	var out Model
	err := m.col.FindOne(toCtx(rctx), bson.M{"email": email}).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// FindByID fetches a user by ID. Returns (nil, err) if not found.
func (m *mongoRepo) FindByID(rctx any, id primitive.ObjectID) (*Model, error) {
	var out Model
	err := m.col.FindOne(toCtx(rctx), bson.M{"_id": id}).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// toCtx normalizes rctx to context.Context.
func toCtx(rctx any) context.Context {
	if c, ok := rctx.(context.Context); ok {
		return c
	}
	return context.Background()
}
