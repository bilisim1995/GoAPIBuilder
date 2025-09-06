package handlers

import (
        "context"
        "encoding/json"
        "net/http"
        "strconv"
        "strings"
        "time"

        "github.com/gorilla/mux"
        "go.mongodb.org/mongo-driver/bson"
        "go.mongodb.org/mongo-driver/bson/primitive"
        "go.mongodb.org/mongo-driver/mongo/options"

        "legal-documents-api/config"
        "legal-documents-api/models"
        "legal-documents-api/utils"
)

// GetDocumentsByInstitution returns documents filtered by institution
func GetDocumentsByInstitution(w http.ResponseWriter, r *http.Request) {
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

        // Pagination parameters
        limitStr := r.URL.Query().Get("limit")
        offsetStr := r.URL.Query().Get("offset")
        
        limit := int64(50) // default limit
        offset := int64(0) // default offset

        if limitStr != "" {
                if parsedLimit, err := strconv.ParseInt(limitStr, 10, 64); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
                        limit = parsedLimit
                }
        }

        if offsetStr != "" {
                if parsedOffset, err := strconv.ParseInt(offsetStr, 10, 64); err == nil && parsedOffset >= 0 {
                        offset = parsedOffset
                }
        }

        collection := config.GetMetadataCollection(mongoClient)

        // Build filter directly with kurum_id
        filter := bson.M{
                "kurum_id": kurumID,
                "status":   "aktif",
        }

        // Additional filters
        belgeTuru := r.URL.Query().Get("belge_turu")
        if belgeTuru != "" {
                filter["belge_turu"] = belgeTuru
        }

        belgeDurumu := r.URL.Query().Get("belge_durumu")
        if belgeDurumu != "" {
                filter["belge_durumu"] = belgeDurumu
        }

        // Search in title or description
        search := r.URL.Query().Get("search")
        if search != "" {
                searchRegex := bson.M{"$regex": primitive.Regex{Pattern: search, Options: "i"}}
                filter["$or"] = []bson.M{
                        {"pdf_adi": searchRegex},
                        {"aciklama": searchRegex},
                        {"anahtar_kelimeler": searchRegex},
                }
        }

        // Count total documents
        totalCount, err := collection.CountDocuments(ctx, filter)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to count documents: "+err.Error())
                return
        }

        // Setup find options
        findOptions := options.Find()
        findOptions.SetLimit(limit)
        findOptions.SetSkip(offset)
        findOptions.SetSort(bson.D{primitive.E{Key: "belge_yayin_tarihi", Value: -1}}) // Sort by publication date, newest first

        // Only select necessary fields for summary
        findOptions.SetProjection(bson.M{
                "_id":                1,
                "kurum_id":           1,  // New field for institution reference
                "pdf_adi":            1,
                "etiketler":          1,
                "belge_yayin_tarihi": 1,
                "belge_durumu":       1,
                "aciklama":           1,
                "url_slug":           1,
        })

        cursor, err := collection.Find(ctx, filter, findOptions)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch documents: "+err.Error())
                return
        }
        defer cursor.Close(ctx)

        var documents []models.DocumentMetadata
        if err := cursor.All(ctx, &documents); err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to decode documents: "+err.Error())
                return
        }

        // Convert to summary format
        var summaries []models.DocumentSummary
        for _, doc := range documents {
                // Truncate description if too long
                aciklama := doc.Aciklama
                if len(aciklama) > 200 {
                        aciklama = aciklama[:200] + "..."
                }

                // Get kurum info from cache
                kurumAdi := utils.GetKurumAdiByID(doc.KurumID)
                kurumLogo := utils.GetKurumLogoByID(doc.KurumID)
                kurumAciklama := utils.GetKurumAciklamaByID(doc.KurumID)

                summary := models.DocumentSummary{
                        ID:               doc.ID.Hex(),
                        KurumAdi:         kurumAdi,
                        KurumLogo:        kurumLogo,
                        KurumAciklama:    kurumAciklama,
                        PdfAdi:           doc.PdfAdi,
                        Etiketler:        doc.Etiketler,
                        BelgeYayinTarihi: doc.BelgeYayinTarihi,
                        BelgeDurumu:      doc.BelgeDurumu,
                        Aciklama:         aciklama,
                        URLSlug:          doc.URLSlug,
                }
                summaries = append(summaries, summary)
        }

        // Prepare response with pagination info
        response := models.APIResponse{
                Success: true,
                Data:    summaries,
                Count:   len(summaries),
                Message: "Documents fetched successfully",
        }

        // Add pagination metadata in headers
        w.Header().Set("X-Total-Count", strconv.FormatInt(totalCount, 10))
        w.Header().Set("X-Limit", strconv.FormatInt(limit, 10))
        w.Header().Set("X-Offset", strconv.FormatInt(offset, 10))
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}

