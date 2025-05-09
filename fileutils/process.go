package fileutils

import (
	"context"
	"errors"
	"fmt"
	"github.com/nocturna-ta/golib/custerr"
	"github.com/nocturna-ta/golib/http/filehandler"
	"github.com/nocturna-ta/golib/response"
	"io"
	"mime/multipart"
)

type FileUploadConfig struct {
	FieldName   string
	Required    bool
	UploadFunc  func() *filehandler.UploadOptions
	ErrorMsgs   map[error]string
	DefaultCode int
}

type UploadedFile struct {
	File             io.ReadCloser
	OriginalFilename string
	FilePath         string
	ContentType      string
	Size             int64
}

func ProcessFileUploads(ctx context.Context, form *multipart.Form, configs []FileUploadConfig) (map[string]UploadedFile, error) {
	result := make(map[string]UploadedFile)

	for _, config := range configs {
		uploadOptions := config.UploadFunc()
		uploadOptions.FieldName = config.FieldName

		uploadResult, err := filehandler.UploadFile(ctx, form, uploadOptions)
		if err != nil {
			if errors.Is(err, filehandler.ErrNoFile) && !config.Required {
				continue
			}
			return nil, MapFileUploadError(err, config)
		}

		file, err := OpenFile(ctx, uploadResult.FilePath)
		if err != nil {
			return nil, &custerr.ErrChain{
				Message: fmt.Sprintf("Failed to open uploaded file: %s", config.FieldName),
				Code:    500,
				Type:    response.ErrInternalServerError,
				Cause:   err,
			}
		}

		result[config.FieldName] = UploadedFile{
			File:             file,
			OriginalFilename: uploadResult.OriginalFilename,
			FilePath:         uploadResult.FilePath,
			ContentType:      uploadResult.ContentType,
			Size:             uploadResult.Size,
		}
	}

	return result, nil
}

func MapFileUploadError(err error, config FileUploadConfig) *custerr.ErrChain {
	var errorMsg string
	var errorCode int = config.DefaultCode
	if errorCode == 0 {
		errorCode = 400
	}

	if config.ErrorMsgs != nil {
		if msg, exists := config.ErrorMsgs[err]; exists {
			errorMsg = msg
		}
	}

	if errorMsg == "" {
		switch {
		case errors.Is(err, filehandler.ErrNoFile):
			errorMsg = fmt.Sprintf("No file provided for %s", config.FieldName)
		case errors.Is(err, filehandler.ErrFileTooLarge):
			errorMsg = fmt.Sprintf("File size exceeds maximum allowed size for %s", config.FieldName)
		case errors.Is(err, filehandler.ErrInvalidFileFormat):
			errorMsg = fmt.Sprintf("Invalid file format for %s", config.FieldName)
		default:
			errorMsg = fmt.Sprintf("Failed to process uploaded file: %s", config.FieldName)
			errorCode = 500
		}
	}

	return &custerr.ErrChain{
		Message: errorMsg,
		Code:    errorCode,
		Type:    response.ErrBadRequest,
		Cause:   err,
	}
}

func CloseFiles(files map[string]UploadedFile) {
	for _, fileInfo := range files {
		if fileInfo.File != nil {
			fileInfo.File.Close()
		}
	}
}

func CloseReadClosers(closers ...io.ReadCloser) {
	for _, closer := range closers {
		if closer != nil {
			closer.Close()
		}
	}
}
