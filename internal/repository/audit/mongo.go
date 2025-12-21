package auditrepo

import (
	"context"
	"time"

	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
)

type AuditRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
}

type Entry struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	RequestID string             `bson:"request_id,omitempty"`
	Actor     string             `bson:"actor,omitempty"`
	Action    string             `bson:"action"`
	Entity    string             `bson:"entity"`
	EntityID  string             `bson:"entity_id,omitempty"`
	Payload   bson.M             `bson:"payload,omitempty"`
	CreatedAt time.Time          `bson:"created_at"`
}

func New(db *mongo.Database, logger *slog.Logger) *AuditRepo {
	return &AuditRepo{
		col:    db.Collection("audit_logs"),
		logger: logger.With("repo", "audit"),
	}
}

// Write inserts an audit entry. Best-effort: logs error but returns it to caller.
func (a *AuditRepo) Write(ctx context.Context, e Entry) error {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	if e.ID.IsZero() {
		e.ID = primitive.NewObjectID()
	}
	if e.RequestID == "" {
		e.RequestID = requestctx.RequestID(ctx)
	}
	if e.Actor == "" {
		e.Actor = requestctx.UID(ctx)
	}

	_, err := a.col.InsertOne(ctx, e)
	if err != nil {
		a.logger.Error("failed to write audit entry", "request_id", e.RequestID, "actor", e.Actor, "err", err)
	}
	return err
}
