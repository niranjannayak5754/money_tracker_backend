package mongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ObjectIDFromHex(id string) (primitive.ObjectID, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid mongo object id: %w", err)
	}
	return oid, nil
}
