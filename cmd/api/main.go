package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/config"
	"github.com/mwork/mwork-api/internal/domain/admin"
	"github.com/mwork/mwork-api/internal/domain/auth"
	"github.com/mwork/mwork-api/internal/domain/casting"
	"github.com/mwork/mwork-api/internal/domain/chat"
	"github.com/mwork/mwork-api/internal/domain/content"
	"github.com/mwork/mwork-api/internal/domain/credit"
	"github.com/mwork/mwork-api/internal/domain/dashboard"
	"github.com/mwork/mwork-api/internal/domain/experience"
	"github.com/mwork/mwork-api/internal/domain/favorite"
	"github.com/mwork/mwork-api/internal/domain/lead"
	"github.com/mwork/mwork-api/internal/domain/moderation"
	"github.com/mwork/mwork-api/internal/domain/notification"
	"github.com/mwork/mwork-api/internal/domain/organization"
	"github.com/mwork/mwork-api/internal/domain/payment"
	"github.com/mwork/mwork-api/internal/domain/photo"
	"github.com/mwork/mwork-api/internal/domain/photostudio_booking"
	"github.com/mwork/mwork-api/internal/domain/profile"
	"github.com/mwork/mwork-api/internal/domain/promotion"
	"github.com/mwork/mwork-api/internal/domain/relationships"
	"github.com/mwork/mwork-api/internal/domain/response"
	"github.com/mwork/mwork-api/internal/domain/review"
	"github.com/mwork/mwork-api/internal/domain/subscription"
	uploadDomain "github.com/mwork/mwork-api/internal/domain/upload"
	"github.com/mwork/mwork-api/internal/domain/user"
	"github.com/mwork/mwork-api/internal/domain/wallet"
	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/database"
	emailpkg "github.com/mwork/mwork-api/internal/pkg/email"
	"github.com/mwork/mwork-api/internal/pkg/featurepayment"
	"github.com/mwork/mwork-api/internal/pkg/jwt"
	"github.com/mwork/mwork-api/internal/pkg/logger"
	"github.com/mwork/mwork-api/internal/pkg/photostudio"
	pkgresponse "github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/robokassa"
	"github.com/mwork/mwork-api/internal/pkg/storage"
	"github.com/mwork/mwork-api/internal/pkg/upload"

	_ "github.com/mwork/mwork-api/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           MWork API
// @version         1.0
// @description     API server for model agency aggregator.
// @termsOfService  http://swagger.io/terms/

// @contact.name    API Support
// @contact.email   support@swagger.io

