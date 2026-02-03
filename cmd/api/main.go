package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/config"
	"github.com/mwork/mwork-api/internal/domain/admin"
	"github.com/mwork/mwork-api/internal/domain/auth"
	"github.com/mwork/mwork-api/internal/domain/casting"
	"github.com/mwork/mwork-api/internal/domain/chat"
	"github.com/mwork/mwork-api/internal/domain/content"
	"github.com/mwork/mwork-api/internal/domain/dashboard"
	"github.com/mwork/mwork-api/internal/domain/experience"
	"github.com/mwork/mwork-api/internal/domain/lead"
	"github.com/mwork/mwork-api/internal/domain/moderation"
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
	"github.com/mwork/mwork-api/internal/pkg/kaspi"
	pkgresponse "github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/storage"
	"github.com/mwork/mwork-api/internal/pkg/upload"
)

func main() {
	cfg := config.Load()
	setupLogger(cfg)

	log.Info().
		Str("env", cfg.Env).
		Str("port", cfg.Port).
		Msg("Starting MWork API")

	db, err := database.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to PostgreSQL")
	}
	defer database.ClosePostgres(db)

	redis, err := database.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer database.CloseRedis(redis)

	jwtService := jwt.NewService(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	// Legacy upload service used by photo module (as in your current main)
	uploadSvc := upload.NewService(&upload.Config{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		AccessKeySecret: cfg.R2AccessKeySecret,
		BucketName:      cfg.R2BucketName,
		PublicURL:       cfg.R2PublicURL,
	})

	// ---------- Repositories ----------
	userRepo := user.NewRepository(db)
	modelRepo := profile.NewModelRepository(db)
	experienceRepo := experience.NewRepository(db)
	employerRepo := profile.NewEmployerRepository(db)
	castingRepo := casting.NewRepository(db)
	responseRepo := response.NewRepository(db)
	photoRepo := photo.NewRepository(db)
	chatRepo := chat.NewRepository(db)
	moderationRepo := moderation.NewRepository(db)
	notificationRepo := notification.NewRepository(db)
	subscriptionRepo := subscription.NewRepository(db)
	paymentRepo := payment.NewRepository(db)
	dashboardRepo := dashboard.NewRepository(db)
	dashboardSvc := dashboard.NewService(db)
	promotionRepo := promotion.NewRepository(db)

	// ---------- Upload domain (2-phase) ----------
	// R2 storage client (presign/move/exists)
	r2Storage, err := storage.NewR2Storage(storage.R2Config{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		AccessKeySecret: cfg.R2AccessKeySecret,
		BucketName:      cfg.R2BucketName,
		PublicURL:       cfg.R2PublicURL,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create R2 storage")
	}

	uploadRepo := uploadDomain.NewRepository(db)

	// NOTE: This matches your current call style:
	// NewService(uploadRepo, storage, imageProcessor, baseURL)
	uploadService := uploadDomain.NewService(uploadRepo, r2Storage, nil, "/api/v1/files")

	// IMPORTANT: NewHandler MUST accept (service, baseURL, storage, repo)
	uploadHandler := uploadDomain.NewHandler(uploadService, "/api/v1/files", r2Storage, uploadRepo)

	// ---------- WebSocket hub ----------
	chatHub := chat.NewHub(redis)
	go chatHub.Run()

	// ---------- Services ----------
	authService := auth.NewService(userRepo, jwtService, redis, nil)
	profileService := profile.NewService(modelRepo, employerRepo, userRepo)
	castingService := casting.NewService(castingRepo, userRepo)
	responseService := response.NewService(responseRepo, castingRepo, modelRepo, employerRepo)
	photoService := photo.NewService(photoRepo, modelRepo, uploadSvc)
	moderationService := moderation.NewService(moderationRepo, userRepo)
	var chatService *chat.Service
	notificationService := notification.NewService(notificationRepo)
	subscriptionService := subscription.NewService(subscriptionRepo, nil, nil, nil, nil)
	paymentService := payment.NewService(paymentRepo, nil)

	// ---------- Adapters ----------
	// Adapter for auth employer profile repository
	authEmployerRepo := &authEmployerProfileAdapter{repo: employerRepo}

	// Stub adapters for subscription service
	subscriptionPhotoRepo := &subscriptionPhotoAdapter{repo: photoRepo}
	subscriptionResponseRepo := &subscriptionResponseAdapter{repo: responseRepo}
	subscriptionCastingRepo := &subscriptionCastingAdapter{repo: castingRepo}
	subscriptionProfileRepo := &subscriptionProfileAdapter{repo: modelRepo}

	// Adapter for casting profile service
	castingProfileService := &castingProfileServiceAdapter{service: profileService}

	// Adapter for subscription payment service
	subscriptionPaymentService := &subscriptionPaymentAdapter{service: paymentService}

	// Adapter for subscription kaspi client
	subscriptionKaspiClient := &subscriptionKaspiAdapter{client: kaspi.NewClient(kaspi.Config{
		BaseURL:    cfg.KaspiBaseURL,
		MerchantID: cfg.KaspiMerchantID,
		SecretKey:  cfg.KaspiSecretKey,
	})}

	// Update authService with authEmployerRepo
	authService = auth.NewService(userRepo, jwtService, redis, authEmployerRepo)

	// Update services with proper dependencies
	subscriptionService = subscription.NewService(subscriptionRepo, subscriptionPhotoRepo, subscriptionResponseRepo, subscriptionCastingRepo, subscriptionProfileRepo)
	paymentService = payment.NewService(paymentRepo, subscriptionService)
	limitChecker := subscription.NewLimitChecker(subscriptionService)
	chatService = chat.NewService(chatRepo, userRepo, chatHub, moderationService, limitChecker)

	adminRepo := admin.NewRepository(db)
	adminService := admin.NewService(adminRepo)
	adminJWTService := admin.NewJWTService(cfg.JWTSecret, 24*time.Hour)

	orgRepo := organization.NewRepository(db)
	leadRepo := lead.NewRepository(db)
	leadService := lead.NewService(leadRepo, orgRepo, userRepo)

	// ---------- Handlers ----------
	authHandler := auth.NewHandler(authService)
	profileHandler := profile.NewHandler(profileService)
	castingHandler := casting.NewHandler(castingService, castingProfileService)
	experienceHandler := experience.NewHandler(experienceRepo, modelRepo)
	responseHandler := response.NewHandler(responseService, limitChecker)
	photoHandler := photo.NewHandler(photoService)
	chatHandler := chat.NewHandler(chatService, chatHub, redis, cfg.AllowedOrigins)
	moderationHandler := moderation.NewHandler(moderationService)
	notificationHandler := notification.NewHandler(notificationService)

	prefsRepo := notification.NewPreferencesRepository(db)
	deviceRepo := notification.NewDeviceTokenRepository(db)
	preferencesHandler := notification.NewPreferencesHandler(prefsRepo, deviceRepo)

	subscriptionHandler := subscription.NewHandler(subscriptionService, subscriptionPaymentService, subscriptionKaspiClient, &subscription.Config{
		FrontendURL: "http://localhost:3000",
		BackendURL:  "http://localhost:8080",
	})
	paymentHandler := payment.NewHandler(paymentService, cfg.KaspiSecretKey)

	dashboardHandler := dashboard.NewHandler(dashboardRepo, dashboardSvc)
	promotionHandler := promotion.NewHandler(promotionRepo)

	savedCastingsHandler := casting.NewSavedCastingsHandler(db)
	socialLinksHandler := profile.NewSocialLinksHandler(db, modelRepo)
	reviewRepo := review.NewRepository(db)
	reviewHandler := review.NewHandler(reviewRepo)
	faqHandler := content.NewFAQHandler(db)

	adminHandler := admin.NewHandler(adminService, adminJWTService)
	adminModerationHandler := admin.NewModerationHandler(db, adminService)
	leadHandler := lead.NewHandler(leadService)
	userAdminHandler := admin.NewUserHandler(db, adminService)

	authMiddleware := middleware.Auth(jwtService)
	responseLimitMiddleware := middleware.RequireResponseLimit(limitChecker, &responseLimitCounter{repo: responseRepo})
	chatLimitMiddleware := middleware.RequireChatLimit(limitChecker)
	photoLimitMiddleware := middleware.RequirePhotoLimit(limitChecker, &photoLimitCounter{repo: photoRepo}, &modelProfileIDProvider{repo: modelRepo})

	// ---------- Router ----------
	r := chi.NewRouter()

	r.Use(chimw.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recover)
	r.Use(middleware.CORSHandler(cfg.AllowedOrigins))

	// WebSocket endpoint (before Compress)
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		authMiddleware(http.HandlerFunc(chatHandler.WebSocket)).ServeHTTP(w, r)
	})

	// Compress for everything else
	r.Group(func(r chi.Router) {
		r.Use(chimw.Compress(5))
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		pkgresponse.OK(w, map[string]string{
			"status":  "ok",
			"version": "1.0.0",
		})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			pkgresponse.OK(w, map[string]string{"message": "pong"})
		})

		r.Mount("/auth", authHandler.Routes(authMiddleware))
		r.Mount("/profiles", profileHandler.Routes(authMiddleware))
		r.Mount("/castings", castingHandler.Routes(authMiddleware))

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

		r.Route("/castings/{id}/responses", func(r chi.Router) {
			r.Use(authMiddleware)
			r.With(responseLimitMiddleware).Post("/", responseHandler.Apply)
			r.Get("/", responseHandler.ListByCasting)
		})
		r.Mount("/responses", responseHandler.Routes(authMiddleware))

		// legacy uploads/photos
		r.Mount("/uploads", photoHandler.UploadRoutes(authMiddleware))
		r.Route("/photos", func(r chi.Router) {
			r.Use(authMiddleware)
			r.With(photoLimitMiddleware).Post("/", photoHandler.ConfirmUpload)
			r.Delete("/{id}", photoHandler.Delete)
			r.Patch("/{id}/avatar", photoHandler.SetAvatar)
			r.Patch("/reorder", photoHandler.Reorder)
		})

		// NEW: 2-phase file uploads
		r.Route("/files", func(r chi.Router) {
			r.Use(authMiddleware)

			// New 2-phase endpoints
			r.Post("/init", uploadHandler.Init)
			r.Post("/confirm", uploadHandler.Confirm)

			// Existing staging system endpoints
			r.Post("/stage", uploadHandler.Stage)
			r.Post("/{id}/commit", uploadHandler.Commit)
			r.Get("/{id}", uploadHandler.GetByID)
			r.Delete("/{id}", uploadHandler.Delete)
			r.Get("/", uploadHandler.ListMy)
		})

		r.Get("/profiles/{id}/photos", photoHandler.ListByProfile)

		r.Mount("/", experienceHandler.Routes(authMiddleware))

		r.Route("/profiles/{id}/social-links", func(r chi.Router) {
			r.Get("/", socialLinksHandler.List)
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware)
				r.Post("/", socialLinksHandler.Create)
				r.Delete("/{platform}", socialLinksHandler.Delete)
			})
		})

		r.Get("/profiles/{id}/completeness", socialLinksHandler.GetCompleteness)

		r.Get("/profiles/{id}/reviews", reviewHandler.ListByProfile)
		r.Get("/profiles/{id}/reviews/summary", reviewHandler.GetSummary)

		r.Route("/chat", func(r chi.Router) {
			r.Use(authMiddleware)
			r.With(chatLimitMiddleware).Post("/rooms", chatHandler.CreateRoom)
			r.Get("/rooms", chatHandler.ListRooms)

			r.Get("/rooms/{id}/messages", chatHandler.GetMessages)
			r.With(chatLimitMiddleware).Post("/rooms/{id}/messages", chatHandler.SendMessage)
			r.Post("/rooms/{id}/read", chatHandler.MarkAsRead)

			r.Get("/unread", chatHandler.GetUnreadCount)
		})
		r.Mount("/moderation", moderationHandler.Routes(authMiddleware))
		r.Mount("/notifications", notificationHandler.Routes(authMiddleware))
		r.Mount("/notifications/preferences", preferencesHandler.Routes(authMiddleware))

		r.Mount("/subscriptions", subscriptionHandler.Routes(authMiddleware))
		r.Mount("/payments", paymentHandler.Routes(authMiddleware))

		r.Mount("/dashboard", dashboard.Routes(dashboardHandler, authMiddleware))
		r.Mount("/promotions", promotion.Routes(promotionHandler, authMiddleware))
		r.Mount("/reviews", review.Routes(reviewHandler, authMiddleware))
		r.Mount("/faq", faqHandler.Routes())
	})

	r.Mount("/webhooks", paymentHandler.WebhookRoutes())
	r.Mount("/api/v1/leads", leadHandler.PublicRoutes())

	r.Route("/api/admin", func(r chi.Router) {
		r.Mount("/", adminHandler.Routes())
		r.Mount("/moderation", adminModerationHandler.Routes(adminJWTService, adminService))

		// User-facing moderation admin routes (reports)
		adminAuthMiddleware := admin.AuthMiddleware(adminJWTService, adminService)
		adminOnlyMiddleware := admin.RequirePermission(admin.PermModerateContent) // Using content moderation permission
		r.Mount("/reports", moderationHandler.AdminRoutes(adminAuthMiddleware, adminOnlyMiddleware))

		r.Mount("/leads", leadHandler.AdminRoutes(adminJWTService, adminService))
		r.Mount("/users", userAdminHandler.Routes(adminJWTService, adminService))
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("addr", server.Addr).Msg("HTTP server listening")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

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
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.IsDevelopment() {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		})
	}
}

