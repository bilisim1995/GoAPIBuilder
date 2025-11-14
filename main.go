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
        "legal-documents-api/utils"
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

        // Load kurumlar data into cache
        if err := utils.LoadKurumlarToCache(mongoClient); err != nil {
                log.Printf("Warning: Failed to load kurumlar cache: %v", err)
        }

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

        // Sitemap endpoints
        api.HandleFunc("/sitemap/institutions", handlers.GetSitemapInstitutions).Methods("GET", "OPTIONS")
        api.HandleFunc("/sitemap/documents", handlers.GetSitemapDocumentsByInstitution).Methods("GET", "OPTIONS")
        api.HandleFunc("/sitemap/all-documents", handlers.GetSitemapAllDocuments).Methods("GET", "OPTIONS")
        
        // XML Sitemap endpoint
        router.HandleFunc("/sitemap.xml", handlers.GetSitemapXML).Methods("GET", "OPTIONS")

        // Search endpoints
        api.HandleFunc("/search", handlers.GlobalSearch).Methods("GET", "OPTIONS")
        api.HandleFunc("/autocomplete", handlers.Autocomplete).Methods("GET", "OPTIONS")

        // Kurum duyuru endpoint
        api.HandleFunc("/kurum-duyuru", handlers.GetKurumDuyuru).Methods("GET", "OPTIONS")
        
        // Links endpoint
        api.HandleFunc("/links", handlers.GetLinks).Methods("GET", "OPTIONS")
        
        // Cookie management endpoints
        api.HandleFunc("/clear-cookies", handlers.ClearCookies).Methods("POST", "OPTIONS")
        api.HandleFunc("/clear-cookie", handlers.ClearSpecificCookie).Methods("POST", "OPTIONS")
        
        // Recent regulations endpoint
        api.HandleFunc("/regulations/recent", handlers.GetRecentRegulations).Methods("GET", "OPTIONS")

        // Statistics endpoint
        api.HandleFunc("/statistics", handlers.GetStatistics).Methods("GET", "OPTIONS")

        // Health check endpoint
        api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                fmt.Fprint(w, `{"status":"healthy","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)
        }).Methods("GET", "OPTIONS")
        
        

        // Root endpoint - API documentation (with basic authentication)
        router.HandleFunc("/", middleware.BasicAuth(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                response := `{
  "message": "Legal Documents API",
  "version": "1.0.0",
  "endpoints": {
    "/api/v1/institutions": "GET - List all unique institutions",
    "/api/v1/documents?kurum_adi={name}": "GET - Get documents by institution (query param)",
    "/api/v1/kurum/{kurum_slug}": "GET - Get documents by institution (URL path)",
    "/api/v1/documents/{slug}": "GET - Get document details with content",
    "/api/v1/sitemap/institutions": "GET - Sitemap: All institutions",
    "/api/v1/sitemap/documents?kurum_id={id}": "GET - Sitemap: Documents by institution",
    "/api/v1/sitemap/all-documents": "GET - Sitemap: All documents",
    "/api/v1/search?q={query}&limit={limit}&offset={offset}&kurum={institution}&kurum_id={id}": "GET - Global search in titles, content, tags, institutions",
    "/api/v1/autocomplete?q={partial_query}&limit={limit}&kurum={institution}": "GET - Autocomplete suggestions for search",
    "/api/v1/statistics": "GET - Get statistics (total institutions, total documents, document types)",
    "/api/v1/health": "GET - Health check"
  },
  "database": "Connected to MongoDB Atlas",
  "timestamp": "` + time.Now().UTC().Format(time.RFC3339) + `",
  "auth": "Basic authentication required for this page only"
}`
                fmt.Fprint(w, response)
        })).Methods("GET", "OPTIONS")

        return router
}
