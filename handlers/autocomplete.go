package handlers

import (
        "context"
        "encoding/json"
        "net/http"
        "regexp"
        "sort"
        "strconv"
        "strings"
        "time"

        "go.mongodb.org/mongo-driver/bson"
        "go.mongodb.org/mongo-driver/bson/primitive"
        "go.mongodb.org/mongo-driver/mongo"
        "go.mongodb.org/mongo-driver/mongo/options"

        "legal-documents-api/config"
        "legal-documents-api/models"
        "legal-documents-api/utils"
)

// SuggestionItem represents an autocomplete suggestion
type SuggestionItem struct {
        Text  string `json:"text"`
        Count int    `json:"count"`
        Type  string `json:"type"` // "title", "keyword", "tag", "institution"
}

// AutocompleteResponse represents the response structure for autocomplete
type AutocompleteResponse struct {
        Suggestions []SuggestionItem `json:"suggestions"`
}

// Autocomplete provides word suggestions based on partial input
func Autocomplete(w http.ResponseWriter, r *http.Request) {
        if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
        }

        // Get query parameter
        query := r.URL.Query().Get("q")
        if query == "" {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Query parameter 'q' is required")
                return
        }

        // Clean and prepare query
        query = strings.TrimSpace(query)
        if len(query) < 2 {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Query must be at least 2 characters long")
                return
        }

        // Get limit parameter
        limitStr := r.URL.Query().Get("limit")
        limit := 10 // default limit
        if limitStr != "" {
                if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
                        limit = parsedLimit
                }
        }

        // Get institution filter (optional)
        institution := r.URL.Query().Get("kurum")

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        // Get suggestions from database
        suggestions, err := getSuggestions(ctx, query, institution, limit)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to get suggestions: "+err.Error())
                return
        }

        response := models.APIResponse{
                Success: true,
                Data: AutocompleteResponse{
                        Suggestions: suggestions,
                },
                Message: "Suggestions retrieved successfully",
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}

// getSuggestions retrieves word suggestions from MongoDB
func getSuggestions(ctx context.Context, query string, institution string, limit int) ([]SuggestionItem, error) {
        collection := config.GetMetadataCollection(mongoClient)
        
        // Create case-insensitive regex pattern
        pattern := "(?i)" + regexp.QuoteMeta(query)
        searchRegex := bson.M{"$regex": primitive.Regex{Pattern: pattern, Options: "i"}}
        
        // Build base filter
        baseFilter := bson.M{"status": "aktif"}
        
        // Add institution filter if specified (using kurum_id from cache)
        if institution != "" {
                // Find kurum_id by kurum_adi from cache
                var kurumID string
                allKurumlar := utils.GetAllKurumlar()
                for _, kurum := range allKurumlar {
                        if strings.Contains(strings.ToLower(kurum.KurumAdi), strings.ToLower(institution)) {
                                kurumID = kurum.ID.Hex()
                                break
                        }
                }
                
                if kurumID != "" {
                        baseFilter["kurum_id"] = kurumID
                } else {
                        // Institution not found, return empty filter to get no results
                        baseFilter["_id"] = bson.M{"$exists": false}
                }
        }

        // Map to store unique suggestions with their counts
        suggestionMap := make(map[string]*SuggestionItem)

        // Search in titles (highest priority)
        titleFilter := bson.M{"pdf_adi": searchRegex}
        for k, v := range baseFilter {
                titleFilter[k] = v
        }
        
        if err := extractSuggestions(ctx, collection, titleFilter, "pdf_adi", "title", query, suggestionMap); err != nil {
                return nil, err
        }

        // Search in keywords
        keywordFilter := bson.M{"anahtar_kelimeler": searchRegex}
        for k, v := range baseFilter {
                keywordFilter[k] = v
        }
        
        if err := extractSuggestions(ctx, collection, keywordFilter, "anahtar_kelimeler", "keyword", query, suggestionMap); err != nil {
                return nil, err
        }

        // Search in tags
        tagFilter := bson.M{"etiketler": searchRegex}
        for k, v := range baseFilter {
                tagFilter[k] = v
        }
        
        if err := extractSuggestions(ctx, collection, tagFilter, "etiketler", "tag", query, suggestionMap); err != nil {
                return nil, err
        }

        // Search in institution names (from cache, not database field)
        // Since kurum_adi is no longer in metadata, we'll add institution suggestions from cache
        allKurumlar := utils.GetAllKurumlar()
        queryLower := strings.ToLower(query)
        for _, kurum := range allKurumlar {
                kurumAdiLower := strings.ToLower(kurum.KurumAdi)
                if strings.Contains(kurumAdiLower, queryLower) {
                        // Extract words from institution name
                        words := extractRelevantWords(kurum.KurumAdi, queryLower)
                        for _, word := range words {
                                wordLower := strings.ToLower(word)
                                if !strings.Contains(wordLower, queryLower) {
                                        continue
                                }
                                
                                if existing, exists := suggestionMap[wordLower]; exists {
                                        existing.Count++
                                        if getTypePriority("institution") < getTypePriority(existing.Type) {
                                                existing.Type = "institution"
                                        }
                                } else {
                                        suggestionMap[wordLower] = &SuggestionItem{
                                                Text:  word,
                                                Count: 1,
                                                Type:  "institution",
                                        }
                                }
                        }
                }
        }

        // Convert map to slice and sort by relevance
        suggestions := make([]SuggestionItem, 0, len(suggestionMap))
        for _, suggestion := range suggestionMap {
                suggestions = append(suggestions, *suggestion)
        }

        // Sort by count (descending) and then by type priority
        sort.Slice(suggestions, func(i, j int) bool {
                if suggestions[i].Count == suggestions[j].Count {
                        return getTypePriority(suggestions[i].Type) < getTypePriority(suggestions[j].Type)
                }
                return suggestions[i].Count > suggestions[j].Count
        })

        // Limit results
        if len(suggestions) > limit {
                suggestions = suggestions[:limit]
        }

        return suggestions, nil
}

