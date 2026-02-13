package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestMountProfileExperienceRoutes_NoDuplicateProfilesMount(t *testing.T) {
	root := chi.NewRouter()
	root.Mount("/profiles", chi.NewRouter())

	okHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	func() {
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatalf("registering experience routes panicked: %v", rec)
			}
		}()
		mountProfileExperienceRoutes(root, func(next http.Handler) http.Handler { return next }, okHandler, okHandler, okHandler)
	}()

	t.Run("list route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/profiles/123/experience", nil)
		rr := httptest.NewRecorder()
		root.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("create route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/profiles/123/experience", nil)
		rr := httptest.NewRecorder()
		root.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("delete route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/profiles/123/experience/456", nil)
		rr := httptest.NewRecorder()
		root.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}
	})
}
