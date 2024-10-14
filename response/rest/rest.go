package rest

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/nocturna-ta/golib/context"
	"github.com/nocturna-ta/golib/custerr"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/response"

	"io"
	"net/http"
)

var (
	mapFiberErrorCode = map[int]error{
		404: response.ErrNotFound,
		408: response.ErrTimeoutError,
		413: response.ErrRequestTooLarge,
	}
)

func GetErrorCode(err error) int {
	err = getErrType(err)

	switch err {
	case response.ErrBadRequest:
		return http.StatusBadRequest
	case response.ErrForbiddenResource:
		return http.StatusForbidden
	case response.ErrNotFound:
		return http.StatusNotFound
	case response.ErrInternalServerError:
		return http.StatusInternalServerError
	case response.ErrTimeoutError:
		return http.StatusRequestTimeout
	case response.ErrUnauthorized:
		return http.StatusUnauthorized
	case response.ErrConflict:
		return http.StatusConflict
	case response.ErrRequestTooLarge:
		return http.StatusRequestEntityTooLarge
	case nil:
		return http.StatusOK
	default:
		return http.StatusInternalServerError
	}
}

func getErrType(err error) error {
	switch e := err.(type) {
	case *custerr.ErrChain:
		errType := e.Type
		if errType != nil {
			err = errType
		}
	}
	return err
}

func getCustErr(err error) *custerr.ErrChain {
	var res *custerr.ErrChain
	switch e := err.(type) {
	case *custerr.ErrChain:
		res = e
	case *fiber.Error:
		fiberErr := e
		res = &custerr.ErrChain{
			Message: fiberErr.Message,
			Code:    fiberErr.Code,
			Type:    mapFiberErrorCode[fiberErr.Code],
		}
	default:
		// default to internal server error
		res = &custerr.ErrChain{
			Message: "internal server error",
			Cause:   err,
			Code:    http.StatusInternalServerError,
			Type:    response.ErrInternalServerError,
		}
	}
	return res
}

type Response interface {
	JSONResponse | AttachmentResponse
	Send(c *fiber.Ctx) error
}

type JSONResponse struct {
	Data    any            `json:"data,omitempty"`
	Code    int            `json:"code,omitempty"`
	Message string         `json:"message,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
}

type AttachmentResponse struct {
	File        io.Reader
	FileName    string
	ContentType string
}

type ErrorResponse struct {
	ErrorCode    int `json:"error_code,omitempty"`
	ErrorMessage any `json:"error_message,omitempty"`
}

func NewJSONResponse() *JSONResponse {
	return &JSONResponse{Code: http.StatusOK}
}

func (r *JSONResponse) SetData(data any) *JSONResponse {
	r.Data = data
	return r
}

func (r *JSONResponse) SetCode(code int) *JSONResponse {
	r.Code = code
	return r
}

func (r *JSONResponse) SetMessage(msg string) *JSONResponse {
	r.Message = msg
	return r
}

func (r *JSONResponse) SetError(err error) *JSONResponse {
	custErr := getCustErr(err)
	r.Code = GetErrorCode(custErr)

	r.Error = &ErrorResponse{
		ErrorCode:    custErr.Code,
		ErrorMessage: custErr.Error(),
	}
	return r
}

func (r JSONResponse) Send(c *fiber.Ctx) error {
	requestId := context.ReadRequestId(c.Context())

	c.Response().SetStatusCode(r.Code)
	c.Response().Header.Add("Content-Type", "application/json")

	err := json.NewEncoder(c.Response().BodyWriter()).Encode(r)
	if err != nil {
		log.WithFields(log.Fields{
			"error":      err.Error(),
			"request-id": requestId,
		}).Error("[JSONResponse.Send] Error encoding response")
		return err
	}

	return nil
}

func NewAttachmentResponse() *AttachmentResponse {
	return &AttachmentResponse{}
}

func (ar *AttachmentResponse) SetFile(reader io.Reader) *AttachmentResponse {
	ar.File = reader
	return ar
}

func (ar *AttachmentResponse) SetFileName(fileName string) *AttachmentResponse {
	ar.FileName = fileName
	return ar
}

func (ar *AttachmentResponse) SetContentType(contentType string) *AttachmentResponse {
	ar.ContentType = contentType
	return ar
}

func (ar AttachmentResponse) Send(c *fiber.Ctx) error {
	c.Response().SetStatusCode(http.StatusOK)
	if ar.ContentType != "" {
		c.Response().Header.Add("Content-Type", ar.ContentType)
	} else {
		c.Response().Header.Add("Content-Type", "application/octet-stream")
	}

	c.Response().Header.Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", ar.FileName))

	if _, err := io.Copy(c.Response().BodyWriter(), ar.File); err != nil {
		return err
	}
	return nil
}
