package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/pkg/jwt"
)

const (
	UserIDKey   = "user_id"
	UserRoleKey = "user_role"
)

// UserStatusChecker — минимальный интерфейс, нужный middleware для проверки
// что пользователь не удалён и не заблокирован. Реализуется UserRepository.
type UserStatusChecker interface {
	GetByID(ctx context.Context, id int) (*domain.User, error)
}

// Auth извлекает JWT, проверяет, что пользователь активен (не удалён, не заблокирован),
// и кладёт user_id и user_role в контекст.
func Auth(jwtManager *jwt.Manager, users UserStatusChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "отсутствует токен"},
			})
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "INVALID_TOKEN", "message": "неверный формат токена"},
			})
			return
		}
		claims, err := jwtManager.ParseAccessToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "INVALID_TOKEN", "message": "недействительный токен"},
			})
			return
		}

		// Проверка актуальности учётной записи: не удалена и не заблокирована.
		// GetByID уже фильтрует deleted_at IS NULL.
		u, err := users.GetByID(c.Request.Context(), claims.UserID)
		if err != nil || u == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "учётная запись недоступна"},
			})
			return
		}
		// token_version: после смены пароля или блокировки сервер инкрементирует
		// этот счётчик, что мгновенно делает все ранее выпущенные access-токены
		// недействительными (даже если их exp ещё в будущем).
		if claims.TokenVersion != u.TokenVersion {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "INVALID_TOKEN", "message": "токен отозван"},
			})
			return
		}
		if u.IsBlocked {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "учётная запись заблокирована"},
			})
			return
		}

		c.Set(UserIDKey, u.ID)
		c.Set(UserRoleKey, u.Role)
		c.Next()
	}
}

// GetUserID возвращает user_id из контекста (после Auth middleware).
func GetUserID(c *gin.Context) int {
	id, _ := c.Get(UserIDKey)
	if id == nil {
		return 0
	}
	if v, ok := id.(int); ok {
		return v
	}
	return 0
}

// GetUserRole возвращает роль пользователя из контекста.
func GetUserRole(c *gin.Context) string {
	r, _ := c.Get(UserRoleKey)
	if r == nil {
		return ""
	}
	if v, ok := r.(string); ok {
		return v
	}
	return ""
}

// RequireRole — RBAC middleware. Должен использоваться после Auth.
// Возвращает 403, если роль текущего пользователя не входит в список допустимых.
func RequireRole(allowed ...domain.UserRole) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allowedSet[string(r)] = struct{}{}
	}
	return func(c *gin.Context) {
		role := GetUserRole(c)
		if _, ok := allowedSet[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "недостаточно прав"},
			})
			return
		}
		c.Next()
	}
}
