package handlers

import (
        "context"
        "encoding/json"
        "fmt"
        "net/http"
        "strconv"
        "time"

        "go.mongodb.org/mongo-driver/bson"
        "go.mongodb.org/mongo-driver/bson/primitive"
        "go.mongodb.org/mongo-driver/mongo"
        "go.mongodb.org/mongo-driver/mongo/options"

        "legal-documents-api/config"
        "legal-documents-api/models"
        "legal-documents-api/utils"
)

// Use the global mongoClient from handlers package

// CreateInstitution creates a new institution
func CreateInstitution(w http.ResponseWriter, r *http.Request) {
        if r.Method == "OPTIONS" {
                utils.HandleCORS(w, r)
                return
        }

        if r.Method != http.MethodPost {
                utils.SendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
                return
        }

        utils.HandleCORS(w, r)

        var req models.InstitutionCreateRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
                return
        }

        // Validate required fields
        if req.KurumAdi == "" {
                utils.SendErrorResponse(w, http.StatusBadRequest, "kurum_adi is required")
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        collection := config.GetInstitutionsCollection(MongoClient)

        // Check if institution with same name already exists
        existingCount, err := collection.CountDocuments(ctx, bson.M{"kurum_adi": req.KurumAdi})
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
                return
        }

        if existingCount > 0 {
                utils.SendErrorResponse(w, http.StatusConflict, "Institution with this name already exists")
                return
        }

        // Create new institution
        now := time.Now().Format("2006-01-02 15:04:05")
        institution := models.InstitutionModel{
                ID:        primitive.NewObjectID(),
                KurumAdi:  req.KurumAdi,
                KurumLogo: req.KurumLogo,
                Aciklama:  req.Aciklama,
                Website:   req.Website,
                Aktif:     true,
                CreatedAt: now,
                UpdatedAt: now,
        }

        result, err := collection.InsertOne(ctx, institution)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to create institution: "+err.Error())
                return
        }

        institution.ID = result.InsertedID.(primitive.ObjectID)

        response := models.APIResponse{
                Success: true,
                Data:    institution,
                Message: "Institution created successfully",
                Count:   1,
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(response)
}

// GetInstitutions lists all institutions with pagination
func GetInstitutions(w http.ResponseWriter, r *http.Request) {
        if r.Method == "OPTIONS" {
                utils.HandleCORS(w, r)
                return
        }

        if r.Method != http.MethodGet {
                utils.SendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
                return
        }

        utils.HandleCORS(w, r)

        // Parse pagination parameters
        page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
        if page < 1 {
                page = 1
        }

        limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
        if limit < 1 || limit > 100 {
                limit = 20
        }

        offset := (page - 1) * limit

        // Parse filter parameters
        filter := bson.M{}
        
        search := r.URL.Query().Get("search")
        if search != "" {
                filter["kurum_adi"] = bson.M{"$regex": primitive.Regex{Pattern: search, Options: "i"}}
        }

        aktif := r.URL.Query().Get("aktif")
        if aktif != "" {
                if aktif == "true" {
                        filter["aktif"] = true
                } else if aktif == "false" {
                        filter["aktif"] = false
                }
        }

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        collection := config.GetInstitutionsCollection(MongoClient)

        // Count total documents
        totalCount, err := collection.CountDocuments(ctx, filter)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to count institutions: "+err.Error())
                return
        }

        // Find institutions with pagination
        findOptions := options.Find()
        findOptions.SetLimit(limit)
        findOptions.SetSkip(offset)
        findOptions.SetSort(bson.D{primitive.E{Key: "kurum_adi", Value: 1}}) // Sort alphabetically

        cursor, err := collection.Find(ctx, filter, findOptions)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch institutions: "+err.Error())
                return
        }
        defer cursor.Close(ctx)

        var institutions []models.InstitutionModel
        if err := cursor.All(ctx, &institutions); err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to decode institutions: "+err.Error())
                return
        }

        response := models.APIResponse{
                Success: true,
                Data:    institutions,
                Message: "Institutions fetched successfully",
                Count:   len(institutions),
        }

        // Add pagination metadata
        w.Header().Set("X-Total-Count", strconv.FormatInt(totalCount, 10))
        w.Header().Set("X-Limit", strconv.FormatInt(limit, 10))
        w.Header().Set("X-Offset", strconv.FormatInt(offset, 10))
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}

