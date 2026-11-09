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
)

const MaxPaymentProofSize int64 = 5 << 20

var allowedProofTypes = map[string]string{
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
}

// SavePaymentProof mengunggah bukti pembayaran langsung ke Neon S3 Object Storage jika dikonfigurasi,
// atau menyimpan ke direktori lokal (dengan fallback otomatis /tmp di Vercel serverless).
func SavePaymentProof(file multipart.File, header *multipart.FileHeader) (string, error) {
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

	// 1. Cek apakah Neon Object Storage (S3) dikonfigurasi
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

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			return "", err
		}

		key := "payment-proofs/" + fileName
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		_, err = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(bucketName),
			Key:         aws.String(key),
			Body:        bytes.NewReader(fileBytes),
			ContentType: aws.String(contentType),
		})
		if err == nil {
			// Kembalikan URL publik langsung dari Neon Object Storage
			return fmt.Sprintf("%s/%s/%s", strings.TrimRight(endpoint, "/"), bucketName, key), nil
		}
		LogError("Neon S3 upload failed, falling back to local/tmp storage: %v", err)
	}

	// 2. Fallback: simpan di direktori lokal atau /tmp
	targetDir := GetProofStorageDir()
	fullPath := filepath.Join(targetDir, fileName)

	destination, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return "", err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, file); err != nil {
		_ = os.Remove(fullPath)
		return "", err
	}

	return filepath.ToSlash(filepath.Join("storage", "payment-proofs", fileName)), nil
}

// DeletePaymentProof menghapus file bukti pembayaran dari Neon S3 atau storage lokal
func DeletePaymentProof(proofPath string) {
	if strings.TrimSpace(proofPath) == "" {
		return
	}

	// Jika merupakan URL Neon S3
	endpoint := os.Getenv("AWS_ENDPOINT_URL_S3")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("AWS_S3_BUCKET")
	if bucketName == "" {
		bucketName = "payment-proofs"
	}

	if endpoint != "" && accessKey != "" && secretKey != "" && strings.HasPrefix(proofPath, "http") {
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

		parts := strings.Split(proofPath, bucketName+"/")
		if len(parts) == 2 {
			key := parts[1]
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(key),
			})
		}
	}

	_ = os.Remove(proofPath)
	_ = os.Remove(filepath.Join(os.TempDir(), proofPath))
	_ = os.Remove(filepath.Join(os.TempDir(), "storage", "payment-proofs", filepath.Base(proofPath)))
	_ = os.Remove(filepath.Join(os.TempDir(), filepath.Base(proofPath)))
}

func GetProofStorageDir() string {
	primary := filepath.Join("storage", "payment-proofs")
	if err := os.MkdirAll(primary, 0o755); err == nil {
		return primary
	}

	fallback := filepath.Join(os.TempDir(), "storage", "payment-proofs")
	if err := os.MkdirAll(fallback, 0o755); err == nil {
		return fallback
	}

	return os.TempDir()
}

func detectContentType(data []byte) string {
	return http.DetectContentType(data)
}
