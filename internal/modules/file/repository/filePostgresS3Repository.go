package repository

import (
	"context"
	"lorem-backend/internal/database"
	"mime/multipart"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type filePostgresS3Repository struct {
	db database.Database
	s3 ObjectStorage
}

func NewFileMetaPostgresRepository(db database.Database, s3 ObjectStorage) FileRepository {
	return &filePostgresS3Repository{
		db: db,
		s3: s3,
	}
}

func (f *filePostgresS3Repository) CreateFileMeta(ctx context.Context, fileMeta *database.File) (uuid.UUID, error) {
	result := gorm.WithResult()

	err := gorm.G[database.File](f.db.GetDb(), result).Create(ctx, fileMeta)
	if err != nil {
		return uuid.Nil, err
	}

	return fileMeta.ID, nil
}

func (f *filePostgresS3Repository) GetFileMetaByID(ctx context.Context, fileID uuid.UUID) (*database.File, error) {
	fileMeta, err := gorm.G[database.File](f.db.GetDb()).
		Where("id = ?", fileID).
		First(ctx)

	if err != nil {
		return nil, err
	}

	return &fileMeta, nil
}

func (f *filePostgresS3Repository) GetAllFilesMetadata(ctx context.Context, page int64, pageSize int64) ([]database.File, int64, error) {
	var filesMeta []database.File
	var total int64
	db := f.db.GetDb()

	// Count total records
	if err := db.Model(&database.File{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Calculate Offset
	offset := (page - 1) * pageSize

	// Fetch paginated data
	err := db.WithContext(ctx).
		Limit(int(pageSize)).
		Offset(int(offset)).
		Order("created_at DESC"). // Ordered by DESC by default
		Find(&filesMeta).Error

	if err != nil {
		return nil, 0, err
	}

	return filesMeta, total, nil
}

func (f *filePostgresS3Repository) UploadFile(ctx context.Context, objKey string, file multipart.File, size int64, contentType string) (string, error) {
	return f.s3.UploadFile(ctx, objKey, file, size, contentType)
}

func (f *filePostgresS3Repository) GeneratePresignUrl(ctx context.Context, objKey string) (string, error) {
	return f.s3.GeneratePresignUrl(ctx, objKey)
}
