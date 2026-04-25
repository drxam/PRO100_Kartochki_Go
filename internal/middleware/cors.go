package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS возвращает middleware, разрешающий кросс-доменные запросы.
//
// allowedOrigins — список разрешённых origin'ов (например, "https://app.example.com").
// Если список пуст или содержит "*" — разрешены любые источники (поведение по
// умолчанию для разработки и для нативного iOS-клиента, у которого нет
// CORS-проверки в принципе).
//
// Когда конкретные origin'ы перечислены, заголовок Access-Control-Allow-Origin
// отдаётся персонально под пришедший Origin (если он в списке), что нужно для
// корректной работы запросов с credentials.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll := len(allowedOrigins) == 0
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
		}
		allowed[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		switch {
		case allowAll:
			c.Header("Access-Control-Allow-Origin", "*")
		case origin != "":
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
