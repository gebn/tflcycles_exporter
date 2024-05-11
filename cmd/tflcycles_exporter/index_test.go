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


func TestBuildIndexHandler(t *testing.T) {
	t.Parallel()

	handler, err := buildIndexHandler(slog.Default())
	if err != nil {
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
