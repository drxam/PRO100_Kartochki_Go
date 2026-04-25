package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	RequestIDKey    = "request_id"
	requestIDHeader = "X-Request-ID"
)

// RequestID генерирует UUID-идентификатор запроса (или подхватывает входящий
// X-Request-ID, если клиент его прислал) и кладёт его в gin-контекст и в
// заголовок ответа. Идентификатор затем используется в логах и audit-записях
// для сквозной трассировки.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(RequestIDKey, id)
		c.Writer.Header().Set(requestIDHeader, id)
		c.Next()
	}
}

// GetRequestID возвращает идентификатор запроса из контекста (после RequestID middleware).
func GetRequestID(c *gin.Context) string {
	v, _ := c.Get(RequestIDKey)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
