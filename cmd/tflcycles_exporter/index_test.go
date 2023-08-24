package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gebn/go-stamp/v2"
)

func TestRenderIndex(t *testing.T) {
	t.Parallel()

	if _, err := renderIndex(); err != nil {
		t.Fatal(err)
	}
}

func TestBuildIndexHandler_NotFound(t *testing.T) {
	t.Parallel()

	handler, err := buildIndexHandler(slog.Default())
	if _, err := renderIndex(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, "/should-not-exist", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wanted %v for %v, got %v", http.StatusNotFound, req.URL.Path, rr.Code)
	}
}

func TestBuildIndexHandler_Root(t *testing.T) {
	t.Parallel()

	handler, err := buildIndexHandler(slog.Default())
	if _, err := renderIndex(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wanted %v for %v, got %v", http.StatusOK, req.URL.Path, rr.Code)
	}

	if !strings.Contains(rr.Body.String(), stamp.Version) {
		t.Errorf("response body for %v did not contain version", req.URL.Path)
	}
}
