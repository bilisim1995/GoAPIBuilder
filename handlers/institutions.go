package handlers

import (
        "context"
        "encoding/json"
        "net/http"
        "time"

        "go.mongodb.org/mongo-driver/bson"
        "go.mongodb.org/mongo-driver/mongo"

        "legal-documents-api/config"
        "legal-documents-api/models"
        "legal-documents-api/utils"
)

var mongoClient *mongo.Client

// InitHandlers initializes handlers with MongoDB client
func InitHandlers(client *mongo.Client) {
        mongoClient = client
}

// GetInstitutions returns a list of unique institutions from the metadata collection
func GetInstitutions(w http.ResponseWriter, r *http.Request) {
        // Handle CORS preflight
        if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        collection := config.GetMetadataCollection(mongoClient)

        // Aggregation pipeline to get unique institutions with document counts
        pipeline := []bson.M{
                {
                        "$match": bson.M{
                                "status": "aktif", // Only active documents
                        },
                },
                {
                        "$group": bson.M{
                                "_id":   "$kurum_adi",
                                "count": bson.M{"$sum": 1},
                        },
                },
                {
                        "$sort": bson.M{"_id": 1}, // Sort alphabetically
                },
        }

        cursor, err := collection.Aggregate(ctx, pipeline)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch institutions: "+err.Error())
                return
        }
        defer cursor.Close(ctx)

        var institutions []models.Institution
        if err := cursor.All(ctx, &institutions); err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to decode institutions: "+err.Error())
                return
        }

        // Send successful response
        response := models.APIResponse{
                Success: true,
                Data:    institutions,
                Count:   len(institutions),
                Message: "Institutions fetched successfully",
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}
