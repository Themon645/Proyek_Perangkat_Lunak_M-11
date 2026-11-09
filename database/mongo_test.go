package database

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
)

func TestGetMongoClientWithoutURI(t *testing.T) {
	// Reset singleton for testing
	mongoClientInstance = nil
	mongoOnce = sync.Once{}
	mongoInitErr = nil

	t.Setenv("MONGODB_URI", "")

	client, err := GetMongoClient()
	if err == nil {
		t.Fatal("expected error when MONGODB_URI is not set, got nil")
	}
	if client != nil {
		t.Fatal("expected nil client when MONGODB_URI is not set")
	}
}

func TestLiveMongoConnection(t *testing.T) {
	_ = godotenv.Load("../.env")

	mongoClientInstance = nil
	mongoOnce = sync.Once{}
	mongoInitErr = nil

	client, err := GetMongoClient()
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB Atlas: %v", err)
	}

	db, err := GetMongoDatabase()
	if err != nil {
		t.Fatalf("Failed to get database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collections, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		t.Fatalf("Failed to list collections: %v", err)
	}

	t.Logf("Successfully connected to MongoDB Atlas! Database: %s, Client: %p, Collections: %v", db.Name(), client, collections)
}
