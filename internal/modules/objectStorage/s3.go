package objectstorage

import (
	"context"
	"fmt"
	"lorem-backend/internal/config"
	"mime/multipart"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Repository struct {
	Client        *s3.Client
	PresignClient *s3.PresignClient
	BucketName    string
}

func NewS3Repository(s3Client *s3.Client, cfg *config.Config) ObjectStorage {
	presignClient := s3.NewPresignClient(s3Client)

	return &s3Repository{
		Client:        s3Client,
		PresignClient: presignClient,
		BucketName:    cfg.BucketName,
	}
}

func (s *s3Repository) UploadFile(ctx context.Context, prefixKey string, file multipart.File, size int64, contentType, fileName string) (string, error) {
	imageKey := fmt.Sprintf("%v/%v-%v", prefixKey, time.Now().Unix(), fileName)

	_, err := s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.BucketName),
		Key:           aws.String(imageKey),
		Body:          file,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})

	if err != nil {
		return "", err
	}

	return imageKey, nil
}

func (s *s3Repository) GeneratePresignUrl(ctx context.Context, objKey string) (string, error) {
	expiresIn := 15 * time.Minute

	req, err := s.PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(objKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiresIn
	})

	if err != nil {
		return "", err
	}

	return req.URL, nil
}