// @BasePath        /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
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
		log.Warn().Err(err).Msg("Failed to connect to Redis - running without Redis...")
		redis = nil
	}
	defer database.CloseRedis(redis)

	jwtService := jwt.NewService(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	emailService := emailpkg.NewService(emailpkg.SendGridConfig{
		APIKey:    cfg.SendGridAPIKey,
		FromEmail: cfg.SendGridFromEmail,
		FromName:  cfg.SendGridFromName,
	})
	defer emailService.Close()

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
	adminProfRepo := profile.NewAdminRepository(db)
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
	favoriteRepo := favorite.NewRepository(db)
	walletRepo := wallet.NewRepository(db)

	// ---------- Upload domain (2-phase) ----------
	const filesBaseURL = "/api/v1/media"

	var (
		uploadStorage      uploadDomain.UploadStorage
		uploadFileStorage  storage.Storage
		uploadCloudStorage storage.Storage
		servingLocalFiles  bool
		proxyUploadMode    bool
	)

	if cfg.R2AccountID != "" && cfg.R2AccessKeyID != "" && cfg.R2AccessKeySecret != "" && cfg.R2BucketName != "" {
		r2Storage, r2Err := storage.NewR2Storage(storage.R2Config{
			AccountID:       cfg.R2AccountID,
			AccessKeyID:     cfg.R2AccessKeyID,
			AccessKeySecret: cfg.R2AccessKeySecret,
			BucketName:      cfg.R2BucketName,
			PublicURL:       cfg.R2PublicURL,
		})
		if r2Err != nil {
			log.Fatal().Err(r2Err).Msg("Failed to create R2 storage")
		}
		uploadStorage = r2Storage
		uploadFileStorage = r2Storage
		uploadCloudStorage = r2Storage
		log.Info().Msg("Upload storage: R2")
	} else {
		localStorage, localErr := storage.NewLocalStorage(cfg.UploadLocalPath, filesBaseURL)
		if localErr != nil {
			log.Fatal().Err(localErr).Msg("Failed to create local upload storage")
		}
		uploadStorage = storage.NewUploadStorageAdapter(localStorage)
		uploadFileStorage = localStorage
		proxyUploadMode = true
		servingLocalFiles = true
		log.Warn().Str("path", cfg.UploadLocalPath).Msg("R2 is not configured, using local upload storage")
	}

	uploadRepo := uploadDomain.NewRepository(db)
	uploadService := uploadDomain.NewServiceWithConfig(uploadRepo, uploadFileStorage, uploadCloudStorage, filesBaseURL, uploadDomain.Config{
		MaxUploadSize: int64(cfg.UploadMaxMB) * 1024 * 1024,
		StagingTTL:    time.Duration(cfg.UploadStagingHours) * time.Hour,
		PresignExpiry: time.Duration(cfg.UploadPresignMin) * time.Minute,
	})
	uploadHandler := uploadDomain.NewHandler(uploadService, filesBaseURL, uploadStorage, uploadRepo, proxyUploadMode)

	// ---------- WebSocket hub ----------
	chatHub := chat.NewHub(redis)
	go chatHub.Run()

	photoStudioTimeout := time.Duration(cfg.PhotoStudioTimeoutSeconds) * time.Second
	photoStudioSyncEnabled := cfg.PhotoStudioSyncEnabled && cfg.PhotoStudioBaseURL != ""
	var photoStudioClient auth.PhotoStudioClient
	var photoStudioConcreteClient *photostudio.Client
	if photoStudioSyncEnabled {
		photoStudioConcreteClient = photostudio.NewClient(
			cfg.PhotoStudioBaseURL,
			cfg.PhotoStudioToken,
			photoStudioTimeout,
			"MWork/1.0.0 photostudio-sync",
		)
		photoStudioClient = photoStudioConcreteClient
	}

	// ---------- Services ----------
	profileService := profile.NewService(modelRepo, employerRepo, adminProfRepo, userRepo)
	castingService := casting.NewService(castingRepo, userRepo)
	responseService := response.NewService(responseRepo, castingRepo, modelRepo, employerRepo)
	photoService := photo.NewService(photoRepo, modelRepo, uploadSvc)
	moderationService := moderation.NewService(moderationRepo, userRepo)
	relationshipsRepo := relationships.NewRepository(db)
	relationshipsService := relationships.NewService(relationshipsRepo)
	var chatService *chat.Service
	notificationService := notification.NewService(notificationRepo)
	subscriptionService := subscription.NewService(subscriptionRepo, nil, nil, nil, nil)
	paymentService := payment.NewService(paymentRepo, nil)
	walletService := wallet.NewService(walletRepo)

	// ---------- Adapters ----------
	// AccessChecker: composite adapter for chat access (checks both bans and blocks)
	accessChecker := &chatAccessChecker{
		moderationService:    moderationService,
		relationshipsService: relationshipsService,
	}

	// UploadResolver: adapter for resolving upload details
	uploadResolver := &chatUploadResolver{
		uploadService: uploadService,
		baseURL:       filesBaseURL,
	}

	// Adapter for auth model profile repository
	authModelRepo := &authModelProfileAdapter{repo: modelRepo}

	// Adapter for auth employer profile repository
	authEmployerRepo := &authEmployerProfileAdapter{repo: employerRepo}

	// Stub adapters for subscription service
	subscriptionPhotoRepo := &subscriptionPhotoAdapter{repo: photoRepo}
	subscriptionResponseRepo := &subscriptionResponseAdapter{repo: responseRepo}
	subscriptionCastingRepo := &subscriptionCastingAdapter{repo: castingRepo}
	subscriptionProfileRepo := &subscriptionProfileAdapter{repo: modelRepo}

	// Adapter for casting profile service
	castingProfileService := &castingProfileServiceAdapter{service: profileService}

	verificationCodeRepo := auth.NewVerificationCodeRepository(db)
	refreshTokenRepo := auth.NewRefreshTokenRepository(db)

	// Update authService with authEmployerRepo
	authService := auth.NewService(
		userRepo,
		authModelRepo,
		jwtService,
		refreshTokenRepo,
		authEmployerRepo,
		photoStudioClient,
		photoStudioSyncEnabled,
		photoStudioTimeout,
		verificationCodeRepo,
		cfg.VerificationCodePepper,
		cfg.IsDevelopment(),
		cfg.AllowLegacyRefresh,
		emailService,
	)

	// Update services with proper dependencies
	subscriptionService = subscription.NewService(subscriptionRepo, subscriptionPhotoRepo, subscriptionResponseRepo, subscriptionCastingRepo, subscriptionProfileRepo)
	paymentService = payment.NewService(paymentRepo, subscriptionService)
	hashAlgo, err := robokassa.NormalizeHashAlgorithm(cfg.RobokassaHashAlgo)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid ROBOKASSA_HASH_ALGO")
	}
	paymentService.SetRobokassaConfig(payment.RobokassaConfig{
		MerchantLogin: cfg.RobokassaMerchantLogin,
		Password1:     cfg.RobokassaPassword1,
		Password2:     cfg.RobokassaPassword2,
		TestPassword1: cfg.RobokassaTestPassword1,
		TestPassword2: cfg.RobokassaTestPassword2,
		IsTest:        cfg.RobokassaIsTest,
		HashAlgo:      hashAlgo,
		PaymentURL:    cfg.RobokassaPaymentURL,
		ResultURL:     cfg.RobokassaResultURL,
		SuccessURL:    cfg.RobokassaSuccessURL,
		FailURL:       cfg.RobokassaFailURL,
	})

	// Adapter for subscription payment service (must use configured paymentService instance)
	subscriptionPaymentService := &subscriptionPaymentAdapter{service: paymentService}
	limitChecker := subscription.NewLimitChecker(subscriptionService)
	chatService = chat.NewService(chatRepo, userRepo, chatHub, accessChecker, limitChecker, uploadResolver)

	// Adapter for chat service to response service
	chatServiceAdapter := &chatServiceAdapter{service: chatService}

	adminRepo := admin.NewRepository(db)
	adminService := admin.NewService(adminRepo)
	adminJWTService := admin.NewJWTService(cfg.JWTSecret, 24*time.Hour)

	// Credit service initialization
	creditService := credit.NewService(db)

	featurePayProvider, err := featurepayment.NewPaymentProvider(cfg.PaymentMode, walletService, creditService)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize payment provider")
	}

	// Inject payment abstractions into response service
	responseService.SetCreditService(creditService)
	responseService.SetPaymentProvider(featurePayProvider)

	// TASK 1: Inject chat service into response service via adapter
	// This enables auto-creation of chat rooms when responses are accepted
	responseService.SetChatService(chatServiceAdapter)

	// B4: Inject credit service into payment service for credit purchases
	paymentService.SetCreditService(creditService)

	orgRepo := organization.NewRepository(db)
	leadRepo := lead.NewRepository(db)
	leadService := lead.NewService(leadRepo, orgRepo, userRepo, &leadEmployerProfileAdapter{repo: employerRepo})

	// ---------- Handlers ----------
	authHandler := auth.NewHandler(authService)
	profileHandler := profile.NewHandler(profileService)
	castingHandler := casting.NewHandler(castingService, castingProfileService)
	experienceHandler := experience.NewHandler(experienceRepo, modelRepo)
	responseHandler := response.NewHandler(responseService, limitChecker)
	photoHandler := photo.NewHandler(photoService)
	chatProfileFetcher := &chatProfileFetcher{
		userRepo:       userRepo,
		profileService: profileService,
	}
	chatHandler := chat.NewHandler(chatService, chatHub, redis, cfg.AllowedOrigins, chatProfileFetcher)
	moderationHandler := moderation.NewHandler(moderationService)
	relationshipProfileFetcher := &relationshipProfileFetcher{
		userRepo:       userRepo,
		profileService: profileService,
	}
	relationshipHandler := relationships.NewHandler(relationshipsService, relationshipProfileFetcher)
	notificationHandler := notification.NewHandler(notificationService)

	prefsRepo := notification.NewPreferencesRepository(db)
	deviceRepo := notification.NewDeviceTokenRepository(db)
	preferencesHandler := notification.NewPreferencesHandler(prefsRepo, deviceRepo)

	subscriptionHandler := subscription.NewHandler(subscriptionService, subscriptionPaymentService, &subscription.Config{
		FrontendURL: "http://localhost:3000",
		BackendURL:  "http://localhost:8080",
	})
	paymentHandler := payment.NewHandler(paymentService)

	dashboardHandler := dashboard.NewHandler(dashboardRepo, dashboardSvc)
	promotionHandler := promotion.NewHandler(promotionRepo)
	favoriteHandler := favorite.NewHandler(favoriteRepo)
	walletHandler := wallet.NewHandler(walletService)

	savedCastingsHandler := casting.NewSavedCastingsHandler(db)
	socialLinksHandler := profile.NewSocialLinksHandler(db, modelRepo)
	reviewRepo := review.NewRepository(db)
	reviewHandler := review.NewHandler(reviewRepo, modelRepo)
	faqHandler := content.NewFAQHandler(db)

	creditHandler := admin.NewCreditHandler(creditService, adminService)
	photoStudioAdminHandler := admin.NewPhotoStudioHandler(db, photoStudioClient, photoStudioSyncEnabled, photoStudioTimeout)
	adminHandler := admin.NewHandler(adminService, adminJWTService, photoStudioAdminHandler, creditHandler)
	adminModerationHandler := admin.NewModerationHandler(db, adminService)
	leadHandler := lead.NewHandler(leadService)
	userAdminHandler := admin.NewUserHandler(db, adminService, creditHandler)

	// PhotoStudio booking integration
	photoStudioBookingService := photostudio_booking.NewService(photoStudioConcreteClient, photoStudioSyncEnabled)
	photoStudioBookingHandler := photostudio_booking.NewHandler(photoStudioBookingService)

	authMiddleware := middleware.Auth(jwtService)
	emailVerificationWhitelist := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/refresh",
		"/api/v1/auth/logout",
		"/api/v1/auth/verify/request",
		"/api/v1/auth/verify/confirm",
		"/api/v1/auth/verify/request/me",
		"/api/v1/auth/verify/confirm/me",
		"/health",
		"/healthz",
		"/swagger",
		"/docs",
	}
	emailVerifiedMiddleware := middleware.RequireVerifiedEmail(userRepo, emailVerificationWhitelist)
	authWithVerifiedEmailMiddleware := func(next http.Handler) http.Handler {
		return authMiddleware(emailVerifiedMiddleware(next))
	}
	responseLimitMiddleware := middleware.RequireResponseLimit(limitChecker, &responseLimitCounter{repo: responseRepo})
	chatLimitMiddleware := middleware.RequireChatLimit(limitChecker)
	photoLimitMiddleware := middleware.RequirePhotoLimit(limitChecker, &photoLimitCounter{repo: photoRepo}, &modelProfileIDProvider{repo: modelRepo})

	// ---------- Router ----------
	r := chi.NewRouter()

	r.Use(chimw.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.CORSHandler(cfg.AllowedOrigins))
	r.Use(chimw.Compress(5))

	// Swagger будет доступен по адресу: http://localhost:PORT/swagger/index.html
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("doc.json"), // URL указывающий на doc.json
	))

	// ======================
	// WebSocket endpoint
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		authWithVerifiedEmailMiddleware(http.HandlerFunc(chatHandler.WebSocket)).ServeHTTP(w, r)
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		pkgresponse.OK(w, map[string]string{
			"status":  "ok",
			"version": "1.0.0",
		})
	})

	if servingLocalFiles {
		r.Handle(filesBaseURL+"/*", http.StripPrefix(filesBaseURL+"/", http.FileServer(http.Dir(cfg.UploadLocalPath))))
	}

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			pkgresponse.OK(w, map[string]string{"message": "pong"})
		})

		r.Mount("/auth", authHandler.Routes(authWithVerifiedEmailMiddleware))
		r.Mount("/profiles", profileHandler.Routes(authWithVerifiedEmailMiddleware))
		mountProfileExperienceRoutes(
			r,
			authWithVerifiedEmailMiddleware,
			experienceHandler.List,
			experienceHandler.Create,
			experienceHandler.Delete,
		)
		r.Mount("/castings", castingHandler.Routes(authWithVerifiedEmailMiddleware))

		r.Route("/castings/saved", func(r chi.Router) {
			r.Use(authWithVerifiedEmailMiddleware)
			r.Get("/", savedCastingsHandler.ListSaved)
		})
		r.Route("/castings/{id}/save", func(r chi.Router) {
			r.Use(authWithVerifiedEmailMiddleware)
			r.Post("/", savedCastingsHandler.Save)
			r.Delete("/", savedCastingsHandler.Unsave)
			r.Get("/", savedCastingsHandler.CheckSaved)
		})

		r.Route("/castings/{id}/responses", func(r chi.Router) {
			r.Use(authWithVerifiedEmailMiddleware)
			r.With(responseLimitMiddleware).Post("/", responseHandler.Apply)
			r.Get("/", responseHandler.ListByCasting)
		})
		r.Mount("/responses", responseHandler.Routes(authWithVerifiedEmailMiddleware))

		// legacy uploads/photos
		r.Mount("/uploads", photoHandler.UploadRoutes(authWithVerifiedEmailMiddleware))
		r.Route("/photos", func(r chi.Router) {
			r.Use(authWithVerifiedEmailMiddleware)
			r.With(photoLimitMiddleware).Post("/", photoHandler.ConfirmUpload)
			r.Delete("/{id}", photoHandler.Delete)
			r.Patch("/{id}/avatar", photoHandler.SetAvatar)
			r.Patch("/reorder", photoHandler.Reorder)
		})

		// NEW: 2-phase file uploads
		r.Route("/files", func(r chi.Router) {
			// Public read for committed uploads; handler enforces access rules for non-committed files.
			r.Get("/{id}", uploadHandler.GetByID)

			r.Group(func(r chi.Router) {
				r.Use(authWithVerifiedEmailMiddleware)

				// New 2-phase endpoints
				r.Post("/init", uploadHandler.Init)
				r.Post("/confirm", uploadHandler.Confirm)

				// Existing staging system endpoints
				r.Post("/stage", uploadHandler.Stage)
				r.Post("/{id}/commit", uploadHandler.Commit)
				r.Delete("/{id}", uploadHandler.Delete)
				r.Get("/", uploadHandler.ListMy)
			})
		})

		r.Get("/profiles/{id}/photos", photoHandler.ListByProfile)

		r.Route("/profiles/{id}/social-links", func(r chi.Router) {
			r.Get("/", socialLinksHandler.List)
			r.Group(func(r chi.Router) {
				r.Use(authWithVerifiedEmailMiddleware)
				r.Post("/", socialLinksHandler.Create)
				r.Delete("/{platform}", socialLinksHandler.Delete)
			})
		})

		r.Get("/profiles/{id}/completeness", socialLinksHandler.GetCompleteness)

		r.Get("/profiles/{id}/reviews", reviewHandler.ListByProfile)
		r.Get("/profiles/{id}/reviews/summary", reviewHandler.GetSummary)

		r.Route("/chat", func(r chi.Router) {
			r.Use(authWithVerifiedEmailMiddleware)
			r.Post("/rooms", chatHandler.CreateRoom)
			r.Get("/rooms", chatHandler.ListRooms)

			r.Get("/rooms/{id}/messages", chatHandler.GetMessages)
			r.With(chatLimitMiddleware).Post("/rooms/{id}/messages", chatHandler.SendMessage)
			r.Post("/rooms/{id}/read", chatHandler.MarkAsRead)

			// Member management
			r.Get("/rooms/{id}/members", chatHandler.GetMembers)
			r.Post("/rooms/{id}/members", chatHandler.AddMember)
			r.Delete("/rooms/{id}/members/{userId}", chatHandler.RemoveMember)
			r.Post("/rooms/{id}/leave", chatHandler.LeaveRoom)

			r.Get("/unread", chatHandler.GetUnreadCount)
		})
		r.Mount("/relationships", relationshipHandler.Routes(authWithVerifiedEmailMiddleware))
		r.Mount("/moderation", moderationHandler.Routes(authWithVerifiedEmailMiddleware))
		r.Mount("/notifications", notificationHandler.Routes(authWithVerifiedEmailMiddleware))
		r.Mount("/notifications/preferences", preferencesHandler.Routes(authWithVerifiedEmailMiddleware))

		r.Mount("/subscriptions", subscriptionHandler.Routes(authWithVerifiedEmailMiddleware))
		r.Mount("/payments", paymentHandler.Routes(authWithVerifiedEmailMiddleware))

		r.Mount("/dashboard", dashboard.Routes(dashboardHandler, authWithVerifiedEmailMiddleware))
		r.Mount("/promotions", promotion.Routes(promotionHandler, authWithVerifiedEmailMiddleware))
		r.Mount("/favorites", favorite.Routes(favoriteHandler, authWithVerifiedEmailMiddleware))
		r.Mount("/demo/wallet", walletHandler.Routes(authWithVerifiedEmailMiddleware))
		r.Mount("/reviews", review.Routes(reviewHandler, authWithVerifiedEmailMiddleware))
		r.Mount("/faq", faqHandler.Routes())

		// PhotoStudio booking integration
		r.Mount("/photostudio", photoStudioBookingHandler.Routes(authWithVerifiedEmailMiddleware))
	})

	r.Mount("/webhooks", paymentHandler.WebhookRoutes())
	r.Mount("/api/v1/leads", leadHandler.PublicRoutes())

	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Mount("/", adminHandler.Routes())
		r.Mount("/moderation", adminModerationHandler.Routes(adminJWTService, adminService))

		// User-facing moderation admin routes (reports)
		adminAuthMiddleware := admin.AuthMiddleware(adminJWTService, adminService)
		adminOnlyMiddleware := admin.RequirePermission(admin.PermModerateContent) // Using content moderation permission
		r.Mount("/reports", moderationHandler.AdminRoutes(adminAuthMiddleware, adminOnlyMiddleware))

		r.Mount("/leads", leadHandler.AdminRoutes(adminJWTService, adminService))
		r.Mount("/users", userAdminHandler.Routes(adminJWTService, adminService))
	})
	rootHandler := middleware.Logger(middleware.Recover(r))
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      rootHandler,
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

