package repository

import (
	"bytes"
	"context"
	"errors"
	"lorem-backend/internal/database"
	"mime/multipart"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// ────────────────────────────────────────────────────────────
// Mock ObjectStorage Helper
// ────────────────────────────────────────────────────────────

type MockObjectStorage struct {
	mock.Mock
}

func (m *MockObjectStorage) UploadFile(ctx context.Context, objKey string, file multipart.File, size int64, contentType string) (string, error) {
	args := m.Called(ctx, objKey, file, size, contentType)
	return args.String(0), args.Error(1)
}

func (m *MockObjectStorage) GeneratePresignUrl(ctx context.Context, objKey string) (string, error) {
	args := m.Called(ctx, objKey)
	return args.String(0), args.Error(1)
}

// ────────────────────────────────────────────────────────────
// Suite Setup
// ────────────────────────────────────────────────────────────

type FileRepositoryTestSuite struct {
	suite.Suite
	mockDB   *database.MockDatabase
	mockS3   *MockObjectStorage
	fileRepo FileRepository
	ctx      context.Context
}

func (s *FileRepositoryTestSuite) SetupTest() {
	s.mockDB = database.NewMockDatabase(s.T())
	s.mockS3 = new(MockObjectStorage)
	s.fileRepo = NewFileMetaPostgresRepository(s.mockDB, s.mockS3)
	s.ctx = context.Background()
}

func (s *FileRepositoryTestSuite) TearDownTest() {
	s.NoError(s.mockDB.Mock.ExpectationsWereMet())
}

// ────────────────────────────────────────────────────────────
// Test CreateFileMeta
// ────────────────────────────────────────────────────────────

func (s *FileRepositoryTestSuite) TestCreateFileMeta() {
	fileID := uuid.New()
	fileMeta := &database.File{
		OriginalName: "photo.png",
		Name:         "uuid-name.png",
		Size:         200,
		ContentType:  "image/png",
		ObjectKey:    "uploads/uuid-name.png",
	}

	testCases := []struct {
		name    string
		setup   func()
		wantErr bool
		verify  func(uuid.UUID, error)
	}{
		{
			name: "Success - inserts file metadata",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "files"`)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(fileID))
				s.mockDB.Mock.ExpectCommit()
			},
			wantErr: false,
			verify: func(id uuid.UUID, err error) {
				s.Require().NoError(err)
				s.Equal(fileID, id)
			},
		},
		{
			name: "Failure - db insert error",
			setup: func() {
				s.mockDB.Mock.ExpectBegin()
				s.mockDB.Mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "files"`)).
					WillReturnError(errors.New("insert error"))
				s.mockDB.Mock.ExpectRollback()
			},
			wantErr: true,
			verify: func(id uuid.UUID, err error) {
				s.Require().Error(err)
				s.Contains(err.Error(), "insert error")
				s.Equal(uuid.Nil, id)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setup()

			id, err := s.fileRepo.CreateFileMeta(s.ctx, fileMeta)

			tc.verify(id, err)
			s.TearDownTest()
		})
	}
}

// ────────────────────────────────────────────────────────────
// Test GetFileMetaByID
// ────────────────────────────────────────────────────────────

func (s *FileRepositoryTestSuite) TestGetFileMetaByID() {
	fileID := uuid.New()

	testCases := []struct {
		name    string
		setup   func()
		wantErr bool
		verify  func(*database.File, error)
	}{
		{
			name: "Success - returns file metadata",
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "original_name", "name", "size", "content_type", "object_key"}).
					AddRow(fileID, time.Now(), time.Now(), nil, "test.txt", "uuid-test.txt", 123, "text/plain", "uploads/uuid-test.txt")

				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "files" WHERE`).
					WithArgs(fileID, 1).
					WillReturnRows(rows)
			},
			wantErr: false,
			verify: func(f *database.File, err error) {
				s.Require().NoError(err)
				s.NotNil(f)
				s.Equal(fileID, f.ID)
				s.Equal("test.txt", f.OriginalName)
				s.Equal(int64(123), f.Size)
			},
		},
		{
			name: "Failure - record not found",
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "files" WHERE`).
					WithArgs(fileID, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: true,
			verify: func(f *database.File, err error) {
				s.Require().Error(err)
				s.True(errors.Is(err, gorm.ErrRecordNotFound))
				s.Nil(f)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setup()

			f, err := s.fileRepo.GetFileMetaByID(s.ctx, fileID)

			tc.verify(f, err)
			s.TearDownTest()
		})
	}
}

// ────────────────────────────────────────────────────────────
// Test GetAllFilesMetadata
// ────────────────────────────────────────────────────────────

