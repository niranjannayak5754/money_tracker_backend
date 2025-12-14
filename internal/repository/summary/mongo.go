package summaryrepo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/shared"
)

type MongoRepo struct {
	incomeCol   *mongo.Collection
	expensesCol *mongo.Collection
}

func New(db *mongo.Database) *MongoRepo {
	return &MongoRepo{
		incomeCol:   db.Collection("income"),
		expensesCol: db.Collection("expenses"),
	}
}

func (r *MongoRepo) IncomeTotal(ctx context.Context, uid primitive.ObjectID, start, end time.Time) (float64, error) {
	cur, err := r.incomeCol.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: bson.M{
			"user_id": uid,
			"date":    bson.M{"$gte": start, "$lt": end},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$amount"},
		}}},
	})
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	var out struct {
		Total primitive.Decimal128 `bson:"total"`
	}

	if cur.Next(ctx) {
		if err := cur.Decode(&out); err != nil {
			return 0, err
		}
		return shared.Decimal128ToFloat(out.Total), nil
	}

	return 0, nil
}

func (r *MongoRepo) ExpenseTotals(ctx context.Context, uid primitive.ObjectID, start, end time.Time) (float64, []map[string]any, error) {
	cur, err := r.expensesCol.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: bson.M{
			"user_id": uid,
			"date":    bson.M{"$gte": start, "$lt": end},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   "$category_id",
			"total": bson.M{"$sum": "$amount"},
		}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "categories",
			"localField":   "_id",
			"foreignField": "_id",
			"as":           "cat",
		}}},
		bson.D{{Key: "$unwind", Value: bson.M{
			"path":                       "$cat",
			"preserveNullAndEmptyArrays": true,
		}}},
	})
	if err != nil {
		return 0, nil, err
	}
	defer cur.Close(ctx)

	var total float64
	var cats []map[string]any

	for cur.Next(ctx) {
		var x struct {
			CatID primitive.ObjectID   `bson:"_id"`
			Total primitive.Decimal128 `bson:"total"`
			Cat   struct {
				Name string `bson:"name"`
			} `bson:"cat"`
		}

		if err := cur.Decode(&x); err != nil {
			return 0, nil, err
		}

		f := shared.Decimal128ToFloat(x.Total)
		total += f

		cats = append(cats, map[string]any{
			"category_id":   x.CatID.Hex(),
			"category_name": x.Cat.Name,
			"total":         f,
		})
	}

	return total, cats, nil
}
