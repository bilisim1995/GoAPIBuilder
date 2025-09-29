package handlers

import (
        "context"
        "encoding/json"
        "net/http"
        "strconv"
        "time"

        "go.mongodb.org/mongo-driver/bson"
        "go.mongodb.org/mongo-driver/bson/primitive"

        "legal-documents-api/config"
        "legal-documents-api/models"
        "legal-documents-api/utils"
)

// GetRecentRegulations returns the most recently published regulations
func GetRecentRegulations(w http.ResponseWriter, r *http.Request) {
        // Handle CORS preflight
        if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        // Parse query parameters
        limit := 50 // Default limit
        if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
                if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
                        limit = parsedLimit
                }
        }

        sortBy := "belge_yayin_tarihi" // Default sort field
        if sortParam := r.URL.Query().Get("sort_by"); sortParam != "" {
                // Allow sorting by specific fields only
                allowedFields := map[string]bool{
                        "belge_yayin_tarihi":  true,
                        "olusturulma_tarihi": true,
                        "yukleme_tarihi":     true,
                        "pdf_adi":            true,
                }
                if allowedFields[sortParam] {
                        sortBy = sortParam
                }
        }

        sortOrder := -1 // Default: descending (newest first)
        if orderParam := r.URL.Query().Get("sort_order"); orderParam == "asc" {
                sortOrder = 1
        }

        // Get metadata collection
        metadataCollection := config.GetMetadataCollection(mongoClient)

        // Build aggregation pipeline
        pipeline := []bson.M{
                // Match only active documents
                {
                        "$match": bson.M{
                                "status": "aktif",
                        },
                },
                // Sort by the specified field
                {
                        "$sort": bson.M{
                                sortBy: sortOrder,
                        },
                },
                // Limit results
                {
                        "$limit": limit,
                },
                // Lookup institution information
                {
                        "$lookup": bson.M{
                                "from": "kurumlar",
                                "let":  bson.M{"kurumIdStr": "$kurum_id"},
                                "pipeline": []bson.M{
                                        {
                                                "$match": bson.M{
                                                        "$expr": bson.M{
                                                                "$eq": []interface{}{
                                                                        bson.M{"$toString": "$_id"},
                                                                        "$$kurumIdStr",
                                                                },
                                                        },
                                                },
                                        },
                                },
                                "as": "kurum_info",
                        },
                },
                // Unwind kurum_info (convert array to object)
                {
                        "$unwind": bson.M{
                                "path":                       "$kurum_info",
                                "preserveNullAndEmptyArrays": true,
                        },
                },
                // Project fields for response
                {
                        "$project": bson.M{
                                "_id":                  1,
                                "pdf_adi":              1,
                                "kurum_id":             1,
                                "belge_turu":           1,
                                "belge_durumu":         1,
                                "belge_yayin_tarihi":   1,
                                "etiketler":            1,
                                "aciklama":             1,
                                "url_slug":             1,
                                "sayfa_sayisi":         1,
                                "dosya_boyutu_mb":      1,
                                "yukleme_tarihi":       1,
                                "olusturulma_tarihi":   1,
                                "pdf_url":              1,
                                "kurum_adi":            "$kurum_info.kurum_adi",
                                "kurum_logo":           "$kurum_info.kurum_logo",
                                "kurum_aciklama":       "$kurum_info.aciklama",
                        },
                },
        }

        // Execute aggregation
        cursor, err := metadataCollection.Aggregate(ctx, pipeline)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch recent regulations: "+err.Error())
                return
        }
        defer cursor.Close(ctx)

        // Parse results
        var regulations []bson.M
        if err := cursor.All(ctx, &regulations); err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to decode regulations: "+err.Error())
                return
        }

        // Transform results to proper format
        var formattedRegulations []map[string]interface{}
        for _, reg := range regulations {
                formattedReg := make(map[string]interface{})
                
                // Handle ObjectID conversion
                if id, ok := reg["_id"].(primitive.ObjectID); ok {
                        formattedReg["id"] = id.Hex()
                }
                
                // Copy other fields
                formattedReg["pdf_adi"] = reg["pdf_adi"]
                formattedReg["kurum_id"] = reg["kurum_id"]
                formattedReg["kurum_adi"] = reg["kurum_adi"]
                formattedReg["kurum_logo"] = reg["kurum_logo"]
                formattedReg["kurum_aciklama"] = reg["kurum_aciklama"]
                formattedReg["belge_turu"] = reg["belge_turu"]
                formattedReg["belge_durumu"] = reg["belge_durumu"]
                formattedReg["belge_yayin_tarihi"] = reg["belge_yayin_tarihi"]
                formattedReg["etiketler"] = reg["etiketler"]
                formattedReg["aciklama"] = reg["aciklama"]
                formattedReg["url_slug"] = reg["url_slug"]
                formattedReg["sayfa_sayisi"] = reg["sayfa_sayisi"]
                formattedReg["dosya_boyutu_mb"] = reg["dosya_boyutu_mb"]
                formattedReg["yukleme_tarihi"] = reg["yukleme_tarihi"]
                formattedReg["olusturulma_tarihi"] = reg["olusturulma_tarihi"]
                formattedReg["pdf_url"] = reg["pdf_url"]
                
                formattedRegulations = append(formattedRegulations, formattedReg)
        }

        // Prepare response
        response := models.APIResponse{
                Success: true,
                Data:    formattedRegulations,
                Count:   len(formattedRegulations),
                Message: "Son mevzuatlar başarıyla çekildi",
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}