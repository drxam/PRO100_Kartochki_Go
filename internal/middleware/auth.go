package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pro100kartochki/mozgoemka/pkg/jwt"
)

const UserIDKey = "user_id"
const UserRoleKey = "user_role"

// Auth извлекает JWT из заголовка Authorization и кладёт user_id, user_role в контекст.
func Auth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "отсутствует токен"})
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "неверный формат токена"})
			return
		}
		claims, err := jwtManager.ParseAccessToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "недействительный токен"})
			return
		}
		c.Set(UserIDKey, claims.UserID)
		c.Set(UserRoleKey, claims.Role)
		c.Next()
	}
}

// GetUserID возвращает user_id из контекста (после Auth middleware).
func GetUserID(c *gin.Context) int {
	id, _ := c.Get(UserIDKey)
	if id == nil {
		return 0
	}
	return id.(int)
}
