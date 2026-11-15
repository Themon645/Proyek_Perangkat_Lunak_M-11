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

func TestResolveProofURLs(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCloud string
	}{
		{
			name:          "Empty proof",
			input:         "",
			expectedCloud: "",
		},
		{
			name:          "Full S3 URL",
			input:         "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample.pdf",
			expectedCloud: "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/sample.pdf",
		},
		{
			name:          "Relative filename",
			input:         "storage/payment-proofs/3f8b9c042cbeecb57f69983b0efb3829.png",
			expectedCloud: "https://br-steep-truth-aoy56eud.storage.c-2.ap-southeast-1.aws.neon.tech/payment-proofs/3f8b9c042cbeecb57f69983b0efb3829.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proofURL, attachmentURL, cloudURL := helper.ResolveProofURLs(tt.input)
			if cloudURL != tt.expectedCloud {
				t.Errorf("cloudURL: expected %q, got %q", tt.expectedCloud, cloudURL)
			}
			if proofURL != tt.expectedCloud {
				t.Errorf("proofURL: expected %q, got %q", tt.expectedCloud, proofURL)
			}
			if attachmentURL != tt.expectedCloud {
				t.Errorf("attachmentURL: expected %q, got %q", tt.expectedCloud, attachmentURL)
			}
		})
	}
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

