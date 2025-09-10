package handlers

import (
        "context"
        "encoding/json"
        "log"
        "net/http"
        "strconv"
        "strings"
        "time"

        "go.mongodb.org/mongo-driver/bson"
        "go.mongodb.org/mongo-driver/bson/primitive"
        "go.mongodb.org/mongo-driver/mongo/options"

        "legal-documents-api/config"
        "legal-documents-api/models"
        "legal-documents-api/utils"
)

// SearchResult represents a search result
type SearchResult struct {
        ID                   string  `json:"id"`
        PdfAdi               string  `json:"pdf_adi"`
        KurumAdi             string  `json:"kurum_adi"`
        KurumLogo            string  `json:"kurum_logo"`
        BelgeTuru            string  `json:"belge_turu"`
        BelgeDurumu          string  `json:"belge_durumu"`
        BelgeYayinTarihi     string  `json:"belge_yayin_tarihi"`
        Etiketler            string  `json:"etiketler"`
        Aciklama             string  `json:"aciklama"`
        URLSlug              string  `json:"url_slug"`
        MatchType            string  `json:"match_type"`    // "title", "content", "tags", "institution"
        ContentPreview       string  `json:"content_preview,omitempty"`
        RelevanceScore       float64 `json:"relevance_score"`
        RelevancePercentage  int     `json:"relevance_percentage"`
        MatchCount           int     `json:"match_count"`
}

// GlobalSearch performs comprehensive search across titles, content, tags, and institutions
func GlobalSearch(w http.ResponseWriter, r *http.Request) {
        if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
        }

        // Get search query
        query := r.URL.Query().Get("q")
        if query == "" {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Search query 'q' parameter is required")
                return
        }

        // Get institution filters (optional)
        institution := r.URL.Query().Get("kurum")         // Institution name filter
        institutionID := r.URL.Query().Get("kurum_id")    // Institution ID filter (more efficient)
        
        // Log search parameters
        log.Printf("Search request - Query: '%s', Kurum: '%s', KurumID: '%s'", query, institution, institutionID)

        // Clean and prepare query
        query = strings.TrimSpace(query)
        if len(query) < 2 {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Search query must be at least 2 characters long")
                return
        }

        // Pagination parameters
        limitStr := r.URL.Query().Get("limit")
        offsetStr := r.URL.Query().Get("offset")
        
        limit := int64(20) // default limit for search
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

        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        // Search in multiple phases and combine results
        var allResults []SearchResult

        // Phase 1: Search in metadata (titles, descriptions, tags, institutions)
        metadataResults, err := searchInMetadata(ctx, query, institution, institutionID, limit*2) // Get more results to filter later
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to search metadata: "+err.Error())
                return
        }
        allResults = append(allResults, metadataResults...)

        // Phase 2: Search in content
        contentResults, err := searchInContent(ctx, query, institution, institutionID, limit*2)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to search content: "+err.Error())
                return
        }

        // Merge content results with metadata, avoiding duplicates
        allResults = mergeResults(allResults, contentResults)

        // Filter out results with zero relevance score
        var filteredResults []SearchResult
        for _, result := range allResults {
                if result.RelevanceScore > 0 {
                        filteredResults = append(filteredResults, result)
                }
        }

        // Sort by relevance score (higher is better)
        sortResultsByRelevance(filteredResults)

        // Apply pagination
        totalResults := len(filteredResults)
        start := int(offset)
        end := int(offset + limit)

        if start > totalResults {
                start = totalResults
        }
        if end > totalResults {
                end = totalResults
        }

        paginatedResults := filteredResults[start:end]

        response := models.APIResponse{
                Success: true,
                Data:    paginatedResults,
                Count:   len(paginatedResults),
                Message: "Search completed successfully",
        }

        // Add pagination metadata
        w.Header().Set("X-Total-Count", strconv.Itoa(totalResults))
        w.Header().Set("X-Limit", strconv.FormatInt(limit, 10))
        w.Header().Set("X-Offset", strconv.FormatInt(offset, 10))
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}

