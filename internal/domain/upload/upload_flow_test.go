package upload

import (
	"bytes"
	"context"
	"errors"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/storage"
)

type repoStub struct {
	created        *Upload
	getByID        *Upload
	markCommittedN int
	markSize       int64
}

func (r *repoStub) Create(_ context.Context, upload *Upload) error          { r.created = upload; return nil }
func (r *repoStub) GetByID(_ context.Context, _ uuid.UUID) (*Upload, error) { return r.getByID, nil }
func (r *repoStub) Update(_ context.Context, _ *Upload) error               { return nil }
func (r *repoStub) UpdateStaged(_ context.Context, _ *Upload) error         { return nil }
func (r *repoStub) MarkCommitted(_ context.Context, _ uuid.UUID, size int64, _ string, _ string, _ time.Time) error {
	r.markCommittedN++
	r.markSize = size
	return nil
}
func (r *repoStub) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (r *repoStub) ListByUser(_ context.Context, _ uuid.UUID, _ Category) ([]*Upload, error) {
	return nil, nil
}
func (r *repoStub) ListExpired(_ context.Context, _ time.Time) ([]*Upload, error) { return nil, nil }
func (r *repoStub) DeleteExpired(_ context.Context, _ time.Time) (int, error)     { return 0, nil }

type storageStub struct {
	info *storage.FileInfo
}

func (s *storageStub) Put(_ context.Context, _ string, _ io.Reader, _ string) error { return nil }
func (s *storageStub) Get(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("ok"))), nil
}
func (s *storageStub) Delete(_ context.Context, _ string) error         { return nil }
func (s *storageStub) Exists(_ context.Context, _ string) (bool, error) { return false, nil }
func (s *storageStub) GetURL(key string) string                         { return "https://storage/" + key }
func (s *storageStub) GetInfo(_ context.Context, _ string) (*storage.FileInfo, error) {
	return s.info, nil
}

type drainingStorageStub struct {
	storageStub
	lastPutBytes int
}

func (s *drainingStorageStub) Put(_ context.Context, _ string, r io.Reader, _ string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.lastPutBytes = len(data)
	return nil
}

type uploadStorageStub struct{}

func (s *uploadStorageStub) GeneratePresignedPutURL(_ context.Context, _ string, _ time.Duration, _ string) (string, error) {
	return "", nil
}
func (s *uploadStorageStub) Exists(_ context.Context, _ string) (bool, error) { return false, nil }
func (s *uploadStorageStub) Move(_ context.Context, _, _ string) error        { return nil }
func (s *uploadStorageStub) GetURL(key string) string                         { return "https://storage/" + key }

func TestInitCreatesStagedUploadWithNullSizeWhenUnknown(t *testing.T) {
	repo := &repoStub{}
	h := NewHandler(nil, "https://staging", &uploadStorageStub{}, repo, true)

	body := bytes.NewBufferString(`{"file_name":"avatar.jpg","content_type":"image/jpeg","file_size":0}`)
	req := httptest.NewRequest(http.MethodPost, "/files/init", body)
	uid := uuid.New()
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, uid))
	w := httptest.NewRecorder()

	h.Init(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if repo.created == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if repo.created.Size.Valid {
		t.Fatalf("expected size to be NULL for unknown staged upload, got %+v", repo.created.Size)
	}
}

func TestConfirmCommittedWithPositiveSize(t *testing.T) {
	uid := uuid.New()
	uploadID := uuid.New()
	repo := &repoStub{getByID: &Upload{
		ID:           uploadID,
		UserID:       uid,
		Category:     CategoryAvatar,
		Status:       StatusStaged,
		OriginalName: "avatar.jpg",
		MimeType:     "image/jpeg",
		Size:         sql.NullInt64{Int64: 128, Valid: true},
		StagingKey:   "uploads/staging/a.jpg",
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}}
	st := &storageStub{info: &storage.FileInfo{Size: 128, ContentType: "image/jpeg"}}
	svc := NewService(repo, st, nil, "https://staging")

	up, err := svc.Confirm(context.Background(), uploadID, uid)
	if err != nil {
		t.Fatalf("expected confirm success, got error: %v", err)
	}
	if up.Status != StatusCommitted {
		t.Fatalf("expected status committed, got %s", up.Status)
	}
	if repo.markCommittedN != 1 {
		t.Fatalf("expected MarkCommitted to be called once, got %d", repo.markCommittedN)
	}
	if repo.markSize <= 0 {
		t.Fatalf("expected committed size > 0, got %d", repo.markSize)
	}
}

