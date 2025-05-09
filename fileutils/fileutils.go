package fileutils

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/tracing"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultUploadDir = "uploads"
	DefaultMaxSize   = 20 << 20 // 20 MB
)

const (
	ContentTypeJPEG     = "image/jpeg"
	ContentTypePNG      = "image/png"
	ContentTypePDF      = "application/pdf"
	ContentTypeOctetStr = "application/octet-stream"
)

type FileType string

const (
	TypeImage   FileType = "image"
	TypePDF     FileType = "pdf"
	TypeGeneric FileType = "generic"
)

type FileConfig struct {
	UploadDir         string
	MaxSize           int64
	AllowedExtensions string
	EntityType        string
}

func DefaultConfig() *FileConfig {
	return &FileConfig{
		UploadDir:         DefaultUploadDir,
		MaxSize:           DefaultMaxSize,
		AllowedExtensions: ".jpg,.jpeg,.png,.pdf",
		EntityType:        "default",
	}
}

func (c *FileConfig) SetAllowedImageExtension() *FileConfig {
	c.AllowedExtensions = ".jpg,.jpeg,.png,.gif,.webp"
	return c
}

func (c *FileConfig) SetAllowedDocumentExtensions() *FileConfig {
	c.AllowedExtensions = ".pdf,.doc,.docx,.txt"
	return c
}
func StoreFile(ctx context.Context, file io.Reader, filename string, config *FileConfig) (string, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "FileUtils.StoreFile")
	defer span.End()

	if config == nil {
		config = DefaultConfig()
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if config.AllowedExtensions != "" && !strings.Contains(config.AllowedExtensions, ext) {
		return "", fmt.Errorf("unsupported file format: %s, allowed formats: %s", ext, config.AllowedExtensions)
	}

	dirPath := filepath.Join(config.UploadDir, config.EntityType)
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"path":  dirPath,
		}).ErrorWithCtx(ctx, "[FileUtils.StoreFile] failed to create directory")
		return "", err
	}

	uniqueFilename := fmt.Sprintf("%s-%s%s", config.EntityType, uuid.New().String(), ext)
	filePath := filepath.Join(dirPath, uniqueFilename)

	f, err := os.Create(filePath)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"filename": filePath,
		}).ErrorWithCtx(ctx, "[FileUtils.StoreFile] failed to create file")
		return "", err
	}
	defer f.Close()

	_, err = io.Copy(f, file)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"filename": filePath,
		}).ErrorWithCtx(ctx, "[FileUtils.StoreFile] failed to write to file")
		os.Remove(filePath)
		return "", err
	}

	return filePath, nil
}

func OpenFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "FileUtils.OpenFile")
	defer span.End()

	if filePath == "" {
		return nil, errors.New("file path is empty")
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"filepath": filePath,
		}).ErrorWithCtx(ctx, "[FileUtils.OpenFile] failed to open file")
		return nil, err
	}

	return file, nil

}

func DeleteFile(ctx context.Context, filePath string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "FileUtils.DeleteFile")
	defer span.End()

	if filePath == "" {
		return errors.New("file path is empty")
	}

	err := os.Remove(filePath)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"filepath": filePath,
		}).ErrorWithCtx(ctx, "[FileUtils.DeleteFile] failed to delete file")
		return err
	}

	return nil
}

func GetContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return ContentTypeJPEG
	case ".png":
		return ContentTypePNG
	case ".pdf":
		return ContentTypePDF
	default:
		return ContentTypeOctetStr
	}
}

func GetFileTypeFromContentType(contentType string) FileType {
	switch contentType {
	case ContentTypeJPEG, ContentTypePNG:
		return TypeImage
	case ContentTypePDF:
		return TypePDF
	default:
		return TypeGeneric
	}
}

func ValidateFileHeader(header *multipart.FileHeader, config *FileConfig) error {
	if config == nil {
		config = DefaultConfig()
	}

	if header.Size > config.MaxSize {
		return fmt.Errorf("file size exceeds the limit of %d bytes", config.MaxSize)
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if config.AllowedExtensions != "" && !strings.Contains(config.AllowedExtensions, ext) {
		return fmt.Errorf("unsupported file format: %s, allowed formats: %s", ext, config.AllowedExtensions)
	}

	return nil
}

func ValidateContentType(contentType string, allowedTypes []string) error {
	if len(allowedTypes) == 0 {
		return nil
	}

	for _, allowedType := range allowedTypes {
		if contentType == allowedType || strings.HasPrefix(contentType, allowedType) {
			return nil
		}
	}

	return fmt.Errorf("content type %s is not allowed, allowed types: %v", contentType, allowedTypes)
}

func GetFilename(filePath string) string {
	return filepath.Base(filePath)
}

func CreateTempFile(ctx context.Context, prefix string, data []byte) (string, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "FileUtils.CreateTempFile")
	defer span.End()

	tempFile, err := os.CreateTemp("", prefix)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"prefix": prefix,
		}).ErrorWithCtx(ctx, "[FileUtils.CreateTempFile] Failed to create temp file")
		return "", err
	}
	defer tempFile.Close()

	if _, err := tempFile.Write(data); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"path":  tempFile.Name(),
		}).ErrorWithCtx(ctx, "[FileUtils.CreateTempFile] Failed to write to temp file")
		return "", err
	}

	return tempFile.Name(), nil
}

func GetFileExtension(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}

func IsImage(filename string) bool {
	ext := GetFileExtension(filename)
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp"
}

func IsPDF(filename string) bool {
	ext := GetFileExtension(filename)
	return ext == ".pdf"
}

func GetFileSize(filePath string) (int64, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), nil
}

func CopyFile(ctx context.Context, src, dst string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "FileUtils.CopyFile")
	defer span.End()

	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destDir := filepath.Dir(dst)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func IsFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func CreateDirectoryIfNotExists(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return os.MkdirAll(dirPath, 0755)
	}
	return nil
}

func ReadFileToBuffer(ctx context.Context, filePath string) ([]byte, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "FileUtils.ReadFileToBuffer")
	defer span.End()

	return os.ReadFile(filePath)
}

func DownloadFile(ctx context.Context, url, destPath string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "FileUtils.DownloadFile")
	defer span.End()

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
