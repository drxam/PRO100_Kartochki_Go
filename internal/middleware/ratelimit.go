package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter — потокобезопасный per-IP token-bucket лимитер.
// Реализует требование ТЗ §4.1 «реализация механизмов ограничения частоты
// запросов (rate limiting)».
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*ipBucket
	rate    rate.Limit
	burst   int
	idleTTL time.Duration
}

type ipBucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter создаёт лимитер с заданной скоростью пополнения и burst.
// Запускает фоновую горутину очистки давно неактивных IP (раз в idleTTL).
func NewRateLimiter(r rate.Limit, burst int, idleTTL time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*ipBucket),
		rate:    r,
		burst:   burst,
		idleTTL: idleTTL,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if b, ok := rl.clients[ip]; ok {
		b.lastSeen = time.Now()
		return b.limiter
	}
	l := rate.NewLimiter(rl.rate, rl.burst)
	rl.clients[ip] = &ipBucket{limiter: l, lastSeen: time.Now()}
	return l
}

func (rl *RateLimiter) cleanupLoop() {
	t := time.NewTicker(rl.idleTTL)
	defer t.Stop()
	for range t.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, b := range rl.clients {
			if now.Sub(b.lastSeen) > rl.idleTTL {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware — gin handler. Пропускает запрос только если из bucket'а IP
// есть свободный токен; иначе возвращает 429 Too Many Requests.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rl.get(ip).Allow() {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "слишком много запросов, попробуйте позже",
				},
			})
			return
		}
		c.Next()
	}
}
