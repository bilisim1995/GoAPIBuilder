package config

import (
	"context"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectMongoDB establishes connection to MongoDB Atlas
func ConnectMongoDB(ctx context.Context) (*mongo.Client, error) {
	connectionString := os.Getenv("MONGODB_CONNECTION_STRING")
	if connectionString == "" {
		connectionString = "mongodb://localhost:27017" // fallback for development
	}

	clientOptions := options.Client().ApplyURI(connectionString)
	
	// Set connection pool settings for scalability
	clientOptions.SetMaxPoolSize(100)
	clientOptions.SetMinPoolSize(5)
	clientOptions.SetMaxConnIdleTime(0) // Keep connections alive

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// GetDatabase returns the database instance
func GetDatabase(client *mongo.Client) *mongo.Database {
	dbName := os.Getenv("MONGODB_DATABASE")
	if dbName == "" {
		dbName = "mevzuatgpt" // fallback
	}
	return client.Database(dbName)
}

// GetMetadataCollection returns the metadata collection
func GetMetadataCollection(client *mongo.Client) *mongo.Collection {
	db := GetDatabase(client)
	collectionName := os.Getenv("MONGODB_METADATA_COLLECTION")
	if collectionName == "" {
		collectionName = "metadata" // fallback
	}
	return db.Collection(collectionName)
}

// GetContentCollection returns the content collection
func GetContentCollection(client *mongo.Client) *mongo.Collection {
	db := GetDatabase(client)
	collectionName := os.Getenv("MONGODB_CONTENT_COLLECTION")
	if collectionName == "" {
		collectionName = "content" // fallback
	}
	return db.Collection(collectionName)
}
