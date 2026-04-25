package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func runReqID(t *testing.T, incoming string) (responseHeader, contextValue string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/x", func(c *gin.Context) {
		contextValue = GetRequestID(c)
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if incoming != "" {
		req.Header.Set("X-Request-ID", incoming)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Header().Get("X-Request-ID"), contextValue
}

func TestRequestID_GeneratedWhenAbsent(t *testing.T) {
	hdr, ctxVal := runReqID(t, "")
	require.NotEmpty(t, hdr, "должен установить X-Request-ID")
	require.Equal(t, hdr, ctxVal, "значение в ответе и в контексте должно совпадать")
	require.Len(t, hdr, 36, "ожидаем UUID v4 (36 символов)")
}

func TestRequestID_PassedThroughWhenProvided(t *testing.T) {
	hdr, ctxVal := runReqID(t, "trace-abc-123")
	require.Equal(t, "trace-abc-123", hdr)
	require.Equal(t, "trace-abc-123", ctxVal)
}

func TestGetRequestID_NoMiddlewareReturnsEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	got := ""
	r.GET("/x", func(c *gin.Context) { got = GetRequestID(c); c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
	require.Equal(t, "", got)
}
