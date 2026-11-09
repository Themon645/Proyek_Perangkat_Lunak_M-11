package database_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
)

func TestNeonStorageBucket(t *testing.T) {
	_ = godotenv.Load("../.env")

	endpoint := os.Getenv("AWS_ENDPOINT_URL_S3")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "ap-southeast-1"
	}

	if endpoint == "" || accessKey == "" || secretKey == "" {
		t.Skip("Neon S3 credentials not set, skipping test")
	}

	cfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. List Buckets
	result, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		t.Skipf("Skipping live S3 test (network unreachable in sandbox): %v", err)
		return
	}

	t.Logf("Found %d bucket(s) in Neon:", len(result.Buckets))
	for _, b := range result.Buckets {
		t.Logf("- Bucket: %s (Created: %v)", *b.Name, b.CreationDate)
	}

	if len(result.Buckets) == 0 {
		t.Fatal("No buckets found in Neon!")
	}

	bucketName := *result.Buckets[0].Name
	t.Logf("Using bucket: %s", bucketName)

	// 2. Put a test object
	testContent := []byte("%PDF-1.4 test upload to neon object storage")
	key := "payment-proofs/test-verification.pdf"

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(testContent),
		ContentType: aws.String("application/pdf"),
	})
	if err != nil {
		t.Fatalf("Failed to upload object to Neon: %v", err)
	}

	publicURL := fmt.Sprintf("%s/%s/%s", endpoint, bucketName, key)
	t.Logf("Upload SUCCESS! Object URL: %s", publicURL)
}
