package main

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"photo-viewer-server/internal/config"
	"photo-viewer-server/internal/http-server/handlers/auth"
	"photo-viewer-server/internal/http-server/handlers/url/healthcheck"
	"photo-viewer-server/internal/http-server/handlers/url/remove"
	"photo-viewer-server/internal/http-server/handlers/url/update"
	"photo-viewer-server/internal/http-server/handlers/url/upload"
	"photo-viewer-server/internal/http-server/handlers/url/view"
	emptytokenmw "photo-viewer-server/internal/http-server/middleware/empty-token-mw"
	jwtmiddleware "photo-viewer-server/internal/http-server/middleware/jwt-middleware"
	mwlogger "photo-viewer-server/internal/http-server/middleware/mw-logger"
	ratelimitmw "photo-viewer-server/internal/http-server/middleware/rate-limit-mw"
	"photo-viewer-server/internal/lib/mail"
	ratelimiter "photo-viewer-server/internal/lib/rate-limiter"
	"photo-viewer-server/internal/service"
	"photo-viewer-server/internal/storage/minio"
	"photo-viewer-server/internal/storage/postrgesql"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

const (
	appEnvDev  = "dev"
	appEnvProd = "prod"
)

func main() {
	cfg := config.MustLoad()

	isDevEnv := cfg.AppEnv == appEnvDev

	log := setupLogger(cfg.AppEnv)

	log.Info("photo viewer server started", slog.String("env", cfg.AppEnv))
	log.Debug("debug messages are enabled")
	log.Debug("storage configuration",
		slog.String("db_host", cfg.DatabaseHost), slog.Int("db_port", cfg.DatabasePort),
		slog.String("storage_host", cfg.StorageHost), slog.Int("storage_port", cfg.StoragePort),
	)

	metadataStorage, err := postrgesql.New(
		cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseName, cfg.DatabaseUser, cfg.DatabasePassword,
	)

	if err != nil {
		log.Error("failed init postrgesql")
		os.Exit(1)
	}

	storage, err := minio.New(cfg.StorageHost, cfg.StoragePort, cfg.StorageUser, cfg.StoragePassword, false)
	if err != nil {
		log.Error("failed init minio storage")
		os.Exit(1)
	}

	photoService := service.NewPhotoService(log, metadataStorage, storage, cfg.PhotosBucketName, metadataStorage)

	mailService := mail.NewMailService(cfg)

	userService := service.NewUserService(log, &mailService, metadataStorage, metadataStorage)

	healthcheckService := service.NewHealthcheckService([]service.Healthchecker{ storage, metadataStorage })

	authRateLimit := 5
	if isDevEnv {
		authRateLimit = 100
	}

	rateLimits := map[string]ratelimitmw.RateLimit{
		"/api/v1/auth/login": {Limit: authRateLimit, Window: time.Minute},
		"/api/v1/auth/register": {Limit: authRateLimit, Window: time.Minute},
		"/api/v1/auth/refresh": {Limit: authRateLimit, Window: time.Minute},
	}

	rateLimiter := ratelimiter.NewInMemoryRateLimiter()

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(mwlogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.RealIP)
	router.Use(middleware.URLFormat)
	router.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			return true
		},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	router.Route("/api/v1", func(apiv1Router chi.Router) {
		apiv1Router.Group(func(r chi.Router) {
			r.Get("/health", healthcheck.Healthcheck(log, healthcheckService))
			r.Get("/photos", view.ViewPhotos(log, photoService))
			r.Get("/photo/{photo_uuid}", view.ViewPhoto(log, photoService))
			r.Get("/api/v1/photo/{photo_uuid}/info", view.ViewPhotoInfo(log, photoService))
		})

		apiv1Router.Group(func(r chi.Router) {
			r.Use(emptytokenmw.New(cfg.JwtAccessSecret))
			r.Use(ratelimitmw.New(log, rateLimiter, rateLimits))

			r.Post("/auth/register", auth.RegisterUser(log, userService))
			r.Post("/auth/login", auth.LoginUser(log, "/api/v1", cfg.JwtAccessSecret, cfg.JwtRefreshSecret, userService, isDevEnv))
			r.Post("/auth/refresh", auth.RefreshUser(log, "/api/v1", cfg.JwtAccessSecret, cfg.JwtRefreshSecret, userService, isDevEnv))
		})

		apiv1Router.Group(func(r chi.Router) {
			r.Use(jwtmiddleware.New(cfg.JwtAccessSecret))

			r.Get("/auth/me", auth.GetMe(log, userService))
			r.Post("/photos", upload.UploadPhoto(log, photoService))
			r.Delete("/photo/{photo_uuid}", remove.RemovePhoto(log, photoService))
			r.Patch("/photo/{photo_uuid}", update.UpdatePhoto(log, photoService))
		})
	})

	log.Info("starting server", slog.String("host", cfg.Host), slog.Int("port", cfg.Port))

	srv := &http.Server{
		Addr:         net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Handler:      router,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.Timeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	}

	log.Info("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case appEnvDev:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case appEnvProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}),
		)
	}

	return log
}
