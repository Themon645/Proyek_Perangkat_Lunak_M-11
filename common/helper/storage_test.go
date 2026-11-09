package helper_test

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"os"
	"testing"

	"m11/common/helper"
)

func createMultipartFile(filename, contentType string, content []byte) (multipart.File, *multipart.FileHeader, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="proof"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)

	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, nil, err
	}
	if _, err := part.Write(content); err != nil {
		return nil, nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, nil, err
	}

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(int64(len(content) + 1024))
	if err != nil {
		return nil, nil, err
	}

	files := form.File["proof"]
	if len(files) == 0 {
		return nil, nil, os.ErrNotExist
	}
	file, err := files[0].Open()
	return file, files[0], err
}

func TestGetProofStorageDir(t *testing.T) {
	dir := helper.GetProofStorageDir()
	if dir == "" {
		t.Fatal("expected non-empty storage dir")
	}
	// Verify directory exists and is writable
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("expected dir to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected a directory, got %s", dir)
	}
}

func TestSavePaymentProof_PDF(t *testing.T) {
	// Standard PDF header signature
	pdfContent := []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\ntrailer\n<<>>\n%%EOF")
	file, header, err := createMultipartFile("receipt.pdf", "application/pdf", pdfContent)
	if err != nil {
		t.Fatalf("createMultipartFile error: %v", err)
	}
	defer file.Close()

	path, err := helper.SavePaymentProof(file, header)
	if err != nil {
		t.Fatalf("SavePaymentProof returned error: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty saved path")
	}

	// Cleanup
	helper.DeletePaymentProof(path)
}

func TestSavePaymentProof_InvalidType(t *testing.T) {
	txtContent := []byte("plain text file content")
	file, header, err := createMultipartFile("notes.txt", "text/plain", txtContent)
	if err != nil {
		t.Fatalf("createMultipartFile error: %v", err)
	}
	defer file.Close()

	_, err = helper.SavePaymentProof(file, header)
	if err == nil {
		t.Fatal("expected error for invalid content type, got nil")
	}
}