// searchInMetadata searches in document metadata (titles, descriptions, tags, institutions)
func searchInMetadata(ctx context.Context, query string, institution string, institutionID string, limit int64) ([]SearchResult, error) {
        collection := config.GetMetadataCollection(mongoClient)
        
        // Create regex for case-insensitive search
        searchRegex := bson.M{"$regex": primitive.Regex{Pattern: query, Options: "i"}}
        
        // Build base search conditions (removed kurum_adi since it's now a reference)
        searchConditions := []bson.M{
                {"pdf_adi": searchRegex},           // Search in title
                {"aciklama": searchRegex},          // Search in description  
                {"anahtar_kelimeler": searchRegex}, // Search in keywords
                {"etiketler": searchRegex},         // Search in tags
        }
        
        // Build comprehensive search filter with optional institution filter
        var filter bson.M
        var kurumID string
        
        // Priority: kurum_id > kurum (institution name)
        if institutionID != "" {
                // Direct kurum_id filter (most efficient)
                kurumID = institutionID
        } else if institution != "" {
                // Find kurum_id by kurum_adi from cache
                allKurumlar := utils.GetAllKurumlar()
                for _, kurum := range allKurumlar {
                        if strings.Contains(strings.ToLower(kurum.KurumAdi), strings.ToLower(institution)) {
                                kurumID = kurum.ID.Hex()
                                break
                        }
                }
        }
        
        if kurumID != "" {
                filter = bson.M{
                        "status": "aktif",
                        "$and": []bson.M{
                                {"kurum_id": kurumID},
                                {"$or": searchConditions},
                        },
                }
        } else if institution != "" || institutionID != "" {
                // Institution specified but not found, return empty filter
                filter = bson.M{"status": "aktif", "_id": bson.M{"$exists": false}}
        } else {
                // No institution filter
                filter = bson.M{
                        "status": "aktif",
                        "$or": searchConditions,
                }
        }

        findOptions := options.Find()
        findOptions.SetLimit(limit)
        findOptions.SetSort(bson.M{"belge_yayin_tarihi": -1})

        cursor, err := collection.Find(ctx, filter, findOptions)
        if err != nil {
                return nil, err
        }
        defer cursor.Close(ctx)

        var documents []models.DocumentMetadata
        if err := cursor.All(ctx, &documents); err != nil {
                return nil, err
        }

        var results []SearchResult
        for _, doc := range documents {
                // Get kurum info from cache
                kurumAdi := utils.GetKurumAdiByID(doc.KurumID)
                kurumLogo := utils.GetKurumLogoByID(doc.KurumID)

                result := SearchResult{
                        ID:                   doc.ID.Hex(),
                        PdfAdi:               doc.PdfAdi,
                        KurumAdi:             kurumAdi,
                        KurumLogo:            kurumLogo,
                        BelgeTuru:            doc.BelgeTuru,
                        BelgeDurumu:          doc.BelgeDurumu,
                        BelgeYayinTarihi:     doc.BelgeYayinTarihi,
                        Etiketler:            doc.Etiketler,
                        Aciklama:             truncateText(doc.Aciklama, 200),
                        URLSlug:              doc.URLSlug,
                        RelevanceScore:       calculateMetadataRelevance(doc, query),
                }

                // Determine match type based on where the query was found
                result.MatchType = determineMatchType(doc, query)
                result.RelevancePercentage = calculatePercentage(result.RelevanceScore)
                result.MatchCount = countMatches(doc, query)
                results = append(results, result)
        }

        return results, nil
}

