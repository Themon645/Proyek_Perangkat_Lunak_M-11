package database

import (
	"sync"
	"testing"

	"github.com/joho/godotenv"
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

	// Reset singleton
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

	t.Logf("Successfully connected to MongoDB Atlas! Database: %s, Client: %p", db.Name(), client)
}