func mountProfileExperienceRoutes(
	r chi.Router,
	authMiddleware func(http.Handler) http.Handler,
	listHandler http.HandlerFunc,
	createHandler http.HandlerFunc,
	deleteHandler http.HandlerFunc,
) {
	r.Get("/profiles/{id}/experience", listHandler)
	r.With(authMiddleware).Post("/profiles/{id}/experience", createHandler)
	r.With(authMiddleware).Delete("/profiles/{id}/experience/{expId}", deleteHandler)
}

// authModelProfileAdapter adapts profile.ModelRepository to auth.ModelProfileRepository
type authModelProfileAdapter struct {
	repo profile.ModelRepository
}

func (a *authModelProfileAdapter) Create(ctx context.Context, authProfile *auth.ModelProfile) error {
	modelProfile := &profile.ModelProfile{
		ID:           authProfile.ID,
		UserID:       authProfile.UserID,
		IsPublic:     true,
		ProfileViews: 0,
		Rating:       0,
		TotalReviews: 0,
		CreatedAt:    authProfile.CreatedAt,
		UpdatedAt:    authProfile.UpdatedAt,
	}
	modelProfile.SetLanguages(nil)
	modelProfile.SetCategories(nil)
	modelProfile.SetSkills(nil)
	modelProfile.SetTravelCities(nil)

	return a.repo.Create(ctx, modelProfile)
}

