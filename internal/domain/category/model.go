package category

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Model struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Name      string             `bson:"name" json:"name"`
	Type      string             `bson:"type" json:"type"` // expense | income
	Archived  bool               `bson:"archived" json:"archived"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
