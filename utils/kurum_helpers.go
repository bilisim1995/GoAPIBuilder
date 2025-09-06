package utils

import (
        "context"
        "log"
        "sync"

        "go.mongodb.org/mongo-driver/bson"
        "go.mongodb.org/mongo-driver/mongo"
        "legal-documents-api/config"
        "legal-documents-api/models"
)

// KurumCache holds institution data in memory for fast access
type KurumCache struct {
        kurumlar map[string]models.Kurum // kurum_id -> Kurum
        mutex    sync.RWMutex
}

var cache = &KurumCache{
        kurumlar: make(map[string]models.Kurum),
}

// LoadKurumlarToCache loads all institutions into memory cache
func LoadKurumlarToCache(mongoClient *mongo.Client) error {
        ctx := context.Background()
        collection := config.GetKurumlarCollection(mongoClient)
        
        cursor, err := collection.Find(ctx, bson.M{})
        if err != nil {
                return err
        }
        defer cursor.Close(ctx)
        
        var kurumlar []models.Kurum
        if err := cursor.All(ctx, &kurumlar); err != nil {
                return err
        }
        
        cache.mutex.Lock()
        defer cache.mutex.Unlock()
        
        // Clear existing cache and reload
        cache.kurumlar = make(map[string]models.Kurum)
        for _, kurum := range kurumlar {
                cache.kurumlar[kurum.ID.Hex()] = kurum
        }
        
        log.Printf("Loaded %d institutions into cache", len(kurumlar))
        return nil
}

// GetKurumByID returns institution data by kurum_id from cache
func GetKurumByID(kurumID string) (models.Kurum, bool) {
        cache.mutex.RLock()
        defer cache.mutex.RUnlock()
        
        kurum, exists := cache.kurumlar[kurumID]
        return kurum, exists
}

// GetKurumAdiByID returns institution name by kurum_id
func GetKurumAdiByID(kurumID string) string {
        if kurum, exists := GetKurumByID(kurumID); exists {
                return kurum.KurumAdi
        }
        return "Bilinmeyen Kurum" // fallback
}

// GetKurumLogoByID returns institution logo by kurum_id
func GetKurumLogoByID(kurumID string) string {
        if kurum, exists := GetKurumByID(kurumID); exists {
                return kurum.KurumLogo
        }
        return "" // empty logo if not found
}

// GetKurumAciklamaByID returns kurum aciklama from cache by kurum_id
func GetKurumAciklamaByID(kurumID string) string {
        if kurum, exists := GetKurumByID(kurumID); exists {
                return kurum.KurumAciklama
        }
        return ""
}

// GetAllKurumlar returns all institutions from cache
func GetAllKurumlar() []models.Kurum {
        cache.mutex.RLock()
        defer cache.mutex.RUnlock()
        
        kurumlar := make([]models.Kurum, 0, len(cache.kurumlar))
        for _, kurum := range cache.kurumlar {
                kurumlar = append(kurumlar, kurum)
        }
        
        return kurumlar
}

// Debug function - GetCacheStatus returns cache status
func GetCacheStatus() map[string]interface{} {
        cache.mutex.RLock()
        defer cache.mutex.RUnlock()
        
        status := map[string]interface{}{
                "cache_size": len(cache.kurumlar),
                "kurumlar": make([]map[string]string, 0),
        }
        
        for _, kurum := range cache.kurumlar {
                item := map[string]string{
                        "kurum_id": kurum.ID.Hex(),
                        "kurum_adi": kurum.KurumAdi,
                        "kurum_logo": kurum.KurumLogo,
                }
                status["kurumlar"] = append(status["kurumlar"].([]map[string]string), item)
        }
        
        return status
}

// RefreshKurumlarCache reloads the institutions cache
func RefreshKurumlarCache(mongoClient *mongo.Client) error {
        return LoadKurumlarToCache(mongoClient)
}