package photostudio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSyncUserSuccess(t *testing.T) {
	payload := SyncUserPayload{
		MWorkUserID: "6c237b44-0a4f-4a03-8ba9-9724b3a3c5d8",
		Email:       "user@example.com",
		Role:        "model",
	}

	statuses := []int{http.StatusOK, http.StatusCreated}
	for _, status := range statuses {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("invalid method"))
				return
			}
			if r.URL.Path != "/internal/mwork/users/sync" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("invalid path"))
				return
			}
			if r.Header.Get("Content-Type") != "application/json" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("invalid content type"))
				return
			}
			if r.Header.Get("Authorization") != "Bearer test-token" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("invalid auth"))
				return
			}
			if r.Header.Get("User-Agent") != "MWork/1.0 photostudio-sync" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("invalid user agent"))
				return
			}
			w.WriteHeader(status)
		}))
		t.Cleanup(server.Close)

		client := NewClient(server.URL, "test-token", time.Second, "MWork/1.0 photostudio-sync")
		err := client.SyncUser(context.Background(), payload)
		if err != nil {
			t.Fatalf("expected no error for status %d, got %v", status, err)
		}
	}
}

func TestSyncUserHTTPErrorIncludesBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "token", time.Second, "MWork/1.0 photostudio-sync")
	err := client.SyncUser(context.Background(), SyncUserPayload{
		MWorkUserID: "user",
		Email:       "user@example.com",
		Role:        "model",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status=400") || !strings.Contains(err.Error(), "body=bad request") {
		t.Fatalf("expected status and body in error, got %v", err)
	}
}

func TestSyncUserTimeoutClassified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "token", 20*time.Millisecond, "MWork/1.0 photostudio-sync")
	err := client.SyncUser(context.Background(), SyncUserPayload{
		MWorkUserID: "user",
		Email:       "user@example.com",
		Role:        "model",
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "photostudio sync timeout") {
		t.Fatalf("expected timeout classification, got %v", err)
	}
}
