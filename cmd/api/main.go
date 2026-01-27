package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/config"
	"github.com/mwork/mwork-api/internal/domain/admin"
	"github.com/mwork/mwork-api/internal/domain/auth"
	"github.com/mwork/mwork-api/internal/domain/casting"
	"github.com/mwork/mwork-api/internal/domain/chat"
	"github.com/mwork/mwork-api/internal/domain/content"
	"github.com/mwork/mwork-api/internal/domain/dashboard"
	"github.com/mwork/mwork-api/internal/domain/lead"
	"github.com/mwork/mwork-api/internal/domain/notification"
	"github.com/mwork/mwork-api/internal/domain/organization"
	"github.com/mwork/mwork-api/internal/domain/payment"
	"github.com/mwork/mwork-api/internal/domain/photo"
	"github.com/mwork/mwork-api/internal/domain/profile"
	"github.com/mwork/mwork-api/internal/domain/promotion"
	"github.com/mwork/mwork-api/internal/domain/response"
	"github.com/mwork/mwork-api/internal/domain/review"
	"github.com/mwork/mwork-api/internal/domain/subscription"
	uploadDomain "github.com/mwork/mwork-api/internal/domain/upload"
	"github.com/mwork/mwork-api/internal/domain/user"
	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/database"
	"github.com/mwork/mwork-api/internal/pkg/jwt"
	pkgresponse "github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/storage"
	"github.com/mwork/mwork-api/internal/pkg/upload"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logging
	setupLogger(cfg)

	log.Info().
		Str("env", cfg.Env).
		Str("port", cfg.Port).
		Msg("Starting MWork API")

	// Connect to PostgreSQL
	db, err := database.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to PostgreSQL")
	}
	defer database.ClosePostgres(db)

	// Connect to Redis (optional)
	redis, err := database.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer database.CloseRedis(redis)

	// Initialize JWT service
	jwtService := jwt.NewService(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	// Initialize R2 upload service (optional)
	uploadSvc := upload.NewService(&upload.Config{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		AccessKeySecret: cfg.R2AccessKeySecret,
		BucketName:      cfg.R2BucketName,
		PublicURL:       cfg.R2PublicURL,
	})

	// Initialize repositories
	userRepo := user.NewRepository(db)
	modelRepo := profile.NewModelRepository(db)
	employerRepo := profile.NewEmployerRepository(db)
	castingRepo := casting.NewRepository(db)
	responseRepo := response.NewRepository(db)
	photoRepo := photo.NewRepository(db)
	chatRepo := chat.NewRepository(db)
	notificationRepo := notification.NewRepository(db)
	subscriptionRepo := subscription.NewRepository(db)
	paymentRepo := payment.NewRepository(db)

	// NEW: Dashboard and Promotion repositories
	dashboardRepo := dashboard.NewRepository(db)
	promotionRepo := promotion.NewRepository(db)

	// NEW: Upload domain with 2-phase staging
	stagingDir := os.Getenv("STAGING_DIR")
	if stagingDir == "" {
		stagingDir = "./tmp/uploads"
	}
	stagingStorage, err := storage.NewLocalStorage(stagingDir, "/api/v1/files")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create staging storage")
	}
	uploadRepo := uploadDomain.NewRepository(db)
	uploadService := uploadDomain.NewService(uploadRepo, stagingStorage, nil, "/api/v1/files")
	uploadHandler := uploadDomain.NewHandler(uploadService, "/api/v1/files")

	// Initialize WebSocket hub with Redis for scalability
	chatHub := chat.NewHub(redis)
	go chatHub.Run()

	// Initialize services
	authService := auth.NewService(userRepo, jwtService, redis)
	profileService := profile.NewService(modelRepo, employerRepo, userRepo)
	castingService := casting.NewService(castingRepo, userRepo)
	responseService := response.NewService(responseRepo, castingRepo, modelRepo, employerRepo)
	photoService := photo.NewService(photoRepo, modelRepo, uploadSvc)
	chatService := chat.NewService(chatRepo, userRepo, chatHub)
	notificationService := notification.NewService(notificationRepo)
	subscriptionService := subscription.NewService(subscriptionRepo)
	paymentService := payment.NewService(paymentRepo, subscriptionService)

	// Initialize admin module
	adminRepo := admin.NewRepository(db)
	adminService := admin.NewService(adminRepo)
	adminJWTService := admin.NewJWTService(cfg.JWTSecret, 24*time.Hour) // 24h admin sessions

	// Initialize organization and lead modules
	orgRepo := organization.NewRepository(db)
	leadRepo := lead.NewRepository(db)
	leadService := lead.NewService(leadRepo, orgRepo, userRepo)

	// Initialize handlers
	authHandler := auth.NewHandler(authService)
	profileHandler := profile.NewHandler(profileService)
	castingHandler := casting.NewHandler(castingService)
	responseHandler := response.NewHandler(responseService)
	photoHandler := photo.NewHandler(photoService)
	chatHandler := chat.NewHandler(chatService, chatHub, redis, cfg.AllowedOrigins)
	notificationHandler := notification.NewHandler(notificationService)

	// NEW: Notification preferences handler
	prefsRepo := notification.NewPreferencesRepository(db)
	deviceRepo := notification.NewDeviceTokenRepository(db)
	preferencesHandler := notification.NewPreferencesHandler(prefsRepo, deviceRepo)

	subscriptionHandler := subscription.NewHandler(subscriptionService)
	paymentHandler := payment.NewHandler(paymentService)

	// NEW: Dashboard and Promotion handlers
	dashboardHandler := dashboard.NewHandler(dashboardRepo)
	promotionHandler := promotion.NewHandler(promotionRepo)

	// NEW: Saved castings, social links, reviews, FAQ handlers
	savedCastingsHandler := casting.NewSavedCastingsHandler(db)
	socialLinksHandler := profile.NewSocialLinksHandler(db, modelRepo)
	reviewRepo := review.NewRepository(db)
	reviewHandler := review.NewHandler(reviewRepo)
	faqHandler := content.NewFAQHandler(db)

	// Admin handlers
	adminHandler := admin.NewHandler(adminService, adminJWTService)
	moderationHandler := admin.NewModerationHandler(db, adminService)
	leadHandler := lead.NewHandler(leadService)
	userAdminHandler := admin.NewUserHandler(db, adminService)

	// Auth middleware
	authMiddleware := middleware.Auth(jwtService)

	// Create router
	r := chi.NewRouter()

	// Global middleware (no Compress for WebSocket compatibility)
	r.Use(chimw.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recover)
	r.Use(middleware.CORSHandler(cfg.AllowedOrigins))

	// WebSocket endpoint BEFORE Compress (to avoid http.Hijacker issue)
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		authMiddleware(http.HandlerFunc(chatHandler.WebSocket)).ServeHTTP(w, r)
	})

	// Enable compression for all other routes
	r.Group(func(r chi.Router) {
		r.Use(chimw.Compress(5))
	})

	// Health check endpoints
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		pkgresponse.OK(w, map[string]string{
			"status":  "ok",
			"version": "1.0.0",
		})
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Ping endpoint
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			pkgresponse.OK(w, map[string]string{
				"message": "pong",
			})
		})

		// Auth routes
		r.Mount("/auth", authHandler.Routes(authMiddleware))

		// Profile routes
		r.Mount("/profiles", profileHandler.Routes(authMiddleware))

		// Casting routes
		r.Mount("/castings", castingHandler.Routes(authMiddleware))

		// Saved castings routes
		r.Route("/castings/saved", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/", savedCastingsHandler.ListSaved)
		})
		r.Route("/castings/{id}/save", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/", savedCastingsHandler.Save)
			r.Delete("/", savedCastingsHandler.Unsave)
			r.Get("/", savedCastingsHandler.CheckSaved)
		})

		// Casting responses (nested under castings)
		r.Route("/castings/{id}/responses", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/", responseHandler.Apply)
			r.Get("/", responseHandler.ListByCasting)
		})

		// Response routes
		r.Mount("/responses", responseHandler.Routes(authMiddleware))

		// Photo routes (legacy presigned URL system)
		r.Mount("/uploads", photoHandler.UploadRoutes(authMiddleware))
		r.Mount("/photos", photoHandler.Routes(authMiddleware))

		// NEW: File upload routes (2-phase staging system)
		r.Route("/files", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/stage", uploadHandler.Stage)
			r.Post("/{id}/commit", uploadHandler.Commit)
			r.Get("/{id}", uploadHandler.GetByID)
			r.Delete("/{id}", uploadHandler.Delete)
			r.Get("/", uploadHandler.ListMy)
		})

		// Profile photos route
		r.Get("/profiles/{id}/photos", photoHandler.ListByProfile)

		// Profile social links routes
		r.Route("/profiles/{id}/social-links", func(r chi.Router) {
			r.Get("/", socialLinksHandler.List)
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware)
				r.Post("/", socialLinksHandler.Create)
				r.Delete("/{platform}", socialLinksHandler.Delete)
			})
		})

		// Profile completeness route
		r.Get("/profiles/{id}/completeness", socialLinksHandler.GetCompleteness)

		// Profile reviews routes
		r.Get("/profiles/{id}/reviews", reviewHandler.ListByProfile)
		r.Get("/profiles/{id}/reviews/summary", reviewHandler.GetSummary)

		// Chat routes
		r.Mount("/chat", chatHandler.Routes(authMiddleware))

		// Notification routes
		r.Mount("/notifications", notificationHandler.Routes(authMiddleware))

		// Notification preferences routes
		r.Mount("/notifications/preferences", preferencesHandler.Routes(authMiddleware))

		// Subscription routes
		r.Mount("/subscriptions", subscriptionHandler.Routes(authMiddleware))

		// Payment routes
		r.Mount("/payments", paymentHandler.Routes(authMiddleware))

		// NEW: Dashboard routes
		r.Mount("/dashboard", dashboard.Routes(dashboardHandler, authMiddleware))

		// NEW: Promotion routes
		r.Mount("/promotions", promotion.Routes(promotionHandler, authMiddleware))

		// NEW: Reviews routes (create/delete)
		r.Mount("/reviews", review.Routes(reviewHandler, authMiddleware))

		// NEW: FAQ routes (public)
		r.Mount("/faq", faqHandler.Routes())
	})

	// Webhooks (no auth, signature verification inside)
	r.Mount("/webhooks", paymentHandler.WebhookRoutes())

	// Public lead capture endpoint (no auth)
	r.Mount("/api/v1/leads", leadHandler.PublicRoutes())

	// Admin panel routes (separate from API)
	r.Route("/api/admin", func(r chi.Router) {
		r.Mount("/", adminHandler.Routes())
		r.Mount("/moderation", moderationHandler.Routes(adminJWTService, adminService))
		r.Mount("/leads", leadHandler.AdminRoutes(adminJWTService, adminService))
		r.Mount("/users", userAdminHandler.Routes(adminJWTService, adminService))
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info().Str("addr", server.Addr).Msg("HTTP server listening")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited properly")
}

func setupLogger(cfg *config.Config) {
	// Set time format
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Set log level
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)

	// Pretty logging for development
	if cfg.IsDevelopment() {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		})
	}
}
