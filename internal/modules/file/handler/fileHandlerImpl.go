package handler

import (
	"context"
	"fmt"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/file/dto"
	"lorem-backend/internal/modules/file/repository"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type fileHandlerImpl struct {
	fileRepo repository.FileRepository
}

func NewFileHandlerImpl(fileRepo repository.FileRepository) FileHandler {
	return &fileHandlerImpl{
		fileRepo: fileRepo,
	}
}

func (f *fileHandlerImpl) UploadFile(ctx context.Context, input *dto.UploadFileInputDto) (*dto.UploadFileOutputDto, error) {
	formData := input.RawBody.Data()
	file := formData.File
	objKey := formData.ObjectBaseKey

	putKey := fmt.Sprintf("%v/%v-%v", objKey, time.Now().Unix(), file.Filename)

	// Upload to S3
	key, err := f.fileRepo.UploadFile(ctx, putKey, file, file.Size, file.ContentType)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error uploading a file", err)
	}

	// Store File Metadata to database
	fileMeta := &database.File{
		OriginalName: file.Filename,
		Name:         uuid.New().String(),
		Size:         file.Size,
		ContentType:  file.ContentType,
		ObjectKey:    key,
	}
	fileId, err := f.fileRepo.CreateFileMeta(ctx, fileMeta)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error generating file metadata", err)
	}

	res := &dto.UploadFileOutputDto{
		Body: dto.UploadFileOutputDtoBody{
			FileID: fileId.String(),
		},
	}
	return res, nil
}

func (f *fileHandlerImpl) UploadStaticFile(ctx context.Context, input *dto.UploadStaticFileInputDto) (*dto.UploadStaticFileOutputDto, error) {
	formData := input.RawBody.Data()
	file := formData.File
	objKey := formData.ObjectBaseKey

	// objKey/fileName
	putKey := fmt.Sprintf("%v/%v", objKey, formData.FileName)

	// Upload to S3
	key, err := f.fileRepo.UploadFile(ctx, putKey, file, file.Size, file.ContentType)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error uploading a file", err)
	}

	// Store File Metadata to database
	fileMeta := &database.File{
		OriginalName: file.Filename,
		Name:         uuid.New().String(),
		Size:         file.Size,
		ContentType:  file.ContentType,
		ObjectKey:    key,
	}
	fileId, err := f.fileRepo.CreateFileMeta(ctx, fileMeta)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error generating file metadata", err)
	}

	res := &dto.UploadStaticFileOutputDto{
		Body: dto.UploadStaticFileOutputDtoBody{
			FileID:    fileId.String(),
			ObjectKey: putKey,
		},
	}
	return res, nil
}

func (f *fileHandlerImpl) DownLoadFile(ctx context.Context, input *dto.DownloadFileInputDto) (*dto.DownloadFileOutputDto, error) {
	fileMeta, err := f.fileRepo.GetFileMetaByID(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Error getting file metadata", err)
	}

	url, err := f.fileRepo.GeneratePresignUrl(ctx, fileMeta.ObjectKey)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error generating download url", err)
	}

	res := &dto.DownloadFileOutputDto{
		Body: dto.DownloadFileOutputDtoBody{
			FileName:    fileMeta.Name,
			DownloadURL: url,
		},
	}
	return res, nil
}

func (f *fileHandlerImpl) DownloadFileByKey(ctx context.Context, input *dto.DownloadFileByKeyInputDto) (*dto.DownloadFileByKeyOutputDto, error) {
	url, err := f.fileRepo.GeneratePresignUrl(ctx, input.ObjectKey)
	if err != nil {
		return nil, huma.Error404NotFound("The file with provided key does not exist", err)
	}

	res := &dto.DownloadFileByKeyOutputDto{
		Body: dto.DownloadFileByKeyOutputDtoBody{
			DownloadURL: url,
		},
	}
	return res, nil
}

func (f *fileHandlerImpl) GetFileMetaByID(ctx context.Context, input *dto.GetFileMetaByIDInputDto) (*dto.GetFileMetaByIDOutputDto, error) {
	fileMeta, err := f.fileRepo.GetFileMetaByID(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Error getting file metadata", err)
	}

	res := &dto.GetFileMetaByIDOutputDto{
		Body: dto.GetFileMetaByIDOutputDtoBody{
			ID:          fileMeta.ID,
			Name:        fileMeta.OriginalName,
			Size:        uint(fileMeta.Size),
			ContentType: fileMeta.ContentType,
			ObjectKey:   fileMeta.ObjectKey,
		},
	}

	return res, nil
}

func (f *fileHandlerImpl) GetAllFilesMetadata(ctx context.Context, input *dto.GetAllFilesMetadataInputDto) (*dto.GetAllFilesMetadataOutputDto, error) {
	filesMeta, total, err := f.fileRepo.GetAllFilesMetadata(ctx, input.PageNumber, input.PageSize)
	if err != nil {
		return nil, huma.Error404NotFound("Failed to retrieve files metadata", err)
	}

	results := make([]dto.GetFileMetaByIDOutputDtoBody, len(filesMeta))
	for i, f := range filesMeta {
		results[i] = dto.GetFileMetaByIDOutputDtoBody{
			ID:          f.ID,
			Name:        f.Name,
			Size:        uint(f.Size),
			ContentType: f.ContentType,
			ObjectKey:   f.ObjectKey,
		}
	}

	res := &dto.GetAllFilesMetadataOutputDto{
		Body: dto.GetAllFilesMetadataOutputDtoBody{
			FilesMetadata: results,
			Total:         total,
		},
	}

	return res, nil
}
