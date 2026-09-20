package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterWebFrontendServesAppAndPreservesAPINotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	webDir := t.TempDir()
	assetsDir := filepath.Join(webDir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<html>pan web</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "app.js"), []byte("console.log('pan')"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WEB_DIST_PATH", webDir)

	router := gin.New()
	registerWebFrontend(router)

	tests := []struct {
		name        string
		path        string
		status      int
		contentType string
		body        string
	}{
		{name: "root", path: "/", status: http.StatusOK, contentType: "text/html", body: "pan web"},
		{name: "asset", path: "/assets/app.js", status: http.StatusOK, contentType: "javascript", body: "console.log"},
		{name: "spa fallback", path: "/favorites", status: http.StatusOK, contentType: "text/html", body: "pan web"},
		{name: "api not found", path: "/api/missing", status: http.StatusNotFound, contentType: "application/json", body: "接口不存在"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			router.ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, test.status, recorder.Body.String())
			}
			if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, test.contentType) {
				t.Fatalf("Content-Type = %q, want it to contain %q", contentType, test.contentType)
			}
			if !strings.Contains(recorder.Body.String(), test.body) {
				t.Fatalf("body = %q, want it to contain %q", recorder.Body.String(), test.body)
			}
		})
	}
}

func TestRegisterWebFrontendSkipsMissingDistribution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("WEB_DIST_PATH", filepath.Join(t.TempDir(), "missing"))

	router := gin.New()
	registerWebFrontend(router)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestRegisterWebFrontendHandlesAPIRootAndUnknownMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<html>pan web</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WEB_DIST_PATH", webDir)

	router := gin.New()
	registerWebFrontend(router)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "api root", method: http.MethodGet, path: "/api"},
		{name: "unknown mutation", method: http.MethodPost, path: "/favorites"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.path, nil)
			router.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "pan web") {
				t.Fatalf("response unexpectedly served SPA: %q", recorder.Body.String())
			}
		})
	}
}
