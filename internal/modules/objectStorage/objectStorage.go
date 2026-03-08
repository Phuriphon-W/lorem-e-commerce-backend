package objectstorage

import (
	"context"
	"mime/multipart"
)

type ObjectStorage interface {
	UploadFile(ctx context.Context, prefixKey string, file multipart.File, size int64, contentType, fileName string) (string, error)
	GeneratePresignUrl(ctx context.Context, objKey string) (string, error)
}
