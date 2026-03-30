package handler

import (
	"context"
	"lorem-backend/internal/modules/file/dto"
)

type FileHandler interface {
	UploadFile(ctx context.Context, input *dto.UploadFileInputDto) (*dto.UploadFileOutputDto, error)
	DownLoadFile(ctx context.Context, input *dto.DownloadFileInputDto) (*dto.DownloadFileOutputDto, error)
	GetFileMetaByID(ctx context.Context, input *dto.GetFileMetaByIDInputDto) (*dto.GetFileMetaByIDOutputDto, error)
	GetAllFilesMetadata(ctx context.Context, input *dto.GetAllFilesMetadataInputDto) (*dto.GetAllFilesMetadataOutputDto, error)
}
