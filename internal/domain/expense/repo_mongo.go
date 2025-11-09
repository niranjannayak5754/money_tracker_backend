package expense

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoRepo struct {
	col *mongo.Collection
}

func NewMongoRepo(db *mongo.Database) Repository {
	return &mongoRepo{
		col: db.Collection("expenses"),
	}
}

// Create inserts an expense record.
func (m *mongoRepo) Create(rctx any, x Model) error {
	_, err := m.col.InsertOne(toCtx(rctx), x)
	return err
}

// ListMonth returns a user’s expenses for a given month and category (optional).
func (m *mongoRepo) ListMonth(rctx any, uid primitive.ObjectID, start, end time.Time, category *primitive.ObjectID) ([]Model, error) {

	filter := bson.M{
		"user_id": uid,
		"date": bson.M{
			"$gte": start,
			"$lt":  end,
		},
	}

	if category != nil {
		filter["category_id"] = *category
	}

	opts := options.Find().SetSort(bson.D{{Key: "date", Value: -1}})

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

// Update modifies an existing expense.
func (m *mongoRepo) Update(rctx any, uid, id primitive.ObjectID, set map[string]any) (bool, error) {
	res, err := m.col.UpdateOne(
		toCtx(rctx),
		bson.M{"_id": id, "user_id": uid},
		bson.M{"$set": set},
	)
	if err != nil {
		return false, err
	}
	return res.MatchedCount > 0, nil
}

// Delete removes an expense.
func (m *mongoRepo) Delete(rctx any, uid, id primitive.ObjectID) (bool, error) {
	res, err := m.col.DeleteOne(toCtx(rctx), bson.M{"_id": id, "user_id": uid})
	if err != nil {
		return false, err
	}
	return res.DeletedCount > 0, nil
}

// Context helper
func toCtx(rctx any) context.Context {
	if c, ok := rctx.(context.Context); ok {
		return c
	}
	return context.Background()
}
