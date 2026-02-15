package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestUploadStorageAdapterMove(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	st, err := NewLocalStorage(base, "/media")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	ctx := context.Background()
	srcKey := "staging/u1/a.txt"
	dstKey := "final/u1/b.txt"
	content := "hello-upload"

	if err := st.Put(ctx, srcKey, bytes.NewReader([]byte(content)), "text/plain"); err != nil {
		t.Fatalf("Put src: %v", err)
	}

	adapter := NewUploadStorageAdapter(st)
	if err := adapter.Move(ctx, srcKey, dstKey); err != nil {
		t.Fatalf("Move: %v", err)
	}

	srcExists, err := st.Exists(ctx, srcKey)
	if err != nil {
		t.Fatalf("Exists src: %v", err)
	}
	if srcExists {
		t.Fatalf("source key still exists after move")
	}

	rc, err := st.Get(ctx, dstKey)
	if err != nil {
		t.Fatalf("Get dst: %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("Read dst: %v", err)
	}
	if string(data) != content {
		t.Fatalf("unexpected content: got %q want %q", string(data), content)
	}

	if got, want := adapter.GetURL(dstKey), "/media/"+dstKey; got != want {
		t.Fatalf("GetURL = %q, want %q", got, want)
	}

}