// searchInContent searches in document content
func searchInContent(ctx context.Context, query string, institution string, institutionID string, limit int64) ([]SearchResult, error) {
        contentCollection := config.GetContentCollection(mongoClient)
        metadataCollection := config.GetMetadataCollection(mongoClient)
        
        // Search in content
        searchRegex := bson.M{"$regex": primitive.Regex{Pattern: query, Options: "i"}}
        contentFilter := bson.M{
                "icerik": searchRegex,
        }

        findOptions := options.Find()
        findOptions.SetLimit(limit)

        cursor, err := contentCollection.Find(ctx, contentFilter, findOptions)
        if err != nil {
                return nil, err
        }
        defer cursor.Close(ctx)

        var contents []models.DocumentContent
        if err := cursor.All(ctx, &contents); err != nil {
                return nil, err
        }

        var results []SearchResult
        for _, content := range contents {
                // Get corresponding metadata
                var metadata models.DocumentMetadata
                metadataFilter := bson.M{
                        "_id": content.MetadataID,
                        "status": "aktif",
                }
                
                // Add institution filter if specified
                var kurumID string
                if institutionID != "" {
                        // Direct kurum_id filter (most efficient)
                        kurumID = institutionID
                } else if institution != "" {
                        // Find kurum_id by kurum_adi from cache
                        allKurumlar := utils.GetAllKurumlar()
                        for _, kurum := range allKurumlar {
                                if strings.Contains(strings.ToLower(kurum.KurumAdi), strings.ToLower(institution)) {
                                        kurumID = kurum.ID.Hex()
                                        break
                                }
                        }
                }
                
                if kurumID != "" {
                        metadataFilter["kurum_id"] = kurumID
                }
                
                if err := metadataCollection.FindOne(ctx, metadataFilter).Decode(&metadata); err != nil {
                        continue // Skip if metadata not found or not active
                }

                // Get kurum info from cache
                kurumAdi := utils.GetKurumAdiByID(metadata.KurumID)
                kurumLogo := utils.GetKurumLogoByID(metadata.KurumID)

                result := SearchResult{
                        ID:                   metadata.ID.Hex(),
                        PdfAdi:               metadata.PdfAdi,
                        KurumAdi:             kurumAdi,
                        KurumLogo:            kurumLogo,
                        BelgeTuru:            metadata.BelgeTuru,
                        BelgeDurumu:          metadata.BelgeDurumu,
                        BelgeYayinTarihi:     metadata.BelgeYayinTarihi,
                        Etiketler:            metadata.Etiketler,
                        Aciklama:             truncateText(metadata.Aciklama, 200),
                        URLSlug:              metadata.URLSlug,
                        MatchType:            "content",
                        ContentPreview:       extractContentPreview(content.Icerik, query),
                        RelevanceScore:       calculateContentRelevance(content.Icerik, query),
                }

                result.RelevancePercentage = calculatePercentage(result.RelevanceScore)
                result.MatchCount = countContentMatches(content.Icerik, query)
                results = append(results, result)
        }

        return results, nil
}

// mergeResults combines metadata and content results, removing duplicates
func mergeResults(metadataResults, contentResults []SearchResult) []SearchResult {
        resultMap := make(map[string]SearchResult)
        
        // Add metadata results
        for _, result := range metadataResults {
                resultMap[result.ID] = result
        }
        
        // Add content results, but merge with existing metadata results if duplicate
        for _, contentResult := range contentResults {
                if existing, exists := resultMap[contentResult.ID]; exists {
                        // Merge: keep higher relevance score and combine match types
                        if contentResult.RelevanceScore > existing.RelevanceScore {
                                existing.RelevanceScore = contentResult.RelevanceScore
                                existing.RelevancePercentage = contentResult.RelevancePercentage
                        }
                        existing.MatchType = existing.MatchType + "+content"
                        existing.ContentPreview = contentResult.ContentPreview
                        existing.MatchCount += contentResult.MatchCount // Add content matches to metadata matches
                        resultMap[contentResult.ID] = existing
                } else {
                        resultMap[contentResult.ID] = contentResult
                }
        }
        
        // Convert map back to slice
        var mergedResults []SearchResult
        for _, result := range resultMap {
                mergedResults = append(mergedResults, result)
        }
        
        return mergedResults
}

// Helper functions
func calculateMetadataRelevance(doc models.DocumentMetadata, query string) float64 {
        score := 0.0
        queryLower := strings.ToLower(query)
        
        // Title match (highest weight)
        if strings.Contains(strings.ToLower(doc.PdfAdi), queryLower) {
                score += 10.0
        }
        
        // Institution match - get from cache
        kurumAdi := utils.GetKurumAdiByID(doc.KurumID)
        if strings.Contains(strings.ToLower(kurumAdi), queryLower) {
                score += 5.0
        }
        
        // Tags match
        if strings.Contains(strings.ToLower(doc.Etiketler), queryLower) {
                score += 3.0
        }
        
        // Keywords match
        if strings.Contains(strings.ToLower(doc.AnahtarKelimeler), queryLower) {
                score += 2.0
        }
        
        // Description match
        if strings.Contains(strings.ToLower(doc.Aciklama), queryLower) {
                score += 1.0
        }
        
        return score
}