func (s *FileRepositoryTestSuite) TestGetAllFilesMetadata() {
	fileID1 := uuid.New()
	fileID2 := uuid.New()

	testCases := []struct {
		name     string
		page     int64
		pageSize int64
		setup    func()
		wantErr  bool
		verify   func([]database.File, int64, error)
	}{
		{
			name:     "Success - returns paginated files",
			page:     1,
			pageSize: 10,
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`^SELECT count\(\*\) FROM "files"`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

				rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "original_name", "name", "size", "content_type", "object_key"}).
					AddRow(fileID1, time.Now(), time.Now(), nil, "1.pdf", "u1.pdf", 100, "application/pdf", "keys/u1.pdf").
					AddRow(fileID2, time.Now(), time.Now(), nil, "2.pdf", "u2.pdf", 200, "application/pdf", "keys/u2.pdf")

				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "files"`).
					WithArgs(10).
					WillReturnRows(rows)
			},
			wantErr: false,
			verify: func(files []database.File, total int64, err error) {
				s.Require().NoError(err)
				s.Equal(int64(2), total)
				s.Len(files, 2)
				s.Equal(fileID1, files[0].ID)
				s.Equal(fileID2, files[1].ID)
			},
		},
		{
			name:     "Success - offset calculation for page 3",
			page:     3,
			pageSize: 5,
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`^SELECT count\(\*\) FROM "files"`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))

				rows := sqlmock.NewRows([]string{"id", "original_name", "name"}).
					AddRow(fileID1, "1.pdf", "u1.pdf")

				// limit = 5, offset = (3-1)*5 = 10
				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "files"`).
					WithArgs(5, 10).
					WillReturnRows(rows)
			},
			wantErr: false,
			verify: func(files []database.File, total int64, err error) {
				s.Require().NoError(err)
				s.Equal(int64(15), total)
				s.Len(files, 1)
			},
		},
		{
			name:     "Failure - count query fails",
			page:     1,
			pageSize: 10,
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`^SELECT count\(\*\) FROM "files"`).
					WillReturnError(errors.New("count query fail"))
			},
			wantErr: true,
			verify: func(files []database.File, total int64, err error) {
				s.Require().Error(err)
				s.Contains(err.Error(), "count query fail")
				s.Nil(files)
				s.Equal(int64(0), total)
			},
		},
		{
			name:     "Failure - find query fails",
			page:     1,
			pageSize: 10,
			setup: func() {
				s.mockDB.Mock.ExpectQuery(`^SELECT count\(\*\) FROM "files"`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

				s.mockDB.Mock.ExpectQuery(`^SELECT \* FROM "files"`).
					WithArgs(10).
					WillReturnError(errors.New("find query fail"))
			},
			wantErr: true,
			verify: func(files []database.File, total int64, err error) {
				s.Require().Error(err)
				s.Contains(err.Error(), "find query fail")
				s.Nil(files)
				s.Equal(int64(0), total)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setup()

			files, total, err := s.fileRepo.GetAllFilesMetadata(s.ctx, tc.page, tc.pageSize)

			tc.verify(files, total, err)
			s.TearDownTest()
		})
	}
}

// ────────────────────────────────────────────────────────────
// Test UploadFile (S3 delegation)
// ────────────────────────────────────────────────────────────

type mockFileReader struct {
	*bytes.Reader
}

func (m *mockFileReader) Close() error { return nil }

func (s *FileRepositoryTestSuite) TestUploadFile() {
	mockFile := &mockFileReader{Reader: bytes.NewReader([]byte("test data"))}

	testCases := []struct {
		name       string
		setupMocks func()
		wantErr    bool
		verify     func(string, error)
	}{
		{
			name: "Success - delegates to S3 ObjectStorage",
			setupMocks: func() {
				s.mockS3.On("UploadFile", s.ctx, "keys/file.txt", mockFile, int64(9), "text/plain").
					Return("keys/file.txt", nil).Once()
			},
			wantErr: false,
			verify: func(key string, err error) {
				s.Require().NoError(err)
				s.Equal("keys/file.txt", key)
			},
		},
		{
			name: "Failure - S3 ObjectStorage returns error",
			setupMocks: func() {
				s.mockS3.On("UploadFile", s.ctx, "keys/file.txt", mockFile, int64(9), "text/plain").
					Return("", errors.New("s3 upload fail")).Once()
			},
			wantErr: true,
			verify: func(key string, err error) {
				s.Require().Error(err)
				s.Contains(err.Error(), "s3 upload fail")
				s.Equal("", key)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMocks()

			key, err := s.fileRepo.UploadFile(s.ctx, "keys/file.txt", mockFile, 9, "text/plain")

			tc.verify(key, err)
			s.mockS3.AssertExpectations(s.T())
		})
	}
}

// ────────────────────────────────────────────────────────────
// Test GeneratePresignUrl (S3 delegation)
// ────────────────────────────────────────────────────────────

func (s *FileRepositoryTestSuite) TestGeneratePresignUrl() {
	testCases := []struct {
		name       string
		setupMocks func()
		wantErr    bool
		verify     func(string, error)
	}{
		{
			name: "Success - delegates to S3 ObjectStorage",
			setupMocks: func() {
				s.mockS3.On("GeneratePresignUrl", s.ctx, "keys/file.txt").
					Return("https://s3.amazonaws.com/presigned-url", nil).Once()
			},
			wantErr: false,
			verify: func(url string, err error) {
				s.Require().NoError(err)
				s.Equal("https://s3.amazonaws.com/presigned-url", url)
			},
		},
		{
			name: "Failure - S3 ObjectStorage returns error",
			setupMocks: func() {
				s.mockS3.On("GeneratePresignUrl", s.ctx, "keys/file.txt").
					Return("", errors.New("presign error")).Once()
			},
			wantErr: true,
			verify: func(url string, err error) {
				s.Require().Error(err)
				s.Contains(err.Error(), "presign error")
				s.Equal("", url)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			tc.setupMocks()

			url, err := s.fileRepo.GeneratePresignUrl(s.ctx, "keys/file.txt")

			tc.verify(url, err)
			s.mockS3.AssertExpectations(s.T())
		})
	}
}

// Runner
func TestFileRepository(t *testing.T) {
	suite.Run(t, new(FileRepositoryTestSuite))
}
