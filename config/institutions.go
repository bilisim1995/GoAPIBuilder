package config

import (
        "os"
        "go.mongodb.org/mongo-driver/mongo"
)

// GetInstitutionsCollection returns the institutions collection
func GetInstitutionsCollection(client *mongo.Client) *mongo.Collection {
        database := os.Getenv("MONGODB_DATABASE")
        if database == "" {
                database = "mevzuatgpt"
        }
        return client.Database(database).Collection("institutions")
}