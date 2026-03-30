package repository

import (
	"context"
	"mime/multipart"
)

type ObjectStorage interface {
	UploadFile(ctx context.Context, objKey string, file multipart.File, size int64, contentType string) (string, error)
	GeneratePresignUrl(ctx context.Context, objKey string) (string, error)
}