// GetInstitutionByID gets a single institution by ID
func GetInstitutionByID(w http.ResponseWriter, r *http.Request) {
        if r.Method == "OPTIONS" {
                utils.HandleCORS(w, r)
                return
        }

        if r.Method != http.MethodGet {
                utils.SendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
                return
        }

        utils.HandleCORS(w, r)

        // Extract ID from URL path
        id := r.URL.Path[len("/api/v1/institutions/"):]
        if id == "" {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Institution ID is required")
                return
        }

        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid institution ID format")
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        collection := config.GetInstitutionsCollection(MongoClient)

        var institution models.InstitutionModel
        err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&institution)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        utils.SendErrorResponse(w, http.StatusNotFound, "Institution not found")
                        return
                }
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
                return
        }

        response := models.APIResponse{
                Success: true,
                Data:    institution,
                Message: "Institution fetched successfully",
                Count:   1,
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}

// UpdateInstitution updates an existing institution
func UpdateInstitution(w http.ResponseWriter, r *http.Request) {
        if r.Method == "OPTIONS" {
                utils.HandleCORS(w, r)
                return
        }

        if r.Method != http.MethodPut && r.Method != http.MethodPatch {
                utils.SendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
                return
        }

        utils.HandleCORS(w, r)

        // Extract ID from URL path
        id := r.URL.Path[len("/api/v1/institutions/"):]
        if id == "" {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Institution ID is required")
                return
        }

        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid institution ID format")
                return
        }

        var req models.InstitutionUpdateRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        collection := config.GetInstitutionsCollection(MongoClient)

        // Check if institution exists
        var existingInstitution models.InstitutionModel
        err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&existingInstitution)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        utils.SendErrorResponse(w, http.StatusNotFound, "Institution not found")
                        return
                }
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
                return
        }

        // Build update document
        updateDoc := bson.M{}
        if req.KurumAdi != "" {
                // Check if new name conflicts with existing institution
                if req.KurumAdi != existingInstitution.KurumAdi {
                        existingCount, err := collection.CountDocuments(ctx, bson.M{
                                "kurum_adi": req.KurumAdi,
                                "_id":       bson.M{"$ne": objectID},
                        })
                        if err != nil {
                                utils.SendErrorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
                                return
                        }
                        if existingCount > 0 {
                                utils.SendErrorResponse(w, http.StatusConflict, "Institution with this name already exists")
                                return
                        }
                }
                updateDoc["kurum_adi"] = req.KurumAdi
        }

        if req.KurumLogo != "" {
                updateDoc["kurum_logo"] = req.KurumLogo
        }

        if req.Aciklama != "" {
                updateDoc["aciklama"] = req.Aciklama
        }

        if req.Website != "" {
                updateDoc["website"] = req.Website
        }

        if req.Aktif != nil {
                updateDoc["aktif"] = *req.Aktif
        }

        updateDoc["updated_at"] = time.Now().Format("2006-01-02 15:04:05")

        // Perform update
        result, err := collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": updateDoc})
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to update institution: "+err.Error())
                return
        }

        if result.MatchedCount == 0 {
                utils.SendErrorResponse(w, http.StatusNotFound, "Institution not found")
                return
        }

        // Fetch updated institution
        var updatedInstitution models.InstitutionModel
        err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&updatedInstitution)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch updated institution: "+err.Error())
                return
        }

        response := models.APIResponse{
                Success: true,
                Data:    updatedInstitution,
                Message: "Institution updated successfully",
                Count:   1,
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}

