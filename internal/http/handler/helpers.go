package handler

import (
	"net/http"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/niranjannayak5754/money_tracker_backend/internal/http/requestctx"
	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

func mustUID(w http.ResponseWriter, r *http.Request) (primitive.ObjectID, bool) {
	uidHex := requestctx.UID(r.Context())
	if uidHex == "" {
		response.Unauthorized(w, "missing uid")
		return primitive.NilObjectID, false
	}

	uid, err := primitive.ObjectIDFromHex(uidHex)
	if err != nil {
		response.Unauthorized(w, "invalid uid")
		return primitive.NilObjectID, false
	}

	return uid, true
}
