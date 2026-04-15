package repository

import (
	"context"
	"lorem-backend/internal/database"
	"mime/multipart"

	"github.com/google/uuid"
)

type FileRepository interface {
	CreateFileMeta(ctx context.Context, fileMeta *database.File) (uuid.UUID, error)
	GetFileMetaByID(ctx context.Context, fileID uuid.UUID) (*database.File, error)
	GetAllFilesMetadata(ctx context.Context, page int64, pageSize int64) ([]database.File, int64, error)
	UploadFile(ctx context.Context, objKey string, file multipart.File, size int64, contentType string) (string, error)
	GeneratePresignUrl(ctx context.Context, objKey string) (string, error)
}