// Adapter implementations to bridge interface mismatches

// authEmployerProfileAdapter adapts profile.EmployerRepository to auth.EmployerProfileRepository
type authEmployerProfileAdapter struct {
	repo profile.EmployerRepository
}

func (a *authEmployerProfileAdapter) Create(ctx context.Context, authProfile *auth.EmployerProfile) error {
	// Convert auth.EmployerProfile to profile.EmployerProfile
	employerProfile := &profile.EmployerProfile{
		ID:            authProfile.ID,
		UserID:        authProfile.UserID,
		CompanyName:   authProfile.CompanyName,
		Description:   sql.NullString{String: authProfile.Description, Valid: authProfile.Description != ""},
		Website:       sql.NullString{String: authProfile.Website, Valid: authProfile.Website != ""},
		ContactPerson: sql.NullString{String: authProfile.ContactPerson, Valid: authProfile.ContactPerson != ""},
		CreatedAt:     authProfile.CreatedAt,
		UpdatedAt:     authProfile.UpdatedAt,
	}
	return a.repo.Create(ctx, employerProfile)
}

// Additional adapters for interface mismatches

type castingProfileServiceAdapter struct {
	service *profile.Service
}

func (a *castingProfileServiceAdapter) GetEmployerProfileByUserID(ctx context.Context, userID uuid.UUID) (*casting.EmployerProfile, error) {
	profile, err := a.service.GetEmployerProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &casting.EmployerProfile{
		ID:     profile.ID,
		UserID: profile.UserID,
	}, nil
}

