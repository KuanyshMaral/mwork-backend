package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
)

// repoStub is a mock for Repository
type repoStub struct {
	created *Upload
	getByID *Upload
	deleted uuid.UUID
}

func (r *repoStub) Create(_ context.Context, upload *Upload) error {
	r.created = upload
	return nil
}

func (r *repoStub) GetByID(_ context.Context, _ uuid.UUID) (*Upload, error) {
	if r.getByID == nil {
		return nil, ErrUploadNotFound
	}
	return r.getByID, nil
}

func (r *repoStub) Delete(_ context.Context, id uuid.UUID) error {
	r.deleted = id
	return nil
}

// storageStub is a mock for storage.Storage (LocalStorage)
type storageStub struct {
	savedKey     string
	savedContent []byte
	deletedKey   string
}

func (s *storageStub) Save(_ context.Context, key string, r io.Reader, _ string) error {
	s.savedKey = key
	data, _ := io.ReadAll(r)
	s.savedContent = data
	return nil
}

func (s *storageStub) Delete(_ context.Context, key string) error {
	s.deletedKey = key
	return nil
}

func (s *storageStub) GetURL(key string) string {
	return "http://localhost:8080/static/" + key
}

func TestUpload(t *testing.T) {
	repo := &repoStub{}
	st := &storageStub{}
	svc := NewService(repo, st, "http://localhost:8080/static")

	authorID := uuid.New()
	fileName := "test.txt"
	content := []byte("hello world")

	up, err := svc.Upload(context.Background(), authorID, fileName, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify Service return
	if up.AuthorID != authorID {
		t.Errorf("expected authorID %s, got %s", authorID, up.AuthorID)
	}
	if up.OriginalName != fileName {
		t.Errorf("expected originalName %s, got %s", fileName, up.OriginalName)
	}
	if up.SizeBytes != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), up.SizeBytes)
	}

	// Verify Repo Create
	if repo.created == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if repo.created.ID != up.ID {
		t.Errorf("repo created ID mismatch")
	}

	// Verify Storage Save
	// Key format is {authorID}/{id}.ext
	expectedKey := authorID.String() + "/" + up.ID.String() + ".txt"
	if st.savedKey != expectedKey {
		t.Errorf("expected storage key %s, got %s", expectedKey, st.savedKey)
	}
	if !bytes.Equal(st.savedContent, content) {
		t.Errorf("storage content mismatch")
	}
}

func TestGetByID(t *testing.T) {
	uid := uuid.New()
	repo := &repoStub{
		getByID: &Upload{
			ID:           uid,
			AuthorID:     uuid.New(),
			OriginalName: "foo.jpg",
			CreatedAt:    time.Now(),
		},
	}
	st := &storageStub{}
	svc := NewService(repo, st, "http://localhost:8080/static")

	// Success case
	up, err := svc.GetByID(context.Background(), uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if up.ID != uid {
		t.Errorf("expected ID %s, got %s", uid, up.ID)
	}

	// Not found case
	repo.getByID = nil
	_, err = svc.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, ErrUploadNotFound) {
		t.Errorf("expected ErrUploadNotFound, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	uploadID := uuid.New()
	authorID := uuid.New()
	repo := &repoStub{
		getByID: &Upload{
			ID:       uploadID,
			AuthorID: authorID,
			FilePath: authorID.String() + "/" + uploadID.String() + ".txt",
		},
	}
	st := &storageStub{}
	svc := NewService(repo, st, "http://localhost:8080/static")

	// Success delete
	err := svc.Delete(context.Background(), uploadID, authorID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.deleted != uploadID {
		t.Errorf("expected repo delete %s, got %s", uploadID, repo.deleted)
	}
	expectedKey := authorID.String() + "/" + uploadID.String() + ".txt"
	if st.deletedKey != expectedKey {
		t.Errorf("expected storage delete key %s, got %s", expectedKey, st.deletedKey)
	}

	// Not owner case
	err = svc.Delete(context.Background(), uploadID, uuid.New())
	if !errors.Is(err, ErrNotOwner) {
		t.Errorf("expected ErrNotOwner, got %v", err)
	}
}

func (r *repoStub) ListByAuthor(_ context.Context, authorID uuid.UUID) ([]*Upload, error) {
	if r.created != nil && r.created.AuthorID == authorID {
		return []*Upload{r.created}, nil
	}
	if r.getByID != nil && r.getByID.AuthorID == authorID {
		return []*Upload{r.getByID}, nil
	}
	return []*Upload{}, nil
}

func TestListByAuthor(t *testing.T) {
	authorID := uuid.New()
	upload := &Upload{
		ID:           uuid.New(),
		AuthorID:     authorID,
		OriginalName: "list_me.jpg",
	}
	repo := &repoStub{created: upload}
	st := &storageStub{}
	svc := NewService(repo, st, "http://localhost:8080/static")

	list, err := svc.ListByAuthor(context.Background(), authorID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(list))
	}
	if list[0].ID != upload.ID {
		t.Errorf("expected upload ID %s, got %s", upload.ID, list[0].ID)
	}
}
