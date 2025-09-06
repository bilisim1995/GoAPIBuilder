package config

import (
	"go.mongodb.org/mongo-driver/mongo"
)

// GetInstitutionsCollection returns the institutions collection
func GetInstitutionsCollection(client *mongo.Client) *mongo.Collection {
	return client.Database(MongoDatabase).Collection("institutions")
}