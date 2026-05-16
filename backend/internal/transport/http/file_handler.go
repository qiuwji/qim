package http

import (
	"crypto/rand"
	"encoding/hex"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"qim/internal/pkg/apperr"
	"qim/internal/pkg/resp"

	"github.com/gin-gonic/gin"
)

const (
	maxImageSize   = 10 << 20
	imageUploadDir = "data/uploads/images"
	imageURLPrefix = "/uploads/images"
)

type FileHandler struct{}

func NewFileHandler() *FileHandler {
	return &FileHandler{}
}

type UploadFileDTO struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

func (h *FileHandler) UploadImage(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(maxImageSize); err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		resp.Fail(c, badRequest(err))
		return
	}
	defer file.Close()

	if header.Size > maxImageSize {
		resp.Fail(c, apperr.New(apperr.CodeBadRequest, "image is too large"))
		return
	}

	mimeType, err := detectImageMime(file)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if _, err := file.Seek(0, 0); err != nil {
		resp.Fail(c, internalError(err))
		return
	}

	if err := os.MkdirAll(imageUploadDir, 0755); err != nil {
		resp.Fail(c, internalError(err))
		return
	}
	filename := uniqueImageFilename(header, mimeType)
	dst := filepath.Join(imageUploadDir, filename)
	if err := c.SaveUploadedFile(header, dst); err != nil {
		resp.Fail(c, internalError(err))
		return
	}

	resp.OK(c, UploadFileDTO{
		URL:      imageURLPrefix + "/" + filename,
		Filename: filename,
		Size:     header.Size,
		MimeType: mimeType,
	})
}

func detectImageMime(file multipart.File) (string, error) {
	var header [512]byte
	n, err := file.Read(header[:])
	if err != nil {
		return "", internalError(err)
	}
	mimeType := http.DetectContentType(header[:n])
	if !strings.HasPrefix(mimeType, "image/") {
		return "", apperr.New(apperr.CodeBadRequest, "only image upload is supported")
	}
	return mimeType, nil
}

func uniqueImageFilename(header *multipart.FileHeader, mimeType string) string {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = imageExt(mimeType)
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().Format("20060102150405000000000") + ext
	}
	return time.Now().Format("20060102150405000000000") + "-" + hex.EncodeToString(b[:]) + ext
}

func imageExt(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".img"
	}
}
