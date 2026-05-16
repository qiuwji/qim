package http

import (
	"bytes"
	"image"
	"image/png"
	"mime/multipart"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFileHandlerUploadImage_BitsUT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	t.Run("成功上传图片并返回可访问相对 URL", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/files/upload", NewFileHandler().UploadImage)

		body, contentType := multipartBody(t, "file", "avatar.png", pngBytes(t))
		w := httptest.NewRecorder()
		req := httptest.NewRequest(nethttp.MethodPost, "/api/files/upload", body)
		req.Header.Set("Content-Type", contentType)
		router.ServeHTTP(w, req)

		if w.Code != nethttp.StatusOK {
			t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"url":"/uploads/images/`)) {
			t.Fatalf("body should contain upload url: %s", w.Body.String())
		}
		entries, err := os.ReadDir(filepath.Join("data", "uploads", "images"))
		if err != nil {
			t.Fatalf("read upload dir: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("uploaded files = %d, want 1", len(entries))
		}
	})

	t.Run("非图片上传返回 bad_request", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/files/upload", NewFileHandler().UploadImage)

		body, contentType := multipartBody(t, "file", "note.txt", []byte("not image"))
		w := httptest.NewRecorder()
		req := httptest.NewRequest(nethttp.MethodPost, "/api/files/upload", body)
		req.Header.Set("Content-Type", contentType)
		router.ServeHTTP(w, req)

		if w.Code != nethttp.StatusOK {
			t.Fatalf("status = %d", w.Code)
		}
		if !bytes.Contains(w.Body.Bytes(), []byte(`"bad_request"`)) {
			t.Fatalf("body should contain bad_request: %s", w.Body.String())
		}
	})
}

func TestImageExt_BitsUT(t *testing.T) {
	tests := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
		"image/webp": ".webp",
		"image/bmp":  ".img",
	}
	for mimeType, want := range tests {
		if got := imageExt(mimeType); got != want {
			t.Fatalf("imageExt(%q)=%q, want %q", mimeType, got, want)
		}
	}
}

func multipartBody(t *testing.T, field, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}