// extractSuggestions extracts word suggestions from a specific field
func extractSuggestions(ctx context.Context, collection *mongo.Collection, filter bson.M, field string, suggestionType string, query string, suggestionMap map[string]*SuggestionItem) error {
        findOptions := options.Find()
        findOptions.SetLimit(100) // Limit documents to process
        findOptions.SetProjection(bson.M{field: 1})

        cursor, err := collection.Find(ctx, filter, findOptions)
        if err != nil {
                return err
        }
        defer cursor.Close(ctx)

        queryLower := strings.ToLower(query)
        
        for cursor.Next(ctx) {
                var doc bson.M
                if err := cursor.Decode(&doc); err != nil {
                        continue
                }

                fieldValue, ok := doc[field].(string)
                if !ok {
                        continue
                }

                // Extract words that start with or contain the query
                words := extractRelevantWords(fieldValue, queryLower)
                
                for _, word := range words {
                        wordLower := strings.ToLower(word)
                        
                        // Skip if word doesn't contain the query
                        if !strings.Contains(wordLower, queryLower) {
                                continue
                        }

                        // Add to suggestion map or increase count
                        if existing, exists := suggestionMap[wordLower]; exists {
                                existing.Count++
                                // Prefer higher priority types
                                if getTypePriority(suggestionType) < getTypePriority(existing.Type) {
                                        existing.Type = suggestionType
                                }
                        } else {
                                suggestionMap[wordLower] = &SuggestionItem{
                                        Text:  word,
                                        Count: 1,
                                        Type:  suggestionType,
                                }
                        }
                }
        }

        return cursor.Err()
}

// extractRelevantWords extracts words from text that are relevant to the query
func extractRelevantWords(text, queryLower string) []string {
        // Split text into words and clean them
        words := strings.Fields(text)
        var relevantWords []string
        
        for _, word := range words {
                // Clean word (remove punctuation, etc.)
                cleanWord := regexp.MustCompile(`[^\p{L}\p{N}]`).ReplaceAllString(word, "")
                if len(cleanWord) < 2 {
                        continue
                }
                
                // Check if word contains the query
                if strings.Contains(strings.ToLower(cleanWord), queryLower) {
                        relevantWords = append(relevantWords, cleanWord)
                }
        }
        
        return relevantWords
}

// getTypePriority returns priority order for suggestion types (lower = higher priority)
func getTypePriority(suggestionType string) int {
        switch suggestionType {
        case "title":
                return 1
        case "keyword":
                return 2
        case "tag":
                return 3
        case "institution":
                return 4
        default:
                return 5
        }
}