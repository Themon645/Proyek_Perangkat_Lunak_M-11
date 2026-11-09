package helper

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const MaxPaymentProofSize int64 = 5 << 20

var allowedProofTypes = map[string]string{
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
}

// GetProofStorageDir mengembalikan direktori penyimpanan bukti pembayaran yang aman.
// Pada lingkungan Serverless (Vercel / AWS Lambda), filesystem root bersistem read-only,
// sehingga fungsi ini secara otomatis melakukan fallback ke os.TempDir().
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

	// Kembalikan relative path standar yang rapi untuk dicatat di database
	return filepath.ToSlash(filepath.Join("storage", "payment-proofs", fileName)), nil
}

// DeletePaymentProof menghapus file bukti pembayaran dari storage lokal maupun fallback /tmp
func DeletePaymentProof(proofPath string) {
	if strings.TrimSpace(proofPath) == "" {
		return
	}
	_ = os.Remove(proofPath)
	_ = os.Remove(filepath.Join(os.TempDir(), proofPath))
	_ = os.Remove(filepath.Join(os.TempDir(), "storage", "payment-proofs", filepath.Base(proofPath)))
	_ = os.Remove(filepath.Join(os.TempDir(), filepath.Base(proofPath)))
}

func detectContentType(data []byte) string {
	return http.DetectContentType(data)
}