func calculateContentRelevance(content, query string) float64 {
        queryLower := strings.ToLower(query)
        contentLower := strings.ToLower(content)
        
        // Count occurrences
        occurrences := strings.Count(contentLower, queryLower)
        
        // Base score from occurrences
        score := float64(occurrences) * 0.5
        
        // Bonus for content length (shorter content with matches is more relevant)
        if len(content) > 0 {
                density := float64(len(query)*occurrences) / float64(len(content))
                score += density * 100
        }
        
        return score
}

func determineMatchType(doc models.DocumentMetadata, query string) string {
        queryLower := strings.ToLower(query)
        
        if strings.Contains(strings.ToLower(doc.PdfAdi), queryLower) {
                return "title"
        }
        // Check institution name from cache
        kurumAdi := utils.GetKurumAdiByID(doc.KurumID)
        if strings.Contains(strings.ToLower(kurumAdi), queryLower) {
                return "institution"
        }
        if strings.Contains(strings.ToLower(doc.Etiketler), queryLower) {
                return "tags"
        }
        if strings.Contains(strings.ToLower(doc.AnahtarKelimeler), queryLower) {
                return "keywords"
        }
        return "description"
}

func extractContentPreview(content, query string) string {
        queryLower := strings.ToLower(query)
        contentLower := strings.ToLower(content)
        
        // Find the first occurrence of the query
        index := strings.Index(contentLower, queryLower)
        if index == -1 {
                return truncateText(content, 150)
        }
        
        // Extract surrounding context
        start := index - 75
        if start < 0 {
                start = 0
        }
        
        end := index + len(query) + 75
        if end > len(content) {
                end = len(content)
        }
        
        preview := content[start:end]
        if start > 0 {
                preview = "..." + preview
        }
        if end < len(content) {
                preview = preview + "..."
        }
        
        return preview
}

func truncateText(text string, maxLength int) string {
        if len(text) <= maxLength {
                return text
        }
        return text[:maxLength] + "..."
}

func sortResultsByRelevance(results []SearchResult) {
        // Simple bubble sort by relevance score (descending)
        n := len(results)
        for i := 0; i < n-1; i++ {
                for j := 0; j < n-i-1; j++ {
                        if results[j].RelevanceScore < results[j+1].RelevanceScore {
                                results[j], results[j+1] = results[j+1], results[j]
                        }
                }
        }
}

// calculatePercentage converts relevance score to percentage (0-100)
func calculatePercentage(score float64) int {
        if score <= 0 {
                return 0
        }
        
        // Normalize different score ranges
        var percentage float64
        
        if score >= 1000 {
                // Very high relevance (title + content matches)
                percentage = 95 + (score-1000)/10000*5  // 95-100%
        } else if score >= 100 {
                // High relevance (multiple matches)
                percentage = 80 + (score-100)/900*15     // 80-95%
        } else if score >= 50 {
                // Medium-high relevance
                percentage = 60 + (score-50)/50*20       // 60-80%
        } else if score >= 10 {
                // Medium relevance
                percentage = 30 + (score-10)/40*30       // 30-60%
        } else if score >= 1 {
                // Low relevance
                percentage = 10 + (score-1)/9*20         // 10-30%
        } else {
                // Very low relevance
                percentage = score * 10                  // 0-10%
        }
        
        // Cap at 100%
        if percentage > 100 {
                percentage = 100
        }
        
        return int(percentage)
}

// countMatches counts how many times the query appears in metadata fields
func countMatches(doc models.DocumentMetadata, query string) int {
        count := 0
        queryLower := strings.ToLower(query)
        
        // Count in title
        count += strings.Count(strings.ToLower(doc.PdfAdi), queryLower)
        
        // Count in institution name from cache
        kurumAdi := utils.GetKurumAdiByID(doc.KurumID)
        count += strings.Count(strings.ToLower(kurumAdi), queryLower)
        
        // Count in tags
        count += strings.Count(strings.ToLower(doc.Etiketler), queryLower)
        
        // Count in keywords
        count += strings.Count(strings.ToLower(doc.AnahtarKelimeler), queryLower)
        
        // Count in description
        count += strings.Count(strings.ToLower(doc.Aciklama), queryLower)
        
        return count
}

// countContentMatches counts how many times the query appears in content
func countContentMatches(content, query string) int {
        queryLower := strings.ToLower(query)
        contentLower := strings.ToLower(content)
        
        return strings.Count(contentLower, queryLower)
}