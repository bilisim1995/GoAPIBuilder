package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"legal-documents-api/config"
	"legal-documents-api/models"
	"legal-documents-api/utils"
)

// GetKurumDuyuru returns last 5 announcements for a given kurum_id
func GetKurumDuyuru(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Get query parameters
	kurumID := r.URL.Query().Get("kurum_id")
	if kurumID == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "kurum_id parameter is required")
		return
	}

	collection := config.GetKurumDuyuruCollection(mongoClient)

	// Build filter for kurum_id and active status
	filter := bson.M{
		"kurum_id": kurumID,
		"status":   "aktif",
	}

	// Setup find options - get last 5 records ordered by date descending
	findOptions := options.Find()
	findOptions.SetLimit(5)
	findOptions.SetSort(bson.M{"tarih": -1}) // Sort by date, newest first

	// Only select necessary fields
	findOptions.SetProjection(bson.M{
		"_id":      1,
		"kurum_id": 1,
		"baslik":   1,
		"link":     1,
		"tarih":    1,
	})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch announcements: "+err.Error())
		return
	}
	defer cursor.Close(ctx)

	var duyurular []models.KurumDuyuru
	if err := cursor.All(ctx, &duyurular); err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to decode announcements: "+err.Error())
		return
	}

	// Prepare response
	response := models.APIResponse{
		Success: true,
		Data:    duyurular,
		Count:   len(duyurular),
		Message: "Kurum duyuruları başarıyla getirildi",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}