// GetDocumentBySlug returns complete document details including content
func GetDocumentBySlug(w http.ResponseWriter, r *http.Request) {
        // Handle CORS preflight
        if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
        defer cancel()

        // Get slug from URL parameters
        vars := mux.Vars(r)
        slug := vars["slug"]
        if slug == "" {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Document slug is required")
                return
        }

        // Clean and validate slug
        slug = strings.TrimSpace(slug)
        if len(slug) < 3 {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid document slug")
                return
        }

        metadataCollection := config.GetMetadataCollection(mongoClient)
        contentCollection := config.GetContentCollection(mongoClient)

        // Find metadata by slug
        var metadata models.DocumentMetadata
        filter := bson.M{
                "url_slug": slug,
                "status":   "aktif",
        }

        if err := metadataCollection.FindOne(ctx, filter).Decode(&metadata); err != nil {
                if err.Error() == "mongo: no documents in result" {
                        utils.SendErrorResponse(w, http.StatusNotFound, "Document not found")
                        return
                }
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch document metadata: "+err.Error())
                return
        }

        // Find content by metadata ID
        var content models.DocumentContent
        contentFilter := bson.M{"metadata_id": metadata.ID}
        
        if err := contentCollection.FindOne(ctx, contentFilter).Decode(&content); err != nil {
                if err.Error() == "mongo: no documents in result" {
                        utils.SendErrorResponse(w, http.StatusNotFound, "Document content not found")
                        return
                }
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch document content: "+err.Error())
                return
        }

        // Get kurum info from cache using kurum_id from metadata
        kurumAdi := utils.GetKurumAdiByID(metadata.KurumID)
        kurumLogo := utils.GetKurumLogoByID(metadata.KurumID)
        kurumAciklama := utils.GetKurumAciklamaByID(metadata.KurumID)

        // Combine metadata and content with kurum info
        documentDetails := models.DocumentDetails{
                Metadata:      metadata,
                Content:       content,
                KurumAdi:      kurumAdi,
                KurumLogo:     kurumLogo,
                KurumAciklama: kurumAciklama,
        }

        response := models.APIResponse{
                Success: true,
                Data:    documentDetails,
                Message: "Document details fetched successfully",
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}

// GetDocumentsByInstitutionSlug returns documents filtered by institution using URL slug
func GetDocumentsByInstitutionSlug(w http.ResponseWriter, r *http.Request) {
        // This endpoint uses kurumSlugID variable instead of kurumID
        // Handle CORS preflight
        if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
        }

        // Get kurum_slug from URL parameters
        vars := mux.Vars(r)
        kurumSlug := vars["kurum_slug"]
        if kurumSlug == "" {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Institution slug is required")
                return
        }

        // Find kurum_id by matching slug with kurum names
        var kurumID string
        allKurumlar := utils.GetAllKurumlar()
        for _, kurum := range allKurumlar {
                // Match by slug or name variations
                kurumSlugNormalized := strings.ReplaceAll(strings.ToLower(kurum.KurumAdi), " ", "-")
                if strings.EqualFold(kurumSlugNormalized, kurumSlug) || 
                   strings.EqualFold(kurum.KurumAdi, kurumSlug) {
                        kurumID = kurum.ID.Hex()
                        break
                }
        }
        
        if kurumID == "" {
                utils.SendErrorResponse(w, http.StatusNotFound, "Institution not found: "+kurumSlug)
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
        defer cancel()

        // Get pagination parameters
        limitStr := r.URL.Query().Get("limit")
        offsetStr := r.URL.Query().Get("offset")
        
        limit := int64(50) // default limit
        offset := int64(0) // default offset

        if limitStr != "" {
                if parsedLimit, err := strconv.ParseInt(limitStr, 10, 64); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
                        limit = parsedLimit
                }
        }

        if offsetStr != "" {
                if parsedOffset, err := strconv.ParseInt(offsetStr, 10, 64); err == nil && parsedOffset >= 0 {
                        offset = parsedOffset
                }
        }

        collection := config.GetMetadataCollection(mongoClient)

        // Build filter directly with kurum_id
        filter := bson.M{
                "kurum_id": kurumID,
                "status":   "aktif",
        }

        // Additional filters from query parameters
        belgeTuru := r.URL.Query().Get("belge_turu")
        if belgeTuru != "" {
                filter["belge_turu"] = belgeTuru
        }

        belgeDurumu := r.URL.Query().Get("belge_durumu")
        if belgeDurumu != "" {
                filter["belge_durumu"] = belgeDurumu
        }

        // Search in title or description
        search := r.URL.Query().Get("search")
        if search != "" {
                searchRegex := bson.M{"$regex": primitive.Regex{Pattern: search, Options: "i"}}
                filter["$or"] = []bson.M{
                        {"pdf_adi": searchRegex},
                        {"aciklama": searchRegex},
                        {"anahtar_kelimeler": searchRegex},
                }
        }

        // Count total documents
        totalCount, err := collection.CountDocuments(ctx, filter)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to count documents: "+err.Error())
                return
        }

        // Setup find options
        findOptions := options.Find()
        findOptions.SetLimit(limit)
        findOptions.SetSkip(offset)
        findOptions.SetSort(bson.D{primitive.E{Key: "belge_yayin_tarihi", Value: -1}}) // Sort by publication date, newest first

        // Only select necessary fields for summary
        findOptions.SetProjection(bson.M{
                "_id":                1,
                "kurum_id":           1,  // New field for institution reference
                "pdf_adi":            1,
                "etiketler":          1,
                "belge_yayin_tarihi": 1,
                "belge_durumu":       1,
                "aciklama":           1,
                "url_slug":           1,
        })

        cursor, err := collection.Find(ctx, filter, findOptions)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch documents: "+err.Error())
                return
        }
        defer cursor.Close(ctx)

        var documents []models.DocumentMetadata
        if err := cursor.All(ctx, &documents); err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to decode documents: "+err.Error())
                return
        }

        // Convert to summary format
        var summaries []models.DocumentSummary
        for _, doc := range documents {
                // Truncate description if too long
                aciklama := doc.Aciklama
                if len(aciklama) > 200 {
                        aciklama = aciklama[:200] + "..."
                }

                // Get kurum info from cache
                kurumAdi := utils.GetKurumAdiByID(doc.KurumID)
                kurumLogo := utils.GetKurumLogoByID(doc.KurumID)
                kurumAciklama := utils.GetKurumAciklamaByID(doc.KurumID)

                summary := models.DocumentSummary{
                        ID:               doc.ID.Hex(),
                        KurumAdi:         kurumAdi,
                        KurumLogo:        kurumLogo,
                        KurumAciklama:    kurumAciklama,
                        PdfAdi:           doc.PdfAdi,
                        Etiketler:        doc.Etiketler,
                        BelgeYayinTarihi: doc.BelgeYayinTarihi,
                        BelgeDurumu:      doc.BelgeDurumu,
                        Aciklama:         aciklama,
                        URLSlug:          doc.URLSlug,
                }
                summaries = append(summaries, summary)
        }

        // If no documents found, return helpful message
        if len(summaries) == 0 {
                response := models.APIResponse{
                        Success: true,
                        Data:    []models.DocumentSummary{},
                        Count:   0,
                        Message: "No documents found for institution slug: " + kurumSlug,
                }
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                json.NewEncoder(w).Encode(response)
                return
        }

        // Prepare response with pagination info
        response := models.APIResponse{
                Success: true,
                Data:    summaries,
                Count:   len(summaries),
                Message: "Documents fetched successfully",
        }

        // Add pagination metadata in headers
        w.Header().Set("X-Total-Count", strconv.FormatInt(totalCount, 10))
        w.Header().Set("X-Limit", strconv.FormatInt(limit, 10))
        w.Header().Set("X-Offset", strconv.FormatInt(offset, 10))
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}
