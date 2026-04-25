package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// auditLogger — общий zap-логгер для аудита чувствительных операций
// (вход/выход, регистрация, сброс пароля, блокировка, смена ролей).
// Все записи помечены полем audit=true для фильтрации в SIEM/grep.
var auditLogger *zap.Logger = zap.NewNop()

// SetAuditLogger подключает реальный логгер; вызывается в main.go.
func SetAuditLogger(l *zap.Logger) {
	auditLogger = l
}

// Audit пишет аудитное событие. event — короткое имя (auth.login.success и т.п.),
// extra — дополнительные структурированные поля. client_ip берётся из gin.Context.
func Audit(c *gin.Context, event string, extra ...zap.Field) {
	fields := make([]zap.Field, 0, len(extra)+3)
	fields = append(fields,
		zap.Bool("audit", true),
		zap.String("event", event),
		zap.String("client_ip", c.ClientIP()),
	)
	fields = append(fields, extra...)
	auditLogger.Info("audit", fields...)
}

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
