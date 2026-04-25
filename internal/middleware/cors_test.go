package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func makeCORSServer(origins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(origins))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func corsRequest(r *gin.Engine, method, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/x", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCORS_AllowAll_WhenEmpty(t *testing.T) {
	w := corsRequest(makeCORSServer(nil), http.MethodGet, "https://anywhere.com")
	require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_AllowAll_WhenStarInList(t *testing.T) {
	w := corsRequest(makeCORSServer([]string{"*"}), http.MethodGet, "https://anywhere.com")
	require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_AllowsListedOrigin(t *testing.T) {
	w := corsRequest(
		makeCORSServer([]string{"https://app.example.com", "https://admin.example.com"}),
		http.MethodGet, "https://app.example.com")
	require.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "Origin", w.Header().Get("Vary"))
}

func TestCORS_BlocksUnlistedOrigin(t *testing.T) {
	w := corsRequest(
		makeCORSServer([]string{"https://app.example.com"}),
		http.MethodGet, "https://evil.com")
	require.Empty(t, w.Header().Get("Access-Control-Allow-Origin"),
		"для не-разрешённого origin заголовок не должен ставиться")
}

func TestCORS_PreflightReturns204(t *testing.T) {
	w := corsRequest(makeCORSServer([]string{"https://app.example.com"}),
		http.MethodOptions, "https://app.example.com")
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
}
