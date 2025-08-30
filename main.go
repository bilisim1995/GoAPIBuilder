package main

import (
        "context"
        "fmt"
        "log"
        "net/http"
        "os"
        "time"

        "github.com/gorilla/mux"
        "github.com/joho/godotenv"
        "go.mongodb.org/mongo-driver/mongo"

        "legal-documents-api/config"
        "legal-documents-api/handlers"
        "legal-documents-api/middleware"
)

var mongoClient *mongo.Client

func main() {
        // Load environment variables
        if err := godotenv.Load(); err != nil {
                log.Println("Warning: .env file not found, using system environment variables")
        }

        // Initialize MongoDB connection
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        client, err := config.ConnectMongoDB(ctx)
        if err != nil {
                log.Fatal("Failed to connect to MongoDB:", err)
        }
        mongoClient = client

        // Ensure connection is closed on exit
        defer func() {
                if err := mongoClient.Disconnect(context.Background()); err != nil {
                        log.Printf("Error disconnecting from MongoDB: %v", err)
                }
        }()

        // Test connection
        if err := mongoClient.Ping(ctx, nil); err != nil {
                log.Fatal("Failed to ping MongoDB:", err)
        }
        log.Println("Successfully connected to MongoDB Atlas")

        // Initialize handlers with MongoDB client
        handlers.InitHandlers(mongoClient)

        // Setup routes
        router := setupRoutes()

        // Get port from environment or use default
        port := os.Getenv("PORT")
        if port == "" {
                port = "5000"
        }

        log.Printf("Server starting on port %s", port)
        log.Fatal(http.ListenAndServe("0.0.0.0:"+port, router))
}

func setupRoutes() *mux.Router {
        router := mux.NewRouter()

        // Apply CORS middleware
        router.Use(middleware.CORSMiddleware)

        // API routes
        api := router.PathPrefix("/api/v1").Subrouter()

        // Institution endpoints
        api.HandleFunc("/institutions", handlers.GetInstitutions).Methods("GET", "OPTIONS")

        // Document endpoints
        api.HandleFunc("/documents", handlers.GetDocumentsByInstitution).Methods("GET", "OPTIONS")
        api.HandleFunc("/documents/{slug}", handlers.GetDocumentBySlug).Methods("GET", "OPTIONS")
        
        // Institution-based routing (alternative endpoint)
        api.HandleFunc("/kurum/{kurum_slug}", handlers.GetDocumentsByInstitutionSlug).Methods("GET", "OPTIONS")

        // Health check endpoint
        api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                fmt.Fprint(w, `{"status":"healthy","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)
        }).Methods("GET", "OPTIONS")

        // Root endpoint - API documentation
        router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                response := `{
  "message": "Legal Documents API",
  "version": "1.0.0",
  "endpoints": {
    "/api/v1/institutions": "GET - List all unique institutions",
    "/api/v1/documents?kurum_adi={name}": "GET - Get documents by institution",
    "/api/v1/documents/{slug}": "GET - Get document details with content",
    "/api/v1/health": "GET - Health check"
  },
  "database": "Connected to MongoDB Atlas",
  "timestamp": "` + time.Now().UTC().Format(time.RFC3339) + `"
}`
                fmt.Fprint(w, response)
        }).Methods("GET", "OPTIONS")

        return router
}
