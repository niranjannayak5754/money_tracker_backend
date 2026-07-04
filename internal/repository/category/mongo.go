package categoryrepo

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/category"
	"github.com/niranjannayak5754/money_tracker_backend/internal/domain/common"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	mongohelper "github.com/niranjannayak5754/money_tracker_backend/internal/platform/mongo"
	"github.com/niranjannayak5754/money_tracker_backend/internal/repository"
	auditrepo "github.com/niranjannayak5754/money_tracker_backend/internal/repository/audit"
)

type MongoRepo struct {
	col    *mongo.Collection
	logger *slog.Logger
	audit  *auditrepo.AuditRepo
}

func New(
	db *mongo.Database,
	logger *slog.Logger,
) category.Repository {
	return &MongoRepo{
		col:    db.Collection("categories"),
		logger: logger.With("repo", "category"),
		audit:  auditrepo.New(db, logger),
	}
}

type mongoCategory struct {
	ID        primitive.ObjectID `bson:"_id"`
	UserID    primitive.ObjectID `bson:"user_id"`
	Name      string             `bson:"name"`
	Type      string             `bson:"type"`
	Color     string             `bson:"color,omitempty"`
	Icon      string             `bson:"icon,omitempty"`
	Archived  bool               `bson:"archived"`
	CreatedAt time.Time          `bson:"created_at"`
}

// Create inserts a new category document.
func (m *MongoRepo) Create(
	ctx context.Context,
	cat category.Model,
) (common.CategoryID, error) {

	uid, err := mongohelper.ObjectIDFromHex(string(cat.UserID))
	if err != nil {
		return "", err
	}

	doc := mongoCategory{
		ID:        primitive.NewObjectID(),
		UserID:    uid,
		Name:      cat.Name,
		Type:      cat.Type,
		Color:     cat.Color,
		Icon:      cat.Icon,
		Archived:  cat.Archived,
		CreatedAt: time.Now().UTC(),
	}

	_, err = m.col.InsertOne(ctx, doc)
	if err != nil {
		m.logger.Error(
			"mongo insert failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", cat.UserID,
			"err", err,
		)
		return "", err
	}

	_ = m.audit.Write(ctx, auditrepo.Entry{
		Action:   "create",
		Entity:   "category",
		EntityID: doc.ID.Hex(),
		Payload: bson.M{
			"user_id": doc.UserID.Hex(),
			"name":    doc.Name,
			"type":    doc.Type,
			"color":   doc.Color,
			"icon":    doc.Icon,
		},
	})

	return common.CategoryID(doc.ID.Hex()), nil
}

// List returns all categories for a user.
func (m *MongoRepo) List(
	ctx context.Context,
	userID common.UserID,
	archived *bool,
) ([]category.Model, error) {

	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"user_id": uid,
	}

	if archived != nil {
		filter["archived"] = *archived
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cur, err := m.col.Find(ctx, filter, opts)
	if err != nil {
		m.logger.Error(
			"mongo find failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"err", err,
		)
		return nil, err
	}
	defer cur.Close(ctx)

	var out []category.Model
	for cur.Next(ctx) {
		var mc mongoCategory
		if err := cur.Decode(&mc); err != nil {
			m.logger.Error(
				"mongo decode failed",
				"request_id", requestctx.RequestID(ctx),
				"uid", userID,
				"err", err,
			)
			return nil, err
		}

		out = append(out, category.Model{
			ID:        common.CategoryID(mc.ID.Hex()),
			UserID:    common.UserID(mc.UserID.Hex()),
			Name:      mc.Name,
			Type:      mc.Type,
			Color:     mc.Color,
			Icon:      mc.Icon,
			Archived:  mc.Archived,
			CreatedAt: mc.CreatedAt,
		})
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

// GetByID fetches a single category by ID, scoped to the user — used by
// archive/reassign flows that need to know the category's own Type before
// deciding which entity (expense vs income) to check/reassign against.
func (m *MongoRepo) GetByID(
	ctx context.Context,
	userID common.UserID,
	categoryID common.CategoryID,
) (category.Model, error) {

	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return category.Model{}, err
	}
	cid, err := mongohelper.ObjectIDFromHex(string(categoryID))
	if err != nil {
		return category.Model{}, err
	}

	var mc mongoCategory
	err = m.col.FindOne(ctx, bson.M{"_id": cid, "user_id": uid}).Decode(&mc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return category.Model{}, repository.ErrNotFound
		}
		m.logger.Error(
			"mongo find one failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"category_id", categoryID,
			"err", err,
		)
		return category.Model{}, err
	}

	return category.Model{
		ID:        common.CategoryID(mc.ID.Hex()),
		UserID:    common.UserID(mc.UserID.Hex()),
		Name:      mc.Name,
		Type:      mc.Type,
		Color:     mc.Color,
		Icon:      mc.Icon,
		Archived:  mc.Archived,
		CreatedAt: mc.CreatedAt,
	}, nil
}

// Update modifies category fields.
func (m *MongoRepo) Update(
	ctx context.Context,
	userID common.UserID,
	id common.CategoryID,
	set map[string]any,
) (bool, error) {

	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}

	cid, err := mongohelper.ObjectIDFromHex(string(id))
	if err != nil {
		return false, err
	}

	res, err := m.col.UpdateOne(
		ctx,
		bson.M{
			"_id":     cid,
			"user_id": uid,
		},
		bson.M{"$set": set},
	)

	if err != nil {
		m.logger.Error(
			"mongo update failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"category_id", id,
			"err", err,
		)
		return false, err
	}

	if res.MatchedCount > 0 {
		_ = m.audit.Write(ctx, auditrepo.Entry{
			Action:   "update",
			Entity:   "category",
			EntityID: string(id),
			Payload:  bson.M{"set": set},
		})
	}

	return res.MatchedCount > 0, nil
}

// ExistsForUser checks if a category belongs to the user and is not archived.
func (m *MongoRepo) ExistsForUser(
	ctx context.Context,
	userID common.UserID,
	categoryID common.CategoryID,
) (bool, error) {

	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}

	cid, err := mongohelper.ObjectIDFromHex(string(categoryID))
	if err != nil {
		return false, err
	}

	filter := bson.M{
		"_id":      cid,
		"user_id":  uid,
		"archived": bson.M{"$ne": true},
	}

	count, err := m.col.CountDocuments(ctx, filter)
	if err != nil {
		m.logger.Error(
			"mongo count failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"category_id", categoryID,
			"err", err,
		)
		return false, err
	}

	return count > 0, nil
}

// ExistsForUserWithType is like ExistsForUser but also requires the
// category's Type to match.
func (m *MongoRepo) ExistsForUserWithType(
	ctx context.Context,
	userID common.UserID,
	categoryID common.CategoryID,
	categoryType string,
) (bool, error) {

	uid, err := mongohelper.ObjectIDFromHex(string(userID))
	if err != nil {
		return false, err
	}

	cid, err := mongohelper.ObjectIDFromHex(string(categoryID))
	if err != nil {
		return false, err
	}

	filter := bson.M{
		"_id":      cid,
		"user_id":  uid,
		"type":     categoryType,
		"archived": bson.M{"$ne": true},
	}

	count, err := m.col.CountDocuments(ctx, filter)
	if err != nil {
		m.logger.Error(
			"mongo count failed",
			"request_id", requestctx.RequestID(ctx),
			"uid", userID,
			"category_id", categoryID,
			"err", err,
		)
		return false, err
	}

	return count > 0, nil
}
