package filehandler

import (
	"context"
	"errors"
	"fmt"
	"github.com/nocturna-ta/golib/fileutils"
	"github.com/nocturna-ta/golib/http"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/tracing"
	"io"
	"mime/multipart"
	"strings"
)

var (
	ErrNoFile            = errors.New("no file provided")
	ErrFileTooLarge      = errors.New("file size exceeds maximum allowed size")
	ErrInvalidFileFormat = errors.New("invalid file format")
	ErrFileUploadFailed  = errors.New("file upload failed")
)

type UploadOptions struct {
	FileConfig *fileutils.FileConfig
	FieldName  string
	Validator  func(fileHeader *multipart.FileHeader) error
}

func DefaultUploadOptions() *UploadOptions {
	return &UploadOptions{
		FileConfig: fileutils.DefaultConfig(),
		FieldName:  "file",
		Validator:  nil,
	}
}

func ImageUploadOptions() *UploadOptions {
	cfg := fileutils.DefaultConfig()
	cfg.SetAllowedImageExtension()
	cfg.EntityType = "images"

	return &UploadOptions{
		FileConfig: cfg,
		FieldName:  "image",
		Validator:  nil,
	}
}

func DocumentUploadOptions() *UploadOptions {
	cfg := fileutils.DefaultConfig()
	cfg.SetAllowedDocumentExtensions()
	cfg.EntityType = "documents"

	return &UploadOptions{
		FileConfig: cfg,
		FieldName:  "document",
		Validator:  nil,
	}
}

type UploadResult struct {
	FilePath         string
	OriginalFilename string
	Size             int64
	ContentType      string
}

func UploadFile(ctx context.Context, form *multipart.Form, options *UploadOptions) (*UploadResult, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "FileHandler.UploadFile")
	defer span.End()

	if options == nil {
		options = DefaultUploadOptions()
	}

	files := form.File[options.FieldName]
	if len(files) == 0 {
		return nil, ErrNoFile
	}

	fileHeader := files[0]

	if err := fileutils.ValidateFileHeader(fileHeader, options.FileConfig); err != nil {
		if err.Error() == fmt.Sprintf("file size exceeds maximum allowed size of %d bytes", options.FileConfig.MaxSize) {
			return nil, ErrFileTooLarge
		}
		return nil, ErrInvalidFileFormat
	}

	if options.Validator != nil {
		if err := options.Validator(fileHeader); err != nil {
			return nil, err
		}
	}

	file, err := fileHeader.Open()
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"filename": fileHeader.Filename,
		}).ErrorWithCtx(ctx, "[FileHandler.UploadFile] Failed to open uploaded file")
		return nil, err
	}
	defer file.Close()

	// Store file
	filePath, err := fileutils.StoreFile(ctx, file, fileHeader.Filename, options.FileConfig)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"filename": fileHeader.Filename,
		}).ErrorWithCtx(ctx, "[FileHandler.UploadFile] Failed to store file")
		return nil, ErrFileUploadFailed
	}

	contentType := fileutils.GetContentType(fileHeader.Filename)

	return &UploadResult{
		FilePath:         filePath,
		OriginalFilename: fileHeader.Filename,
		Size:             fileHeader.Size,
		ContentType:      contentType,
	}, nil
}

func GetFileFromPath(ctx context.Context, filePath string) (*http.File, string, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "FileHandler.GetFileFromPath")
	defer span.End()

	if filePath == "" {
		return nil, "", errors.New("file path is empty")
	}

	file, err := fileutils.OpenFile(ctx, filePath)
	if err != nil {
		return nil, "", err
	}

	filename := fileutils.GetFilename(filePath)
	contentType := fileutils.GetContentType(filename)

	return &http.File{
		Reader:   file,
		FileName: filename,
	}, contentType, nil
}

func PrepareAttachmentResponse(ctx context.Context, filePath string) (*http.Response[any], error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "FileHandler.PrepareAttachmentResponse")
	defer span.End()

	file, contentType, err := GetFileFromPath(ctx, filePath)
	if err != nil {
		return nil, err
	}

	response := &http.Response[any]{
		HttpCode:       200,
		ResponseHeader: make(map[string][]string),
		Method:         http.GET,
	}

	// Set proper response headers
	response.ResponseHeader.Add("Content-Type", contentType)
	response.ResponseHeader.Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.FileName))

	return response, nil
}

func ParseMultipartFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) ([]byte, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "FileHandler.ParseMultipartFile")
	defer span.End()

	buffer := make([]byte, header.Size)
	_, err := io.ReadFull(file, buffer)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"filename": header.Filename,
		}).ErrorWithCtx(ctx, "[FileHandler.ParseMultipartFile] Failed to read file data")
		return nil, err
	}

	return buffer, nil
}

func GetFileExtension(filename string) string {
	return fileutils.GetFileExtension(filename)
}

func IsAllowedExtension(ext string, allowedExtensions string) bool {
	return strings.Contains(allowedExtensions, strings.ToLower(ext))
}
