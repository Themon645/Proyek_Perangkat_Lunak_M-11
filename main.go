package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"m11/common/helper"
	"m11/database"
	"m11/routes"
)

func main() {
	_ = godotenv.Load()

	// Vercel memberikan environment variable PORT saat menjalankan server
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("APP_PORT")
	}
	if port == "" {
		port = "8080"
	}

	// 1. Inisialisasi koneksi pool singleton MongoDB jika MONGODB_URI dikonfigurasi
	if os.Getenv("MONGODB_URI") != "" {
		if _, err := database.GetMongoClient(); err != nil {
			helper.LogError("MongoDB init error: %v", err)
		} else {
			helper.LogInfo("MongoDB connected successfully")
		}
	}

	// Gunakan router terpusat yang menangani root (/), health (/health), dan endpoints lainnya
	handler := routes.Register(nil)

	helper.LogInfo("Server running on port :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