// DeleteInstitution deletes an institution (soft delete - sets aktif to false)
func DeleteInstitution(w http.ResponseWriter, r *http.Request) {
        if r.Method == "OPTIONS" {
                utils.HandleCORS(w, r)
                return
        }

        if r.Method != http.MethodDelete {
                utils.SendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
                return
        }

        utils.HandleCORS(w, r)

        // Extract ID from URL path
        id := r.URL.Path[len("/api/v1/institutions/"):]
        if id == "" {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Institution ID is required")
                return
        }

        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusBadRequest, "Invalid institution ID format")
                return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        collection := config.GetInstitutionsCollection(MongoClient)

        // Check if institution exists
        var institution models.InstitutionModel
        err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&institution)
        if err != nil {
                if err == mongo.ErrNoDocuments {
                        utils.SendErrorResponse(w, http.StatusNotFound, "Institution not found")
                        return
                }
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Database error: "+err.Error())
                return
        }

        // Check if force delete is requested
        forceDelete := r.URL.Query().Get("force") == "true"

        if forceDelete {
                // Hard delete
                result, err := collection.DeleteOne(ctx, bson.M{"_id": objectID})
                if err != nil {
                        utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to delete institution: "+err.Error())
                        return
                }

                if result.DeletedCount == 0 {
                        utils.SendErrorResponse(w, http.StatusNotFound, "Institution not found")
                        return
                }

                response := models.APIResponse{
                        Success: true,
                        Data:    nil,
                        Message: "Institution deleted permanently",
                        Count:   0,
                }

                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                json.NewEncoder(w).Encode(response)
        } else {
                // Soft delete - set aktif to false
                updateDoc := bson.M{
                        "aktif":      false,
                        "updated_at": time.Now().Format("2006-01-02 15:04:05"),
                }

                result, err := collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": updateDoc})
                if err != nil {
                        utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to deactivate institution: "+err.Error())
                        return
                }

                if result.MatchedCount == 0 {
                        utils.SendErrorResponse(w, http.StatusNotFound, "Institution not found")
                        return
                }

                response := models.APIResponse{
                        Success: true,
                        Data:    nil,
                        Message: "Institution deactivated successfully",
                        Count:   0,
                }

                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                json.NewEncoder(w).Encode(response)
        }
}

// GetInstitutionsForSelect returns simplified institution list for dropdown/select components
func GetInstitutionsForSelect(w http.ResponseWriter, r *http.Request) {
        if r.Method == "OPTIONS" {
                utils.HandleCORS(w, r)
                return
        }

        if r.Method != http.MethodGet {
                utils.SendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
                return
        }

        utils.HandleCORS(w, r)

        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        collection := config.GetInstitutionsCollection(MongoClient)

        // Only get active institutions
        filter := bson.M{"aktif": true}

        findOptions := options.Find()
        findOptions.SetSort(bson.D{primitive.E{Key: "kurum_adi", Value: 1}}) // Sort alphabetically
        findOptions.SetProjection(bson.M{
                "_id":        1,
                "kurum_adi":  1,
                "kurum_logo": 1,
                "aktif":      1,
        })

        cursor, err := collection.Find(ctx, filter, findOptions)
        if err != nil {
                utils.SendErrorResponse(w, http.StatusInternalServerError, "Failed to fetch institutions: "+err.Error())
                return
        }
        defer cursor.Close(ctx)

        var institutions []models.InstitutionListResponse
        for cursor.Next(ctx) {
                var institution models.InstitutionModel
                if err := cursor.Decode(&institution); err != nil {
                        continue
                }

                institutions = append(institutions, models.InstitutionListResponse{
                        ID:        institution.ID.Hex(),
                        KurumAdi:  institution.KurumAdi,
                        KurumLogo: institution.KurumLogo,
                        Aktif:     institution.Aktif,
                })
        }

        response := models.APIResponse{
                Success: true,
                Data:    institutions,
                Message: "Institutions for selection fetched successfully",
                Count:   len(institutions),
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
}