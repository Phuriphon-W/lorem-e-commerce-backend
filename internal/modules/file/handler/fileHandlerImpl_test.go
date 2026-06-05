package handler

import (
	"bytes"
	"context"
	"errors"
	"lorem-backend/internal/database"
	"lorem-backend/internal/modules/file/dto"
	"mime/multipart"
	"reflect"
	"testing"
	"unsafe"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// ────────────────────────────────────────────────────────────
// Mock File Helper
// ────────────────────────────────────────────────────────────

type mockMultipartFile struct {
	*bytes.Reader
}

func (m *mockMultipartFile) Close() error {
	return nil
}

func newMockMultipartFile(data []byte) multipart.File {
	return &mockMultipartFile{
		Reader: bytes.NewReader(data),
	}
}

// Helper to set private data field in huma.MultipartFormFiles[T]
func setMultipartFormFilesData[T any](m *huma.MultipartFormFiles[T], val *T) {
	v := reflect.ValueOf(m).Elem()
	f := v.FieldByName("data")
	ptr := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	ptr.Set(reflect.ValueOf(val))
}

// ────────────────────────────────────────────────────────────
// Mock Definitions
// ────────────────────────────────────────────────────────────

type MockFileRepository struct {
	mock.Mock
}

func (m *MockFileRepository) CreateFileMeta(ctx context.Context, fileMeta *database.File) (uuid.UUID, error) {
	args := m.Called(ctx, fileMeta)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockFileRepository) GetFileMetaByID(ctx context.Context, fileID uuid.UUID) (*database.File, error) {
	args := m.Called(ctx, fileID)
	if args.Get(0) != nil {
		return args.Get(0).(*database.File), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockFileRepository) GetAllFilesMetadata(ctx context.Context, page int64, pageSize int64) ([]database.File, int64, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) != nil {
		return args.Get(0).([]database.File), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockFileRepository) UploadFile(ctx context.Context, objKey string, file multipart.File, size int64, contentType string) (string, error) {
	args := m.Called(ctx, objKey, file, size, contentType)
	return args.String(0), args.Error(1)
}

func (m *MockFileRepository) GeneratePresignUrl(ctx context.Context, objKey string) (string, error) {
	args := m.Called(ctx, objKey)
	return args.String(0), args.Error(1)
}

// ────────────────────────────────────────────────────────────
// Suite Setup
// ────────────────────────────────────────────────────────────

type FileHandlerTestSuite struct {
	suite.Suite
	mockFileRepo *MockFileRepository
	handler      FileHandler
	ctx          context.Context
}

func (s *FileHandlerTestSuite) SetupTest() {
	s.mockFileRepo = new(MockFileRepository)
	s.handler = NewFileHandlerImpl(s.mockFileRepo)
	s.ctx = context.Background()
}

// ────────────────────────────────────────────────────────────
// Test UploadFile
// ────────────────────────────────────────────────────────────

func (s *FileHandlerTestSuite) TestUploadFile() {
	fileID := uuid.New()
	mockFile := newMockMultipartFile([]byte("test content"))

	testCases := []struct {
		name       string
		setupMocks func()
		inputData  func() *dto.UploadFileInputDto
		wantErr    bool
		errStatus  int
		verify     func(*dto.UploadFileOutputDto)
	}{
		{
			name: "Success - file uploaded and metadata created",
			setupMocks: func() {
				s.mockFileRepo.On("UploadFile", s.ctx, mock.Anything, mock.Anything, int64(12), "text/plain").
					Return("uploads/test.txt", nil).Once()

				s.mockFileRepo.On("CreateFileMeta", s.ctx, mock.MatchedBy(func(f *database.File) bool {
					return f.OriginalName == "test.txt" && f.Size == 12 && f.ContentType == "text/plain" && f.ObjectKey == "uploads/test.txt"
				})).Return(fileID, nil).Once()
			},
			inputData: func() *dto.UploadFileInputDto {
				input := &dto.UploadFileInputDto{}
				setMultipartFormFilesData(&input.RawBody, &struct {
					File          huma.FormFile `form:"file" required:"true" doc:"The file content to upload"`
					ObjectBaseKey string        `form:"objectBaseKey" doc:"Base object key in object storage"`
				}{
					File: huma.FormFile{
						File:        mockFile,
						Filename:    "test.txt",
						Size:        12,
						ContentType: "text/plain",
						IsSet:       true,
					},
					ObjectBaseKey: "uploads",
				})
				return input
			},
			wantErr: false,
			verify: func(out *dto.UploadFileOutputDto) {
				s.NotNil(out)
				s.Equal(fileID.String(), out.Body.FileID)
			},
		},
		{
			name: "Failure - upload fails",
			setupMocks: func() {
				s.mockFileRepo.On("UploadFile", s.ctx, mock.Anything, mock.Anything, int64(12), "text/plain").
					Return("", errors.New("s3 upload error")).Once()
			},
			inputData: func() *dto.UploadFileInputDto {
				input := &dto.UploadFileInputDto{}
				setMultipartFormFilesData(&input.RawBody, &struct {
					File          huma.FormFile `form:"file" required:"true" doc:"The file content to upload"`
					ObjectBaseKey string        `form:"objectBaseKey" doc:"Base object key in object storage"`
				}{
					File: huma.FormFile{
						File:        mockFile,
						Filename:    "test.txt",
						Size:        12,
						ContentType: "text/plain",
						IsSet:       true,
					},
					ObjectBaseKey: "uploads",
				})
				return input
			},
			wantErr:   true,
			errStatus: 500,
		},
		{
			name: "Failure - metadata save fails",
			setupMocks: func() {
				s.mockFileRepo.On("UploadFile", s.ctx, mock.Anything, mock.Anything, int64(12), "text/plain").
					Return("uploads/test.txt", nil).Once()

				s.mockFileRepo.On("CreateFileMeta", s.ctx, mock.Anything).
					Return(uuid.Nil, errors.New("db error")).Once()
			},
			inputData: func() *dto.UploadFileInputDto {
				input := &dto.UploadFileInputDto{}
				setMultipartFormFilesData(&input.RawBody, &struct {
					File          huma.FormFile `form:"file" required:"true" doc:"The file content to upload"`
					ObjectBaseKey string        `form:"objectBaseKey" doc:"Base object key in object storage"`
				}{
					File: huma.FormFile{
						File:        mockFile,
						Filename:    "test.txt",
						Size:        12,
						ContentType: "text/plain",
						IsSet:       true,
					},
					ObjectBaseKey: "uploads",
				})
				return input
			},
			wantErr:   true,
			errStatus: 500,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMocks()

			out, err := s.handler.UploadFile(s.ctx, tc.inputData())

			if tc.wantErr {
				s.Require().Error(err)
				var humaErr huma.StatusError
				s.Require().True(errors.As(err, &humaErr), "expected huma.StatusError")
				s.Equal(tc.errStatus, humaErr.GetStatus())
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					tc.verify(out)
				}
			}

			s.mockFileRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// Test UploadStaticFile
// ────────────────────────────────────────────────────────────

func (s *FileHandlerTestSuite) TestUploadStaticFile() {
	fileID := uuid.New()
	mockFile := newMockMultipartFile([]byte("static content"))

	testCases := []struct {
		name       string
		setupMocks func()
		inputData  func() *dto.UploadStaticFileInputDto
		wantErr    bool
		errStatus  int
		verify     func(*dto.UploadStaticFileOutputDto)
	}{
		{
			name: "Success - static file uploaded and metadata created",
			setupMocks: func() {
				s.mockFileRepo.On("UploadFile", s.ctx, "static/logo.png", mock.Anything, int64(14), "image/png").
					Return("static/logo.png", nil).Once()

				s.mockFileRepo.On("CreateFileMeta", s.ctx, mock.MatchedBy(func(f *database.File) bool {
					return f.OriginalName == "logo.png" && f.Size == 14 && f.ContentType == "image/png" && f.ObjectKey == "static/logo.png"
				})).Return(fileID, nil).Once()
			},
			inputData: func() *dto.UploadStaticFileInputDto {
				input := &dto.UploadStaticFileInputDto{}
				setMultipartFormFilesData(&input.RawBody, &struct {
					File          huma.FormFile `form:"file" required:"true" doc:"The file content to upload"`
					FileName      string        `form:"string" doc:"The name of the file to be stored in storage"`
					ObjectBaseKey string        `form:"objectBaseKey" doc:"Base object key in object storage"`
				}{
					File: huma.FormFile{
						File:        mockFile,
						Filename:    "logo.png",
						Size:        14,
						ContentType: "image/png",
						IsSet:       true,
					},
					FileName:      "logo.png",
					ObjectBaseKey: "static",
				})
				return input
			},
			wantErr: false,
			verify: func(out *dto.UploadStaticFileOutputDto) {
				s.NotNil(out)
				s.Equal(fileID.String(), out.Body.FileID)
				s.Equal("static/logo.png", out.Body.ObjectKey)
			},
		},
		{
			name: "Failure - upload fails",
			setupMocks: func() {
				s.mockFileRepo.On("UploadFile", s.ctx, "static/logo.png", mock.Anything, int64(14), "image/png").
					Return("", errors.New("s3 upload error")).Once()
			},
			inputData: func() *dto.UploadStaticFileInputDto {
				input := &dto.UploadStaticFileInputDto{}
				setMultipartFormFilesData(&input.RawBody, &struct {
					File          huma.FormFile `form:"file" required:"true" doc:"The file content to upload"`
					FileName      string        `form:"string" doc:"The name of the file to be stored in storage"`
					ObjectBaseKey string        `form:"objectBaseKey" doc:"Base object key in object storage"`
				}{
					File: huma.FormFile{
						File:        mockFile,
						Filename:    "logo.png",
						Size:        14,
						ContentType: "image/png",
						IsSet:       true,
					},
					FileName:      "logo.png",
					ObjectBaseKey: "static",
				})
				return input
			},
			wantErr:   true,
			errStatus: 500,
		},
		{
			name: "Failure - metadata save fails",
			setupMocks: func() {
				s.mockFileRepo.On("UploadFile", s.ctx, "static/logo.png", mock.Anything, int64(14), "image/png").
					Return("static/logo.png", nil).Once()

				s.mockFileRepo.On("CreateFileMeta", s.ctx, mock.Anything).
					Return(uuid.Nil, errors.New("db error")).Once()
			},
			inputData: func() *dto.UploadStaticFileInputDto {
				input := &dto.UploadStaticFileInputDto{}
				setMultipartFormFilesData(&input.RawBody, &struct {
					File          huma.FormFile `form:"file" required:"true" doc:"The file content to upload"`
					FileName      string        `form:"string" doc:"The name of the file to be stored in storage"`
					ObjectBaseKey string        `form:"objectBaseKey" doc:"Base object key in object storage"`
				}{
					File: huma.FormFile{
						File:        mockFile,
						Filename:    "logo.png",
						Size:        14,
						ContentType: "image/png",
						IsSet:       true,
					},
					FileName:      "logo.png",
					ObjectBaseKey: "static",
				})
				return input
			},
			wantErr:   true,
			errStatus: 500,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMocks()

			out, err := s.handler.UploadStaticFile(s.ctx, tc.inputData())

			if tc.wantErr {
				s.Require().Error(err)
				var humaErr huma.StatusError
				s.Require().True(errors.As(err, &humaErr), "expected huma.StatusError")
				s.Equal(tc.errStatus, humaErr.GetStatus())
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					tc.verify(out)
				}
			}

			s.mockFileRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// Test DownLoadFile
// ────────────────────────────────────────────────────────────

func (s *FileHandlerTestSuite) TestDownLoadFile() {
	fileID := uuid.New()
	mockFileMeta := &database.File{
		OriginalName: "original.txt",
		Name:         "uuid-name.txt",
		Size:         100,
		ContentType:  "text/plain",
		ObjectKey:    "uploads/uuid-name.txt",
	}

	testCases := []struct {
		name       string
		input      *dto.DownloadFileInputDto
		setupMocks func()
		wantErr    bool
		errStatus  int
		verify     func(*dto.DownloadFileOutputDto)
	}{
		{
			name: "Success - returns fileName and downloadURL",
			input: &dto.DownloadFileInputDto{
				ID: fileID,
			},
			setupMocks: func() {
				s.mockFileRepo.On("GetFileMetaByID", s.ctx, fileID).Return(mockFileMeta, nil).Once()
				s.mockFileRepo.On("GeneratePresignUrl", s.ctx, "uploads/uuid-name.txt").
					Return("https://s3.amazonaws.com/bucket/uploads/uuid-name.txt?signature=...", nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.DownloadFileOutputDto) {
				s.NotNil(out)
				s.Equal("uuid-name.txt", out.Body.FileName)
				s.Equal("https://s3.amazonaws.com/bucket/uploads/uuid-name.txt?signature=...", out.Body.DownloadURL)
			},
		},
		{
			name: "Failure - metadata not found",
			input: &dto.DownloadFileInputDto{
				ID: fileID,
			},
			setupMocks: func() {
				s.mockFileRepo.On("GetFileMetaByID", s.ctx, fileID).Return(nil, errors.New("not found")).Once()
			},
			wantErr:   true,
			errStatus: 404,
		},
		{
			name: "Failure - presign URL fails",
			input: &dto.DownloadFileInputDto{
				ID: fileID,
			},
			setupMocks: func() {
				s.mockFileRepo.On("GetFileMetaByID", s.ctx, fileID).Return(mockFileMeta, nil).Once()
				s.mockFileRepo.On("GeneratePresignUrl", s.ctx, "uploads/uuid-name.txt").
					Return("", errors.New("presign error")).Once()
			},
			wantErr:   true,
			errStatus: 500,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMocks()

			out, err := s.handler.DownLoadFile(s.ctx, tc.input)

			if tc.wantErr {
				s.Require().Error(err)
				var humaErr huma.StatusError
				s.Require().True(errors.As(err, &humaErr), "expected huma.StatusError")
				s.Equal(tc.errStatus, humaErr.GetStatus())
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					tc.verify(out)
				}
			}

			s.mockFileRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// Test DownloadFileByKey
// ────────────────────────────────────────────────────────────

func (s *FileHandlerTestSuite) TestDownloadFileByKey() {
	testCases := []struct {
		name       string
		input      *dto.DownloadFileByKeyInputDto
		setupMocks func()
		wantErr    bool
		errStatus  int
		verify     func(*dto.DownloadFileByKeyOutputDto)
	}{
		{
			name: "Success - returns presigned URL",
			input: &dto.DownloadFileByKeyInputDto{
				ObjectKey: "uploads/logo.png",
			},
			setupMocks: func() {
				s.mockFileRepo.On("GeneratePresignUrl", s.ctx, "uploads/logo.png").
					Return("https://s3.amazonaws.com/bucket/uploads/logo.png?signature=...", nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.DownloadFileByKeyOutputDto) {
				s.NotNil(out)
				s.Equal("https://s3.amazonaws.com/bucket/uploads/logo.png?signature=...", out.Body.DownloadURL)
			},
		},
		{
			name: "Failure - file key not found / error generating url",
			input: &dto.DownloadFileByKeyInputDto{
				ObjectKey: "uploads/logo.png",
			},
			setupMocks: func() {
				s.mockFileRepo.On("GeneratePresignUrl", s.ctx, "uploads/logo.png").
					Return("", errors.New("s3 key not found")).Once()
			},
			wantErr:   true,
			errStatus: 404,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMocks()

			out, err := s.handler.DownloadFileByKey(s.ctx, tc.input)

			if tc.wantErr {
				s.Require().Error(err)
				var humaErr huma.StatusError
				s.Require().True(errors.As(err, &humaErr), "expected huma.StatusError")
				s.Equal(tc.errStatus, humaErr.GetStatus())
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					tc.verify(out)
				}
			}

			s.mockFileRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// Test GetFileMetaByID
// ────────────────────────────────────────────────────────────

func (s *FileHandlerTestSuite) TestGetFileMetaByID() {
	fileID := uuid.New()
	mockFileMeta := &database.File{
		Base: database.Base{
			ID: fileID,
		},
		OriginalName: "photo.jpg",
		Name:         "uuid-photo.jpg",
		Size:         2048,
		ContentType:  "image/jpeg",
		ObjectKey:    "uploads/uuid-photo.jpg",
	}

	testCases := []struct {
		name       string
		input      *dto.GetFileMetaByIDInputDto
		setupMocks func()
		wantErr    bool
		errStatus  int
		verify     func(*dto.GetFileMetaByIDOutputDto)
	}{
		{
			name: "Success - returns file metadata",
			input: &dto.GetFileMetaByIDInputDto{
				ID: fileID,
			},
			setupMocks: func() {
				s.mockFileRepo.On("GetFileMetaByID", s.ctx, fileID).Return(mockFileMeta, nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.GetFileMetaByIDOutputDto) {
				s.NotNil(out)
				s.Equal(fileID, out.Body.ID)
				s.Equal("photo.jpg", out.Body.Name)
				s.Equal(uint(2048), out.Body.Size)
				s.Equal("image/jpeg", out.Body.ContentType)
				s.Equal("uploads/uuid-photo.jpg", out.Body.ObjectKey)
			},
		},
		{
			name: "Failure - file meta not found",
			input: &dto.GetFileMetaByIDInputDto{
				ID: fileID,
			},
			setupMocks: func() {
				s.mockFileRepo.On("GetFileMetaByID", s.ctx, fileID).Return(nil, errors.New("not found")).Once()
			},
			wantErr:   true,
			errStatus: 404,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMocks()

			out, err := s.handler.GetFileMetaByID(s.ctx, tc.input)

			if tc.wantErr {
				s.Require().Error(err)
				var humaErr huma.StatusError
				s.Require().True(errors.As(err, &humaErr), "expected huma.StatusError")
				s.Equal(tc.errStatus, humaErr.GetStatus())
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					tc.verify(out)
				}
			}

			s.mockFileRepo.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// Test GetAllFilesMetadata
// ────────────────────────────────────────────────────────────

func (s *FileHandlerTestSuite) TestGetAllFilesMetadata() {
	fileID1 := uuid.New()
	fileID2 := uuid.New()

	mockFilesMeta := []database.File{
		{
			Base: database.Base{
				ID: fileID1,
			},
			OriginalName: "f1.pdf",
			Name:         "uuid-f1.pdf",
			Size:         512,
			ContentType:  "application/pdf",
			ObjectKey:    "docs/uuid-f1.pdf",
		},
		{
			Base: database.Base{
				ID: fileID2,
			},
			OriginalName: "f2.png",
			Name:         "uuid-f2.png",
			Size:         1024,
			ContentType:  "image/png",
			ObjectKey:    "docs/uuid-f2.png",
		},
	}

	testCases := []struct {
		name       string
		input      *dto.GetAllFilesMetadataInputDto
		setupMocks func()
		wantErr    bool
		errStatus  int
		verify     func(*dto.GetAllFilesMetadataOutputDto)
	}{
		{
			name: "Success - returns files metadata and total",
			input: &dto.GetAllFilesMetadataInputDto{
				PageNumber: 1,
				PageSize:   10,
			},
			setupMocks: func() {
				s.mockFileRepo.On("GetAllFilesMetadata", s.ctx, int64(1), int64(10)).
					Return(mockFilesMeta, int64(2), nil).Once()
			},
			wantErr: false,
			verify: func(out *dto.GetAllFilesMetadataOutputDto) {
				s.NotNil(out)
				s.Equal(int64(2), out.Body.Total)
				s.Len(out.Body.FilesMetadata, 2)
				s.Equal(fileID1, out.Body.FilesMetadata[0].ID)
				s.Equal("uuid-f1.pdf", out.Body.FilesMetadata[0].Name) // Note that handler code uses Name in loop: f.Name (which is uuid-f1.pdf)
				s.Equal(fileID2, out.Body.FilesMetadata[1].ID)
			},
		},
		{
			name: "Failure - repo returns error",
			input: &dto.GetAllFilesMetadataInputDto{
				PageNumber: 1,
				PageSize:   10,
			},
			setupMocks: func() {
				s.mockFileRepo.On("GetAllFilesMetadata", s.ctx, int64(1), int64(10)).
					Return(nil, int64(0), errors.New("db error")).Once()
			},
			wantErr:   true,
			errStatus: 404,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMocks()

			out, err := s.handler.GetAllFilesMetadata(s.ctx, tc.input)

			if tc.wantErr {
				s.Require().Error(err)
				var humaErr huma.StatusError
				s.Require().True(errors.As(err, &humaErr), "expected huma.StatusError")
				s.Equal(tc.errStatus, humaErr.GetStatus())
			} else {
				s.Require().NoError(err)
				if tc.verify != nil {
					tc.verify(out)
				}
			}

			s.mockFileRepo.AssertExpectations(s.T())
		})
	}
}

// Runner
func TestFileHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(FileHandlerTestSuite))
}