type subscriptionPaymentAdapter struct {
	service *payment.Service
}

func (a *subscriptionPaymentAdapter) CreatePayment(ctx context.Context, userID, subscriptionID uuid.UUID, amount float64, provider string) (*subscription.Payment, error) {
	payment, err := a.service.CreatePayment(ctx, userID, subscriptionID, amount, payment.Provider(provider))
	if err != nil {
		return nil, err
	}

	return &subscription.Payment{
		ID:             payment.ID,
		UserID:         payment.UserID,
		SubscriptionID: payment.SubscriptionID,
		Amount:         payment.Amount,
		KaspiOrderID:   payment.KaspiOrderID,
		Status:         string(payment.Status),
		CreatedAt:      payment.CreatedAt,
	}, nil
}

type subscriptionKaspiAdapter struct {
	client *kaspi.Client
}

func (a *subscriptionKaspiAdapter) CreatePayment(ctx context.Context, req subscription.KaspiPaymentRequest) (*subscription.KaspiPaymentResponse, error) {
	kaspiReq := kaspi.CreatePaymentRequest{
		Amount:      req.Amount,
		OrderID:     req.OrderID,
		Description: req.Description,
		ReturnURL:   req.ReturnURL,
		CallbackURL: req.CallbackURL,
	}

	resp, err := a.client.CreatePayment(ctx, kaspiReq)
	if err != nil {
		return nil, err
	}

	return &subscription.KaspiPaymentResponse{
		PaymentID:  resp.PaymentID,
		PaymentURL: resp.PaymentURL,
		Status:     resp.Status,
	}, nil
}