func (a *authModelProfileAdapter) GetByUserID(ctx context.Context, userID uuid.UUID) (*auth.ModelProfile, error) {
	modelProfile, err := a.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if modelProfile == nil {
		return nil, nil
	}

	return &auth.ModelProfile{
		ID:        modelProfile.ID,
		UserID:    modelProfile.UserID,
		CreatedAt: modelProfile.CreatedAt,
		UpdatedAt: modelProfile.UpdatedAt,
	}, nil
}

func setupLogger(cfg *config.Config) {
	loggerCfg := logger.Config{
		Level:       cfg.LogLevel,
		Environment: cfg.Env,
		LogFile:     "", // Set to a file path if you want to log to file
	}

	if err := logger.Init(loggerCfg); err != nil {
		log.Error().Err(err).Msg("Failed to initialize logger")
	}
}

// Adapter implementations to bridge interface mismatches

// chatServiceAdapter adapts chat.Service to response.ChatServiceInterface
type chatServiceAdapter struct {
	service *chat.Service
}

func (a *chatServiceAdapter) CreateOrGetRoom(ctx context.Context, userID uuid.UUID, req *response.ChatRoomRequest) (*response.ChatRoom, error) {
	// Convert response.ChatRoomRequest to chat.CreateRoomRequest
	chatReq := &chat.CreateRoomRequest{
		RecipientID: &req.RecipientID, // Fix: take address
		CastingID:   req.CastingID,
		Message:     req.Message,
		RoomType:    "direct", // Assume direct for response-casting flow
	}
	if req.CastingID != nil {
		chatReq.RoomType = "casting"
	}

	// Call the actual chat service
	room, err := a.service.CreateOrGetRoom(ctx, userID, chatReq)
	if err != nil {
		return nil, err
	}

	// Fetch members to populate legacy fields
	members, err := a.service.GetMembers(ctx, userID, room.ID)
	if err != nil {
		return nil, err
	}

	// Map members (legacy support logic)
	p1 := uuid.Nil
	p2 := uuid.Nil
	if len(members) > 0 {
		p1 = members[0].UserID
	}
	if len(members) > 1 {
		p2 = members[1].UserID
	}

	// Convert chat.Room to response.ChatRoom
	return &response.ChatRoom{
		ID:             room.ID,
		Participant1ID: p1,
		Participant2ID: p2,
	}, nil
}

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

