package handlers

import (
        "context"
        "encoding/json"
        "net/http"
        "strings"
        "time"

        "go.mongodb.org/mongo-driver/bson"
        "go.mongodb.org/mongo-driver/mongo/options"

        "legal-documents-api/config"
        "legal-documents-api/models"
        "legal-documents-api/utils"
)

// SitemapInstitution represents institution data for sitemap
type SitemapInstitution struct {
        KurumAdi string `json:"kurum_adi" bson:"_id"`
        Count    int32  `json:"count" bson:"count"`
        Slug     string `json:"slug"`
}

// SitemapDocument represents document data for sitemap
type SitemapDocument struct {
        URLSlug              string `json:"url_slug" bson:"url_slug"`
        PdfAdi               string `json:"pdf_adi" bson:"pdf_adi"`
        KurumAdi             string `json:"kurum_adi" bson:"kurum_adi"`
        BelgeYayinTarihi     string `json:"belge_yayin_tarihi" bson:"belge_yayin_tarihi"`
        OlusturulmaTarihi    string `json:"olusturulma_tarihi" bson:"olusturulma_tarihi"`
}

// GetSitemapInstitutions returns all institutions for sitemap
func GetSitemapInstitutions(w http.ResponseWriter, r *http.Request) {
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
                                "status": "aktif",
                        },
                },
                {
                        "$group": bson.M{
                                "_id":   "$kurum_adi",
                                "count": bson.M{"$sum": 1},
                        },
                },
                {
                        "$sort": bson.M{"_id": 1},
                },
        }

        cursor, err := collection.Aggregate(ctx, pipeline)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch institutions: "+err.Error())
                return
        }
        defer cursor.Close(ctx)

        var institutions []SitemapInstitution
        for cursor.Next(ctx) {
                var inst models.Institution
                if err := cursor.Decode(&inst); err != nil {
                        continue
                }
                
                // Create slug from institution name
                slug := createSlugFromName(inst.KurumAdi)
                
                sitemapInst := SitemapInstitution{
                        KurumAdi: inst.KurumAdi,
                        Count:    inst.Count,
                        Slug:     slug,
                }
                institutions = append(institutions, sitemapInst)
        }

        response := models.APIResponse{
                Success: true,
                Data:    institutions,
                Count:   len(institutions),
                Message: "Sitemap institutions fetched successfully",
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}

// GetSitemapDocumentsByInstitution returns all documents for a specific institution for sitemap
func GetSitemapDocumentsByInstitution(w http.ResponseWriter, r *http.Request) {
        if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
        }

        kurumAdi := r.URL.Query().Get("kurum_adi")
        if kurumAdi == "" {
                utils.SendErrorResponse(w, http.StatusBadRequest, "kurum_adi parameter is required")
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
        defer cancel()

        collection := config.GetMetadataCollection(mongoClient)

        filter := bson.M{
                "kurum_adi": kurumAdi,
                "status":    "aktif",
        }

        findOptions := options.Find()
        findOptions.SetSort(bson.M{"belge_yayin_tarihi": -1})
        findOptions.SetProjection(bson.M{
                "url_slug":              1,
                "pdf_adi":               1,
                "kurum_adi":             1,
                "belge_yayin_tarihi":    1,
                "olusturulma_tarihi":    1,
        })

        cursor, err := collection.Find(ctx, filter, findOptions)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch documents: "+err.Error())
                return
        }
        defer cursor.Close(ctx)

        var documents []SitemapDocument
        if err := cursor.All(ctx, &documents); err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to decode documents: "+err.Error())
                return
        }

        response := models.APIResponse{
                Success: true,
                Data:    documents,
                Count:   len(documents),
                Message: "Sitemap documents fetched successfully for: " + kurumAdi,
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}

// GetSitemapAllDocuments returns all documents for sitemap
func GetSitemapAllDocuments(w http.ResponseWriter, r *http.Request) {
        if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        collection := config.GetMetadataCollection(mongoClient)

        filter := bson.M{
                "status": "aktif",
        }

        findOptions := options.Find()
        findOptions.SetSort(bson.M{"belge_yayin_tarihi": -1})
        findOptions.SetProjection(bson.M{
                "url_slug":              1,
                "pdf_adi":               1,
                "kurum_adi":             1,
                "belge_yayin_tarihi":    1,
                "olusturulma_tarihi":    1,
        })

        cursor, err := collection.Find(ctx, filter, findOptions)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch all documents: "+err.Error())
                return
        }
        defer cursor.Close(ctx)

        var documents []SitemapDocument
        if err := cursor.All(ctx, &documents); err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to decode documents: "+err.Error())
                return
        }

        response := models.APIResponse{
                Success: true,
                Data:    documents,
                Count:   len(documents),
                Message: "All sitemap documents fetched successfully",
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}

// Helper function to create slug from institution name
func createSlugFromName(name string) string {
        slug := name
        slug = strings.ToLower(slug)
        slug = strings.ReplaceAll(slug, " ", "-")
        slug = strings.ReplaceAll(slug, "ç", "c")
        slug = strings.ReplaceAll(slug, "ğ", "g")
        slug = strings.ReplaceAll(slug, "ı", "i")
        slug = strings.ReplaceAll(slug, "ö", "o")
        slug = strings.ReplaceAll(slug, "ş", "s")
        slug = strings.ReplaceAll(slug, "ü", "u")
        return slug
}