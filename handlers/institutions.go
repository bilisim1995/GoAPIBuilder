package handlers

import (
        "context"
        "encoding/json"
        "net/http"
        "sort"
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

// GetInstitutions returns a list of unique institutions from kurumlar collection with document counts
func GetInstitutions(w http.ResponseWriter, r *http.Request) {
        // Handle CORS preflight
        if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        // Get all kurumlar from cache
        allKurumlar := utils.GetAllKurumlar()
        if len(allKurumlar) == 0 {
                // If cache is empty, try to refresh it
                if err := utils.RefreshKurumlarCache(mongoClient); err != nil {
                        utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to load institutions: "+err.Error())
                        return
                }
                allKurumlar = utils.GetAllKurumlar()
        }

        // Get document counts for each institution from metadata collection
        metadataCollection := config.GetMetadataCollection(mongoClient)
        pipeline := []bson.M{
                {
                        "$match": bson.M{
                                "status": "aktif", // Only active documents
                        },
                },
                {
                        "$group": bson.M{
                                "_id":   "$kurum_id",
                                "count": bson.M{"$sum": 1},
                        },
                },
        }

        cursor, err := metadataCollection.Aggregate(ctx, pipeline)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to count documents: "+err.Error())
                return
        }
        defer cursor.Close(ctx)

        // Create map of kurum_id -> document count
        countMap := make(map[string]int32)
        var countResult struct {
                ID    string `bson:"_id"`
                Count int32  `bson:"count"`
        }
        
        for cursor.Next(ctx) {
                if err := cursor.Decode(&countResult); err != nil {
                        continue
                }
                countMap[countResult.ID] = countResult.Count
        }

        // Build institutions response with kurum data and document counts
        var institutions []models.Institution
        for _, kurum := range allKurumlar {
                // Show all institutions from kurumlar table, even if no documents
                count, exists := countMap[kurum.ID.Hex()]
                if !exists {
                        count = 0 // Set count to 0 if no documents found
                }

                institution := models.Institution{
                        KurumID:   kurum.ID.Hex(),
                        KurumAdi:  kurum.KurumAdi,
                        KurumLogo: kurum.KurumLogo,
                        Count:     count,
                }
                institutions = append(institutions, institution)
        }

        // Sort alphabetically by kurum_adi
        sort.Slice(institutions, func(i, j int) bool {
                return institutions[i].KurumAdi < institutions[j].KurumAdi
        })

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
