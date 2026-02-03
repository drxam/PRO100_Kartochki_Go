package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Коды ошибок по спецификации
const (
	CodeValidationError   = "VALIDATION_ERROR"
	CodeBadRequest       = "BAD_REQUEST"
	CodeUnauthorized      = "UNAUTHORIZED"
	CodeInvalidToken      = "INVALID_TOKEN"
	CodeForbidden         = "FORBIDDEN"
	CodeNotFound          = "NOT_FOUND"
	CodeAlreadyExists     = "ALREADY_EXISTS"
	CodeInternalError     = "INTERNAL_SERVER_ERROR"
)

// ErrorPayload структура ошибки по спецификации
type ErrorPayload struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func errorResponse(c *gin.Context, status int, code, message string, details interface{}) {
	c.JSON(status, gin.H{"error": ErrorPayload{Code: code, Message: message, Details: details}})
}

func JSON(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func BadRequest(c *gin.Context, message string, details interface{}) {
	errorResponse(c, http.StatusBadRequest, CodeValidationError, message, details)
}

func BadRequestSimple(c *gin.Context, message string) {
	errorResponse(c, http.StatusBadRequest, CodeBadRequest, message, nil)
}

func Unauthorized(c *gin.Context, message string) {
	errorResponse(c, http.StatusUnauthorized, CodeUnauthorized, message, nil)
}

func InvalidToken(c *gin.Context, message string) {
	errorResponse(c, http.StatusUnauthorized, CodeInvalidToken, message, nil)
}

func Forbidden(c *gin.Context, message string) {
	errorResponse(c, http.StatusForbidden, CodeForbidden, message, nil)
}

func NotFound(c *gin.Context, message string) {
	errorResponse(c, http.StatusNotFound, CodeNotFound, message, nil)
}

func Conflict(c *gin.Context, message string) {
	errorResponse(c, http.StatusConflict, CodeAlreadyExists, message, nil)
}

func InternalError(c *gin.Context, message string) {
	errorResponse(c, http.StatusInternalServerError, CodeInternalError, message, nil)
}
