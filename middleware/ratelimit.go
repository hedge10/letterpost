package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/tomasen/realip"
	"golang.org/x/time/rate"
)

// RateLimitConfig holds the rate limiter settings.
type RateLimitConfig struct {
	RPS     float64 `env:"RPS" envDefault:"2"`
	Burst   int     `env:"BURST" envDefault:"4"`
	Enabled bool    `env:"ENABLED" envDefault:"true"`
}

// RateLimit wraps next in a rate limit per client IP. Requests over the
// limit go to exceeded. The client IP comes from realip, which trusts
// X-Forwarded-For: the server must only be reachable through a proxy that
// sets it.
func RateLimit(cfg RateLimitConfig, next http.Handler, exceeded http.HandlerFunc) http.Handler {
	if !cfg.Enabled {
		return next
	}

	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()

			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realip.FromRequest(r)
		mu.Lock()

		if _, found := clients[ip]; !found {
			clients[ip] = &client{
				limiter: rate.NewLimiter(rate.Limit(cfg.RPS), cfg.Burst),
			}
		}

		clients[ip].lastSeen = time.Now()

		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			exceeded(w, r)
			return
		}

		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
