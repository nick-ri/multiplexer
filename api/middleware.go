package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/NickRI/multiplexer/limit"
)

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s", r.Method, r.URL.String())
		next.ServeHTTP(w, r)
	})
}

func RateLimitMiddleware(limiter limit.Limiter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				w.WriteHeader(http.StatusTooManyRequests)
				fmt.Fprintln(w, "Too many requests")
			}
			next.ServeHTTP(w, r)
		})
	}
}
