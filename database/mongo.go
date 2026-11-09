package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	mongoClientInstance *mongo.Client
	mongoOnce           sync.Once
	mongoInitErr        error
)

// GetMongoClient mengembalikan instance singleton dari MongoDB Client.
// Menggunakan sync.Once untuk memastikan inisialisasi koneksi pool hanya dilakukan satu kali
// pada runtime serverless (mencegah connection leak / pool exhaustion di MongoDB Atlas).
func GetMongoClient() (*mongo.Client, error) {
	mongoOnce.Do(func() {
		uri := os.Getenv("MONGODB_URI")
		if uri == "" {
			mongoInitErr = errors.New("environment variable MONGODB_URI belum diatur")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// MaxPoolSize dibatasi agar ramah serverless (default 10 koneksi per instance)
		maxPoolSize := uint64(10)
		if maxPoolStr := os.Getenv("MONGODB_MAX_POOL_SIZE"); maxPoolStr != "" {
			if parsed, err := strconv.ParseUint(maxPoolStr, 10, 64); err == nil && parsed > 0 {
				maxPoolSize = parsed
			}
		}

		// MinPoolSize diset 0 agar tidak mempertahankan idle connection saat serverless cold/idle
		minPoolSize := uint64(0)
		if minPoolStr := os.Getenv("MONGODB_MIN_POOL_SIZE"); minPoolStr != "" {
			if parsed, err := strconv.ParseUint(minPoolStr, 10, 64); err == nil {
				minPoolSize = parsed
			}
		}

		clientOptions := options.Client().
			ApplyURI(uri).
			SetMaxPoolSize(maxPoolSize).
			SetMinPoolSize(minPoolSize).
			SetMaxConnIdleTime(30 * time.Second).
			SetServerSelectionTimeout(5 * time.Second)

		client, err := mongo.Connect(ctx, clientOptions)
		if err != nil {
			mongoInitErr = fmt.Errorf("gagal menghubungkan ke MongoDB: %w", err)
			return
		}

		// Verifikasi konektivitas ke cluster MongoDB
		if err := client.Ping(ctx, readpref.Primary()); err != nil {
			mongoInitErr = fmt.Errorf("gagal ping MongoDB: %w", err)
			_ = client.Disconnect(ctx)
			return
		}

		mongoClientInstance = client
	})

	if mongoInitErr != nil {
		// Reset sync.Once agar percobaan berikutnya bisa mencoba kembali jika ada kegagalan transient
		mongoOnce = sync.Once{}
		return nil, mongoInitErr
	}

	return mongoClientInstance, nil
}

// GetMongoDatabase mengembalikan database MongoDB yang aktif.
func GetMongoDatabase(name ...string) (*mongo.Database, error) {
	client, err := GetMongoClient()
	if err != nil {
		return nil, err
	}

	dbName := os.Getenv("MONGODB_DATABASE")
	if len(name) > 0 && name[0] != "" {
		dbName = name[0]
	}
	if dbName == "" {
		dbName = "procurement"
	}

	return client.Database(dbName), nil
}