// Subscription adapters

type subscriptionPhotoAdapter struct {
	repo photo.Repository
}

func (a *subscriptionPhotoAdapter) CountByProfileID(ctx context.Context, profileID uuid.UUID) (int, error) {
	return a.repo.CountByProfile(ctx, profileID)
}

type subscriptionResponseAdapter struct {
	repo response.Repository
}

func (a *subscriptionResponseAdapter) CountMonthlyByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	return a.repo.CountMonthlyByUserID(ctx, userID)
}

type subscriptionCastingAdapter struct {
	repo casting.Repository
}

func (a *subscriptionCastingAdapter) CountActiveByCreatorID(ctx context.Context, creatorID uuid.UUID) (int, error) {
	return a.repo.CountActiveByCreatorID(ctx, creatorID.String())
}

type subscriptionProfileAdapter struct {
	repo profile.ModelRepository
}

func (a *subscriptionProfileAdapter) GetByUserID(ctx context.Context, userID uuid.UUID) (*subscription.Profile, error) {
	modelProfile, err := a.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &subscription.Profile{
		ID:     modelProfile.ID,
		UserID: modelProfile.UserID,
	}, nil
}

type responseLimitCounter struct {
	repo response.Repository
}

func (a *responseLimitCounter) CountMonthlyByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	return a.repo.CountMonthlyByUserID(ctx, userID)
}

type photoLimitCounter struct {
	repo photo.Repository
}

func (a *photoLimitCounter) CountByProfileID(ctx context.Context, profileID uuid.UUID) (int, error) {
	return a.repo.CountByProfile(ctx, profileID)
}

type modelProfileIDProvider struct {
	repo profile.ModelRepository
}

func (a *modelProfileIDProvider) ProfileIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	modelProfile, err := a.repo.GetByUserID(ctx, userID)
	if err != nil || modelProfile == nil {
		return uuid.Nil, err
	}
	return modelProfile.ID, nil
}
