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

func SavePaymentProof(file multipart.File, header *multipart.FileHeader) (string, error) {
	if header.Size > MaxPaymentProofSize {
		return "", errors.New("ukuran bukti pembayaran maksimal 5 MB")
	}
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
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Join("storage", "payment-proofs"), 0o750); err != nil {
		return "", err
	}
	randomName := make([]byte, 16)
	if _, err := rand.Read(randomName); err != nil {
		return "", err
	}
	relativePath := filepath.Join("storage", "payment-proofs", fmt.Sprintf("%x%s", randomName, extension))
	destination, err := os.OpenFile(relativePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return "", err
	}
	defer destination.Close()
	if _, err := io.Copy(destination, file); err != nil {
		_ = os.Remove(relativePath)
		return "", err
	}
	return relativePath, nil
}

func detectContentType(data []byte) string {
	return http.DetectContentType(data)
}
