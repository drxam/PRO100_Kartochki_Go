package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// makeServer возвращает gin-роутер с одной /x ручкой, защищённой rate-limiter'ом.
func makeServer(rl *RateLimiter) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func hit(r *gin.Engine, ip string) int {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = ip + ":1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestRateLimiter_AllowsBurst(t *testing.T) {
	// 1 RPS, burst=3 — три запроса подряд должны пройти.
	rl := NewRateLimiter(rate.Limit(1), 3, 1*time.Minute)
	srv := makeServer(rl)

	for i := 0; i < 3; i++ {
		require.Equal(t, http.StatusOK, hit(srv, "1.2.3.4"),
			"запрос #%d должен пройти в пределах burst", i+1)
	}
}

func TestRateLimiter_RejectsBeyondBurst(t *testing.T) {
	// burst=2, четвёртый и далее за миг — 429.
	rl := NewRateLimiter(rate.Limit(0.001), 2, 1*time.Minute)
	srv := makeServer(rl)

	require.Equal(t, http.StatusOK, hit(srv, "1.2.3.4"))
	require.Equal(t, http.StatusOK, hit(srv, "1.2.3.4"))
	require.Equal(t, http.StatusTooManyRequests, hit(srv, "1.2.3.4"))
	require.Equal(t, http.StatusTooManyRequests, hit(srv, "1.2.3.4"))
}

func TestRateLimiter_PerIPIsolation(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(0.001), 1, 1*time.Minute)
	srv := makeServer(rl)

	// IP A исчерпал свой bucket.
	require.Equal(t, http.StatusOK, hit(srv, "1.1.1.1"))
	require.Equal(t, http.StatusTooManyRequests, hit(srv, "1.1.1.1"))

	// IP B имеет свой собственный bucket — должен проходить.
	require.Equal(t, http.StatusOK, hit(srv, "2.2.2.2"))
}

func TestRateLimiter_RetryAfterHeader(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(0.001), 1, 1*time.Minute)
	srv := makeServer(rl)
	hit(srv, "1.2.3.4") // расходуем

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	require.Equal(t, "60", w.Header().Get("Retry-After"))
}

func TestRateLimiter_Refills(t *testing.T) {
	// 20 RPS = 1 токен каждые 50 ms; burst=1 → за 60ms должно пополниться.
	rl := NewRateLimiter(rate.Limit(20), 1, 1*time.Minute)
	srv := makeServer(rl)

	require.Equal(t, http.StatusOK, hit(srv, "1.2.3.4"))
	require.Equal(t, http.StatusTooManyRequests, hit(srv, "1.2.3.4"))
	time.Sleep(60 * time.Millisecond)
	require.Equal(t, http.StatusOK, hit(srv, "1.2.3.4"),
		"после паузы должен пополниться один токен")
}