func (a *authEmployerProfileAdapter) GetByUserID(ctx context.Context, userID uuid.UUID) (*auth.EmployerProfile, error) {
	employerProfile, err := a.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if employerProfile == nil {
		return nil, nil
	}

	description := ""
	if employerProfile.Description.Valid {
		description = employerProfile.Description.String
	}
	website := ""
	if employerProfile.Website.Valid {
		website = employerProfile.Website.String
	}
	contactPerson := ""
	if employerProfile.ContactPerson.Valid {
		contactPerson = employerProfile.ContactPerson.String
	}

	return &auth.EmployerProfile{
		ID:            employerProfile.ID,
		UserID:        employerProfile.UserID,
		CompanyName:   employerProfile.CompanyName,
		Description:   description,
		Website:       website,
		ContactPerson: contactPerson,
		CreatedAt:     employerProfile.CreatedAt,
		UpdatedAt:     employerProfile.UpdatedAt,
	}, nil
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
		Status:         string(payment.Status),
		CreatedAt:      payment.CreatedAt,
	}, nil
}

func (a *subscriptionPaymentAdapter) InitRobokassaPayment(ctx context.Context, req subscription.InitRobokassaPaymentRequest) (*subscription.InitRobokassaPaymentResponse, error) {
	out, err := a.service.InitRobokassaPayment(ctx, payment.InitRobokassaPaymentRequest{
		UserID:         req.UserID,
		SubscriptionID: req.SubscriptionID,
		Amount:         req.Amount,
		Description:    req.Description,
	})
	if err != nil {
		return nil, err
	}
	return &subscription.InitRobokassaPaymentResponse{
		PaymentID:  out.PaymentID,
		InvID:      out.InvID,
		PaymentURL: out.PaymentURL,
		Status:     out.Status,
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

type leadEmployerProfileAdapter struct{ repo profile.EmployerRepository }

func (a *leadEmployerProfileAdapter) Create(ctx context.Context, p *lead.EmployerProfile) error {
	return a.repo.Create(ctx, &profile.EmployerProfile{ID: p.ID, UserID: p.UserID, CompanyName: p.CompanyName, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt})
}

// Chat adapters

type chatAccessChecker struct {
	moderationService    *moderation.Service
	relationshipsService *relationships.Service
}

func (c *chatAccessChecker) CanCommunicate(ctx context.Context, user1, user2 uuid.UUID) error {
	// Check if either user is banned
	if err := c.moderationService.IsUserBanned(ctx, user1); err != nil {
		return err
	}
	if err := c.moderationService.IsUserBanned(ctx, user2); err != nil {
		return err
	}

	// Check if either has blocked the other
	blocked, err := c.relationshipsService.HasBlocked(ctx, user1, user2)
	if err != nil {
		return err
	}
	if blocked {
		return chat.ErrUserBlocked
	}

	blocked, err = c.relationshipsService.HasBlocked(ctx, user2, user1)
	if err != nil {
		return err
	}
	if blocked {
		return chat.ErrUserBlocked
	}

	return nil
}

type chatUploadResolver struct {
	uploadService *uploadDomain.Service
	baseURL       string
}

func (c *chatUploadResolver) IsCommitted(ctx context.Context, uploadID uuid.UUID) (bool, error) {
	upload, err := c.uploadService.GetByID(ctx, uploadID)
	if err != nil {
		return false, err
	}
	return upload.IsCommitted(), nil
}

func (c *chatUploadResolver) GetUploadURL(ctx context.Context, uploadID uuid.UUID) (string, error) {
	upload, err := c.uploadService.GetByID(ctx, uploadID)
	if err != nil {
		return "", err
	}
	return upload.GetURL(c.baseURL), nil
}

func (c *chatUploadResolver) CommitUpload(ctx context.Context, uploadID, userID uuid.UUID) (*chat.AttachmentInfo, error) {
	up, err := c.uploadService.Confirm(ctx, uploadID, userID)
	if err != nil {
		return nil, err
	}
	return &chat.AttachmentInfo{
		UploadID: up.ID,
		URL:      up.PermanentURL,
		FileName: up.OriginalName,
		MimeType: up.MimeType,
		Size:     up.Size.Int64,
	}, nil
}

type chatProfileFetcher struct {
	userRepo       user.Repository
	profileService *profile.Service
}

func (f *chatProfileFetcher) GetParticipantInfo(ctx context.Context, userID uuid.UUID) (*chat.ParticipantInfo, error) {
	u, err := f.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return &chat.ParticipantInfo{
			ID:        userID,
			FirstName: "Deleted",
			LastName:  "User",
		}, nil
	}

	info := &chat.ParticipantInfo{
		ID: userID,
	}

	if u.IsModel() {
		prof, _ := f.profileService.GetModelProfileByUserID(ctx, userID)
		if prof != nil && prof.Name.Valid {
			// Split name for display
			parts := strings.SplitN(prof.Name.String, " ", 2)
			info.FirstName = parts[0]
			if len(parts) > 1 {
				info.LastName = parts[1]
			}
		}
	} else if u.IsEmployer() {
		prof, _ := f.profileService.GetEmployerProfileByUserID(ctx, userID)
		if prof != nil {
			info.FirstName = prof.CompanyName
			if prof.ContactPerson.Valid {
				info.LastName = "(" + prof.ContactPerson.String + ")"
			}
		}
	} else if u.Role == user.RoleAdmin {
		prof, _ := f.profileService.GetAdminProfileByUserID(ctx, userID)
		if prof != nil {
			if prof.Name.Valid {
				info.FirstName = prof.Name.String
			}
			if prof.AvatarURL.Valid {
				info.AvatarURL = &prof.AvatarURL.String
			}
		}
	}

	if info.FirstName == "" {
		info.FirstName = u.Email
	}

	return info, nil
}

type relationshipProfileFetcher struct {
	userRepo       user.Repository
	profileService *profile.Service
}

func (f *relationshipProfileFetcher) GetUserProfile(ctx context.Context, userID uuid.UUID) (*relationships.UserProfile, error) {
	u, err := f.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return &relationships.UserProfile{
			ID:        userID,
			FirstName: "Unknown",
		}, nil
	}

	prof := &relationships.UserProfile{
		ID: userID,
	}

	if u.IsModel() {
		modelProf, _ := f.profileService.GetModelProfileByUserID(ctx, userID)
		if modelProf != nil && modelProf.Name.Valid {
			parts := strings.SplitN(modelProf.Name.String, " ", 2)
			prof.FirstName = parts[0]
			if len(parts) > 1 {
				prof.LastName = parts[1]
			}
		}
	} else if u.IsEmployer() {
		employerProf, _ := f.profileService.GetEmployerProfileByUserID(ctx, userID)
		if employerProf != nil {
			prof.FirstName = employerProf.CompanyName
			if employerProf.ContactPerson.Valid {
				prof.LastName = employerProf.ContactPerson.String
			}
		}
	}

	if prof.FirstName == "" {
		prof.FirstName = u.Email
	}

	return prof, nil
}
