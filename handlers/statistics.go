package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"legal-documents-api/config"
	"legal-documents-api/models"
	"legal-documents-api/utils"
)

// StatisticsResponse represents the statistics data structure
type StatisticsResponse struct {
	TotalKurumlar      int64                    `json:"total_kurumlar"`
	TotalBelgeler      int64                    `json:"total_belgeler"`
	BelgeTuruIstatistik []BelgeTuruCount        `json:"belge_turu_istatistik"`
}

// BelgeTuruCount represents count by document type
type BelgeTuruCount struct {
	BelgeTuru string `json:"belge_turu"`
	Count     int64  `json:"count"`
}

// GetStatistics returns statistics about institutions and documents
func GetStatistics(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Get total kurumlar count
	allKurumlar := utils.GetAllKurumlar()
	if len(allKurumlar) == 0 {
		// If cache is empty, try to refresh it
		if err := utils.RefreshKurumlarCache(mongoClient); err != nil {
			utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to load institutions: "+err.Error())
			return
		}
		allKurumlar = utils.GetAllKurumlar()
	}
	totalKurumlar := int64(len(allKurumlar))

	// 2. Get total documents count from metadata collection
	metadataCollection := config.GetMetadataCollection(mongoClient)
	totalBelgeler, err := metadataCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to count documents: "+err.Error())
		return
	}

	// 3. Get document counts grouped by belge_turu
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":   "$belge_turu",
				"count": bson.M{"$sum": 1},
			},
		},
		{
			"$sort": bson.M{"count": -1}, // Sort by count descending
		},
	}

	cursor, err := metadataCollection.Aggregate(ctx, pipeline)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to aggregate document types: "+err.Error())
		return
	}
	defer cursor.Close(ctx)

	// Parse belge_turu statistics
	var belgeTuruIstatistik []BelgeTuruCount
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}

		// Handle empty belge_turu
		belgeTuru := result.ID
		if belgeTuru == "" {
			belgeTuru = "Belirtilmemiş"
		}

		belgeTuruIstatistik = append(belgeTuruIstatistik, BelgeTuruCount{
			BelgeTuru: belgeTuru,
			Count:     result.Count,
		})
	}

	// Build response
	statistics := StatisticsResponse{
		TotalKurumlar:      totalKurumlar,
		TotalBelgeler:      totalBelgeler,
		BelgeTuruIstatistik: belgeTuruIstatistik,
	}

	response := models.APIResponse{
		Success: true,
		Data:    statistics,
		Message: "Statistics fetched successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

