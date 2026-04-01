package dto

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// Upload File
type (
	UploadFileInputDto struct {
		RawBody huma.MultipartFormFiles[struct {
			File          huma.FormFile `form:"file" required:"true" doc:"The file content to upload"`
			ObjectBaseKey string        `form:"objectBaseKey" doc:"Base object key in object storage"`
		}]
	}

	UploadFileOutputDtoBody struct {
		FileID string `json:"fileId" doc:"ID of the Created File"`
	}

	UploadFileOutputDto struct {
		Body UploadFileOutputDtoBody
	}
)

// Upload Static File
type (
	UploadStaticFileInputDto struct {
		RawBody huma.MultipartFormFiles[struct {
			File          huma.FormFile `form:"file" required:"true" doc:"The file content to upload"`
			ObjectBaseKey string        `form:"objectBaseKey" doc:"Base object key in object storage"`
		}]
	}

	UploadStaticFileOutputDtoBody struct {
		FileID    string `json:"fileId" doc:"ID of the Created File"`
		ObjectKey string `json:"objectKey" doc:"Key of the created object in storage"`
	}

	UploadStaticFileOutputDto struct {
		Body UploadStaticFileOutputDtoBody
	}
)

// Download File
type (
	DownloadFileInputDto struct {
		ID uuid.UUID `path:"id" required:"true" doc:"ID of an Object in Storage"`
	}

	DownloadFileOutputDtoBody struct {
		FileName    string `json:"fileName" doc:"Name of the file"`
		DownloadURL string `json:"downloadUrl" doc:"Presign URL for Accessing the Object"`
	}

	DownloadFileOutputDto struct {
		Body DownloadFileOutputDtoBody
	}
)

type (
	DownloadFileByKeyInputDto struct {
		ObjectKey string `path:"key" required:"true" doc:"Key of an Object in Storage"`
	}

	DownloadFileByKeyOutputDtoBody struct {
		DownloadURL string `json:"downloadUrl" doc:"Presign URL for Accessing the Object"`
	}

	DownloadFileByKeyOutputDto struct {
		Body DownloadFileByKeyOutputDtoBody
	}
)

// Get File Metadata By ID
type (
	GetFileMetaByIDInputDto struct {
		ID uuid.UUID `path:"id" required:"true" doc:"ID of an Object Record in database"`
	}

	GetFileMetaByIDOutputDtoBody struct {
		ID          uuid.UUID `json:"fileId" doc:"ID of an Object"`
		Name        string    `json:"name" doc:"Name of the file"`
		Size        uint      `json:"size" doc:"Size of the file"`
		ContentType string    `json:"contentType" doc:"Content Type of the file"`
		ObjectKey   string    `json:"objKey" doc:"Key of the Object in Storage"`
	}

	GetFileMetaByIDOutputDto struct {
		Body GetFileMetaByIDOutputDtoBody
	}
)

// Get All Files Metadata
type (
	GetAllFilesMetadataInputDto struct {
		PageNumber uint64 `query:"pageNumber" default:"1" minimum:"1" doc:"Page number"`
		PageSize   uint64 `query:"pageSize" default:"20" minimum:"1" maximum:"100" doc:"Items per page"`
	}

	GetAllFilesMetadataOutputDtoBody struct {
		FilesMetadata []GetFileMetaByIDOutputDtoBody
		Total         int64
	}

	GetAllFilesMetadataOutputDto struct {
		Body GetAllFilesMetadataOutputDtoBody
	}
)