func TestConfirmCommittedWhenStagedSizeUnknownUsesStorageSize(t *testing.T) {
	uid := uuid.New()
	uploadID := uuid.New()
	repo := &repoStub{getByID: &Upload{
		ID:           uploadID,
		UserID:       uid,
		Category:     CategoryAvatar,
		Status:       StatusStaged,
		OriginalName: "avatar.jpg",
		MimeType:     "image/jpeg",
		Size:         sql.NullInt64{},
		StagingKey:   "uploads/staging/a.jpg",
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}}
	st := &storageStub{info: &storage.FileInfo{Size: 128, ContentType: "image/jpeg"}}
	svc := NewService(repo, st, nil, "https://staging")

	up, err := svc.Confirm(context.Background(), uploadID, uid)
	if err != nil {
		t.Fatalf("expected confirm success, got error: %v", err)
	}
	if up.Status != StatusCommitted {
		t.Fatalf("expected status committed, got %s", up.Status)
	}
	if !up.Size.Valid || up.Size.Int64 != 128 {
		t.Fatalf("expected committed size to be set from storage, got %+v", up.Size)
	}
	if repo.markCommittedN != 1 {
		t.Fatalf("expected MarkCommitted to be called once, got %d", repo.markCommittedN)
	}
	if repo.markSize != 128 {
		t.Fatalf("expected committed size 128, got %d", repo.markSize)
	}
}
func TestConfirmAllowsEmptyStagingContentType(t *testing.T) {
	uid := uuid.New()
	uploadID := uuid.New()
	repo := &repoStub{getByID: &Upload{
		ID:           uploadID,
		UserID:       uid,
		Category:     CategoryAvatar,
		Status:       StatusStaged,
		OriginalName: "avatar.jpg",
		MimeType:     "image/jpeg",
		Size:         sql.NullInt64{Int64: 128, Valid: true},
		StagingKey:   "uploads/staging/a.jpg",
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}}
	st := &storageStub{info: &storage.FileInfo{Size: 128, ContentType: ""}}
	svc := NewService(repo, st, nil, "https://staging")

	up, err := svc.Confirm(context.Background(), uploadID, uid)
	if err != nil {
		t.Fatalf("expected confirm success with empty staging content type, got error: %v", err)
	}
	if up.Status != StatusCommitted {
		t.Fatalf("expected status committed, got %s", up.Status)
	}
	if repo.markCommittedN != 1 {
		t.Fatalf("expected MarkCommitted to be called once, got %d", repo.markCommittedN)
	}
}
func TestConfirmFailsWhenSizeZero(t *testing.T) {
	uid := uuid.New()
	uploadID := uuid.New()
	repo := &repoStub{getByID: &Upload{
		ID:           uploadID,
		UserID:       uid,
		Category:     CategoryAvatar,
		Status:       StatusStaged,
		OriginalName: "avatar.jpg",
		MimeType:     "image/jpeg",
		Size:         sql.NullInt64{Int64: 0, Valid: true},
		StagingKey:   "uploads/staging/a.jpg",
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}}
	st := &storageStub{info: &storage.FileInfo{Size: 0, ContentType: "image/jpeg"}}
	svc := NewService(repo, st, nil, "https://staging")

	_, err := svc.Confirm(context.Background(), uploadID, uid)
	if err == nil {
		t.Fatal("expected error for zero size, got nil")
	}
	if err != ErrInvalidUploadSize {
		t.Fatalf("expected ErrInvalidUploadSize, got %v", err)
	}
	if repo.markCommittedN != 0 {
		t.Fatalf("expected MarkCommitted to not be called, got %d", repo.markCommittedN)
	}
}

