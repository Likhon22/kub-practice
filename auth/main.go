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

// User represents a simple user model
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username  string             `bson:"username" json:"username"`
	Email     string             `bson:"email" json:"email"`
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

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
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
		collection = client.Database("auth_db").Collection("users")
		log.Println("Connected to MongoDB successfully!")
	}

	// Basic auth endpoint
	http.HandleFunc("/auth", cors(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Service: "auth",
			Message: "Hello from Auth Service!",
			Status:  "ok",
		})
	}))

	// Simulated login
	http.HandleFunc("/auth/login", cors(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"service": "auth",
			"action":  "login",
			"status":  "simulated_success",
			"token":   "fake-jwt-token-12345",
		})
	}))

	// Health check
	http.HandleFunc("/auth/health", cors(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		dbStatus := "disconnected"
		if collection != nil {
			dbStatus = "connected"
		}
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "healthy",
			"service":   "auth",
			"mongo_url": mongoURL,
			"db_status": dbStatus,
		})
	}))

	// CREATE USER - POST /auth/users/create
	http.HandleFunc("/auth/users/create", cors(func(w http.ResponseWriter, r *http.Request) {
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
			Username string `json:"username"`
			Email    string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
			return
		}

		user := User{
			Username:  input.Username,
			Email:     input.Email,
			CreatedAt: time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := collection.InsertOne(ctx, user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		user.ID = result.InsertedID.(primitive.ObjectID)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "User created successfully",
			"user":    user,
		})
	}))

	// GET ALL USERS - GET /auth/users
	http.HandleFunc("/auth/users", cors(func(w http.ResponseWriter, r *http.Request) {
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

		var users []User
		if err := cursor.All(ctx, &users); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		if users == nil {
			users = []User{}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"count": len(users),
			"users": users,
		})
	}))

	fmt.Printf("Auth server running on port %s\n", port)
	fmt.Printf("MongoDB URL: %s\n", mongoURL)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
