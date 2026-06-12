package ratelimitmw

import (
	"log/slog"
	"net"
	"net/http"
	"photo-viewer-server/internal/lib/api/response"
	"photo-viewer-server/internal/lib/random"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type RateLimiter interface {
	Allow(key string, limit int, window time.Duration) bool
}

type RateLimit struct {
	Limit int
	Window time.Duration
}

func New(lg *slog.Logger, limiter RateLimiter, limits map[string]RateLimit) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := lg.With(
				slog.String("op", "ratelimitmw.RateLimitMiddleware"),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)

			rc := chi.RouteContext(r.Context())

		 	var route string
			if rc != nil {
				route = rc.RoutePattern()
			}
			if route == "" {
				route = r.URL.Path
			}

			if limit, ok := limits[route]; ok {
				host, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					host = r.RemoteAddr
				}

				key := host + ":" + route

				if !limiter.Allow(key, limit.Limit, limit.Window) {
					log.Error("too many requests", slog.String("key", key))

					duration, err := random.CryptoRandInt64(100, 1500)
					if err != nil {
						duration = 750
					}
					time.Sleep(time.Duration(duration) * time.Millisecond)

					render.Status(r, http.StatusTooManyRequests)
					render.JSON(w, r, response.Error("too many requests"))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