func TestUploadResponseUsesNilForNullSize(t *testing.T) {
	u := &Upload{ID: uuid.New(), Category: CategoryPhoto, Status: StatusStaged, OriginalName: "a", MimeType: "image/jpeg", Size: sql.NullInt64{}, CreatedAt: time.Now(), ExpiresAt: time.Now()}
	resp := UploadResponseFromEntity(u, "https://staging")
	if resp.Size != nil {
		t.Fatalf("expected nil size for NULL size, got %v", *resp.Size)
	}
}

func TestStagePersistsSizeAfterStorageReaderConsumption(t *testing.T) {
	repo := &repoStub{}
	st := &drainingStorageStub{}
	svc := NewService(repo, st, nil, "https://staging")

	fileBytes := []byte{0xFF, 0xD8, 0xFF, 0xD9} // minimal jpeg-like bytes for DetectContentType
	up, err := svc.Stage(context.Background(), uuid.New(), CategoryPhoto, "photo.jpg", bytes.NewReader(fileBytes))
	if err != nil {
		t.Fatalf("expected stage success, got error: %v", err)
	}
	if !up.Size.Valid || up.Size.Int64 != int64(len(fileBytes)) {
		t.Fatalf("expected upload size %d, got %+v", len(fileBytes), up.Size)
	}
	if repo.created == nil || !repo.created.Size.Valid || repo.created.Size.Int64 != int64(len(fileBytes)) {
		t.Fatalf("expected persisted upload size %d, got %+v", len(fileBytes), repo.created)
	}
	if st.lastPutBytes != len(fileBytes) {
		t.Fatalf("expected storage to receive %d bytes, got %d", len(fileBytes), st.lastPutBytes)
	}
}

func TestStageExistingPersistsSizeAfterStorageReaderConsumption(t *testing.T) {
	uid := uuid.New()
	uploadID := uuid.New()
	repo := &repoStub{getByID: &Upload{ID: uploadID, UserID: uid, StagingKey: "uploads/staging/u/f.jpg", ExpiresAt: time.Now().Add(time.Hour)}}
	st := &drainingStorageStub{}
	svc := NewService(repo, st, nil, "https://staging")

	fileBytes := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	up, err := svc.StageExisting(context.Background(), uploadID, uid, CategoryPhoto, "photo.jpg", bytes.NewReader(fileBytes))
	if err != nil {
		t.Fatalf("expected stage existing success, got error: %v", err)
	}
	if !up.Size.Valid || up.Size.Int64 != int64(len(fileBytes)) {
		t.Fatalf("expected upload size %d, got %+v", len(fileBytes), up.Size)
	}
	if st.lastPutBytes != len(fileBytes) {
		t.Fatalf("expected storage to receive %d bytes, got %d", len(fileBytes), st.lastPutBytes)
	}
}

func TestGetByIDForUser(t *testing.T) {
	uid := uuid.New()
	uploadID := uuid.New()
	repo := &repoStub{getByID: &Upload{ID: uploadID, UserID: uid}}
	svc := NewService(repo, &storageStub{}, nil, "https://staging")

	t.Run("owner can access upload", func(t *testing.T) {
		up, err := svc.GetByIDForUser(context.Background(), uploadID, uid)
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if up.ID != uploadID {
			t.Fatalf("expected upload id %s, got %s", uploadID, up.ID)
		}
	})

	t.Run("non-owner gets forbidden error", func(t *testing.T) {
		_, err := svc.GetByIDForUser(context.Background(), uploadID, uuid.New())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrNotUploadOwner) {
			t.Fatalf("expected ErrNotUploadOwner, got %v", err)
		}
	})
}
