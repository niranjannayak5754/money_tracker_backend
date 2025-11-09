package category

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoRepo struct {
	col *mongo.Collection
}

func NewMongoRepo(db *mongo.Database) Repository {
	return &mongoRepo{col: db.Collection("categories")}
}

// Create inserts a new category into MongoDB.
func (m *mongoRepo) Create(rctx any, cat Model) error {
	_, err := m.col.InsertOne(toCtx(rctx), cat)
	return err
}

// ListActive returns all categories for a given user that are not archived.
func (m *mongoRepo) ListActive(rctx any, uid primitive.ObjectID) ([]Model, error) {
	filter := bson.M{
		"user_id": uid,
		"archived": bson.M{
			"$ne": true,
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cur, err := m.col.Find(toCtx(rctx), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(toCtx(rctx))

	var out []Model
	for cur.Next(toCtx(rctx)) {
		var v Model
		if err := cur.Decode(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}

	return out, cur.Err()
}

// Update modifies name or archived flags for a user’s category.
func (m *mongoRepo) Update(rctx any, uid, id primitive.ObjectID, set map[string]any) (bool, error) {
	filter := bson.M{"_id": id, "user_id": uid}
	update := bson.M{"$set": set}

	res, err := m.col.UpdateOne(toCtx(rctx), filter, update)
	if err != nil {
		return false, err
	}

	return res.MatchedCount > 0, nil
}

// ExistsForUser returns true if a category exists for the given user.
func (m *mongoRepo) ExistsForUser(rctx any, uid, categoryID primitive.ObjectID) (bool, error) {
	filter := bson.M{
		"_id":     categoryID,
		"user_id": uid,
		"archived": bson.M{
			"$ne": true,
		},
	}

	count, err := m.col.CountDocuments(toCtx(rctx), filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// toCtx normalizes rctx to context.Context.
func toCtx(rctx any) context.Context {
	if c, ok := rctx.(context.Context); ok {
		return c
	}
	return context.Background()
}
