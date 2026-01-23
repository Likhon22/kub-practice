package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Item represents a simple data model
type Item struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

type Response struct {
	Service string `json:"service"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

var collection *mongo.Collection

// CORS middleware
func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func connectMongo(mongoURL string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mongoURL := os.Getenv("MONGO_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://localhost:27017"
	}

	// Connect to MongoDB
	client, err := connectMongo(mongoURL)
	if err != nil {
		log.Printf("Warning: MongoDB connection failed: %v", err)
		log.Println("Running without database - create/get endpoints will fail")
	} else {
		collection = client.Database("backend_db").Collection("items")
		log.Println("Connected to MongoDB successfully!")
	}

	// Basic API endpoint
	http.HandleFunc("/api", cors(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Service: "backend",
			Message: "Hello from Backend API!",
			Status:  "ok",
		})
	}))

	// Health check
	http.HandleFunc("/api/health", cors(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		dbStatus := "disconnected"
		if collection != nil {
			dbStatus = "connected"
		}
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "healthy",
			"service":   "backend",
			"mongo_url": mongoURL,
			"db_status": dbStatus,
		})
	}))

	// CREATE - POST /api/items
	http.HandleFunc("/api/items/create", cors(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed, use POST"})
			return
		}

		if collection == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database not connected"})
			return
		}

		var input struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
			return
		}

		item := Item{
			Name:      input.Name,
			CreatedAt: time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := collection.InsertOne(ctx, item)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		item.ID = result.InsertedID.(primitive.ObjectID)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Item created successfully",
			"item":    item,
		})
	}))

	// GET ALL - GET /api/items
	http.HandleFunc("/api/items", cors(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if collection == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database not connected"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cursor, err := collection.Find(ctx, bson.M{})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var items []Item
		if err := cursor.All(ctx, &items); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		if items == nil {
			items = []Item{}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"count": len(items),
			"items": items,
		})
	}))

	fmt.Printf("Backend server running on port %s\n", port)
	fmt.Printf("MongoDB URL: %s\n", mongoURL)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
