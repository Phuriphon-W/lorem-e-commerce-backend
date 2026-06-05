package repository

import (
	"bytes"
	"context"
	"io"
	"lorem-backend/internal/config"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
)

type mockRoundTripper struct {
	roundTripFn func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFn(req)
}

func TestS3Repository(t *testing.T) {
	config.GlobalConfig = &config.Config{
		BucketName: "test-bucket",
	}

	var mockStatus int
	var mockErr error

	httpClient := &http.Client{
		Transport: &mockRoundTripper{
			roundTripFn: func(req *http.Request) (*http.Response, error) {
				if mockErr != nil {
					return nil, mockErr
				}
				return &http.Response{
					StatusCode: mockStatus,
					Body:       io.NopCloser(bytes.NewReader([]byte(""))),
					Header:     make(http.Header),
				}, nil
			},
		},
	}

	s3Client := s3.New(s3.Options{
		Region:       "us-east-1",
		Credentials:  credentials.StaticCredentialsProvider{Value: aws.Credentials{AccessKeyID: "mock", SecretAccessKey: "mock"}},
		HTTPClient:   httpClient,
		UsePathStyle: true,
	})

	repo := NewS3Repository(s3Client)

	t.Run("UploadFile - Success", func(t *testing.T) {
		mockStatus = 200
		mockErr = nil

		fileContent := []byte("hello s3")
		fileReader := &mockFileReader{Reader: bytes.NewReader(fileContent)}

		key, err := repo.UploadFile(context.Background(), "test-key.txt", fileReader, int64(len(fileContent)), "text/plain")
		assert.NoError(t, err)
		assert.Equal(t, "test-key.txt", key)
	})

	t.Run("UploadFile - Error", func(t *testing.T) {
		mockStatus = 500
		mockErr = nil

		fileContent := []byte("hello s3")
		fileReader := &mockFileReader{Reader: bytes.NewReader(fileContent)}

		_, err := repo.UploadFile(context.Background(), "test-key.txt", fileReader, int64(len(fileContent)), "text/plain")
		assert.Error(t, err)
	})

	t.Run("GeneratePresignUrl - Success", func(t *testing.T) {
		url, err := repo.GeneratePresignUrl(context.Background(), "test-key.txt")
		assert.NoError(t, err)
		assert.Contains(t, url, "test-key.txt")
	})
}
