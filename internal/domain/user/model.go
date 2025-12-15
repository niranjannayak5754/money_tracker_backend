package user

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Model struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	Email     string             `bson:"email" json:"email"`
	PassHash  []byte             `bson:"pass_hash" json:"-"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
