package helper

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
)

const MaxPaymentProofSize int64 = 5 << 20

var allowedProofTypes = map[string]string{
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
}

// SavePaymentProof mengunggah bukti pembayaran langsung ke Neon S3 Object Storage.
func SavePaymentProof(file multipart.File, header *multipart.FileHeader) (string, error) {
	_ = godotenv.Load()
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}
	contentType := strings.Split(detectContentType(buffer[:n]), ";")[0]
	extension, ok := allowedProofTypes[contentType]
	if !ok {
		return "", errors.New("format bukti pembayaran harus PDF, JPG, atau PNG")
	}
	if header.Size > MaxPaymentProofSize {
		return "", errors.New("ukuran bukti pembayaran maksimal 5 MB")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	randomName := make([]byte, 16)
	if _, err := rand.Read(randomName); err != nil {
		return "", err
	}
	fileName := fmt.Sprintf("%x%s", randomName, extension)

	// Cek konfigurasi Neon Object Storage (S3)
	endpoint := os.Getenv("AWS_ENDPOINT_URL_S3")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("AWS_S3_BUCKET")
	if bucketName == "" {
		bucketName = "payment-proofs"
	}

	if endpoint == "" || accessKey == "" || secretKey == "" {
		return "", errors.New("kredensial Neon S3 Object Storage belum dikonfigurasi di environment (.env)")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "ap-southeast-1"
	}

	cfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	key := fileName
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(fileBytes),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("gagal mengunggah bukti pembayaran ke Neon S3: %w", err)
	}

	// Kembalikan URL publik langsung dari Neon Object Storage
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(endpoint, "/"), bucketName, key), nil
}

// DeletePaymentProof menghapus file bukti pembayaran dari Neon S3.
func DeletePaymentProof(proofPath string) {
	if strings.TrimSpace(proofPath) == "" {
		return
	}

	endpoint := os.Getenv("AWS_ENDPOINT_URL_S3")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("AWS_S3_BUCKET")
	if bucketName == "" {
		bucketName = "payment-proofs"
	}

	if endpoint != "" && accessKey != "" && secretKey != "" {
		region := os.Getenv("AWS_REGION")
		if region == "" {
			region = "ap-southeast-1"
		}
		cfg := aws.Config{
			Region:      region,
			Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		}
		client := s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		})

		key := filepath.Base(proofPath)
		if strings.Contains(proofPath, bucketName+"/") {
			parts := strings.Split(proofPath, bucketName+"/")
			if len(parts) == 2 {
				key = parts[1]
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(key),
		})
	}
}

func detectContentType(data []byte) string {
	return http.DetectContentType(data)
}

// GetBaseURL mengembalikan alamat dasar server API yang aktif (misal http://localhost:8080 atau URL Vercel).
func GetBaseURL() string {
	_ = godotenv.Load()
	if base := os.Getenv("BASE_URL"); base != "" {
		return strings.TrimRight(base, "/")
	}
	if vercel := os.Getenv("VERCEL_URL"); vercel != "" {
		if !strings.HasPrefix(vercel, "http://") && !strings.HasPrefix(vercel, "https://") {
			return "https://" + strings.TrimRight(vercel, "/")
		}
		return strings.TrimRight(vercel, "/")
	}
	if appURL := os.Getenv("APP_URL"); appURL != "" {
		return strings.TrimRight(appURL, "/")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("APP_PORT")
	}
	if port == "" {
		port = "8080"
	}
	return "http://localhost:" + port
}

// ResolveProofURLs memformat berkas bukti pembayaran menjadi tautan cloud langsung dari Neon Object Storage.
func ResolveProofURLs(proof string) (proofURL string, attachmentURL string, cloudURL string) {
	_ = godotenv.Load()
	cleanProof := strings.TrimSpace(proof)
	if cleanProof == "" {
		return "", "", ""
	}

	cleanFilename := filepath.Base(cleanProof)

	// Tautan cloud Neon S3
	s3Endpoint := os.Getenv("AWS_ENDPOINT_URL_S3")
	if s3Endpoint == "" {
		s3Endpoint = "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech"
	}
	s3Bucket := os.Getenv("AWS_S3_BUCKET")
	if s3Bucket == "" {
		s3Bucket = "payment-proofs"
	}

	if strings.HasPrefix(cleanProof, "http://") || strings.HasPrefix(cleanProof, "https://") {
		cloudURL = cleanProof
	} else {
		cloudURL = fmt.Sprintf("%s/%s/%s", strings.TrimRight(s3Endpoint, "/"), s3Bucket, cleanFilename)
	}

	proofURL = cloudURL
	attachmentURL = cloudURL
	return proofURL, attachmentURL, cloudURL
}
