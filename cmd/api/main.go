package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/drxam/PRO100_Kartochki_Go/internal/config"
	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/drxam/PRO100_Kartochki_Go/internal/handler"
	"github.com/drxam/PRO100_Kartochki_Go/internal/middleware"
	"github.com/drxam/PRO100_Kartochki_Go/internal/repository"
	"github.com/drxam/PRO100_Kartochki_Go/internal/service"
	"github.com/drxam/PRO100_Kartochki_Go/pkg/jwt"
	"github.com/drxam/PRO100_Kartochki_Go/pkg/validator"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	_ "github.com/drxam/PRO100_Kartochki_Go/docs"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           PRO100_Kartochki API
// @version         1.0
// @description     REST API для образовательной платформы «PRO100_Карточки»
// @host            localhost:8080
// @BasePath        /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.Database.DSN)
	if err != nil {
		logger.Fatal("database", zap.Error(err))
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		logger.Fatal("database ping", zap.Error(err))
	}

	db := repository.NewDB(pool)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewRefreshTokenRepository(db)
	resetRepo := repository.NewPasswordResetRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	tagRepo := repository.NewTagRepository(db)
	deckRepo := repository.NewDeckRepository(db)
	cardRepo := repository.NewCardRepository(db)

	bootstrapAdmin(ctx, logger, userRepo, cfg.BootstrapAdmin.Email, cfg.BootstrapAdmin.Password)

	// Фоновая чистка истёкших токенов (refresh + password reset).
	go runCleanupLoop(ctx, logger, tokenRepo, resetRepo, 1*time.Hour)

	jwtManager := jwt.NewManager(jwt.Config{
		AccessSecret:  cfg.JWT.AccessSecret,
		RefreshSecret: cfg.JWT.RefreshSecret,
		AccessTTL:     cfg.JWT.AccessTTL,
		RefreshTTL:    cfg.JWT.RefreshTTL,
	})

	v := validator.New()

	authSvc := service.NewAuthService(userRepo, tokenRepo, resetRepo, jwtManager)
	userSvc := service.NewUserService(userRepo, deckRepo, cardRepo)
	userSvc.SetUploadConfig(cfg.UploadPath, cfg.BaseURL)
	adminSvc := service.NewAdminService(userRepo, tokenRepo)
	categorySvc := service.NewCategoryService(categoryRepo)
	tagSvc := service.NewTagService(tagRepo)
	deckSvc := service.NewDeckService(deckRepo, cardRepo, userRepo, categoryRepo, tagRepo)
	cardSvc := service.NewCardService(cardRepo, deckRepo, categoryRepo, tagRepo)

	handler.SetAuditLogger(logger)

	authHandler := handler.NewAuthHandler(authSvc, v)
	authHandler.SetDevReturnResetToken(cfg.PasswordResetReturnToken)
	userHandler := handler.NewUserHandler(userSvc, v)
	adminHandler := handler.NewAdminHandler(adminSvc, v)
	categoryHandler := handler.NewCategoryHandler(categorySvc, v)
	tagHandler := handler.NewTagHandler(tagSvc, v)
	deckHandler := handler.NewDeckHandler(deckSvc, v)
	cardHandler := handler.NewCardHandler(cardSvc, v)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.CORS(cfg.CORSOrigins))
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logging(logger))

	r.GET("/health", func(c *gin.Context) {
		pingCtx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unavailable",
				"error":  "database",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.Static("/uploads", cfg.UploadPath)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Rate limiting (ТЗ §4.1).
	globalLimiter := middleware.NewRateLimiter(
		rate.Limit(cfg.RateLimit.GlobalRPS),
		cfg.RateLimit.GlobalBurst,
		cfg.RateLimit.IdleTTL,
	)
	authLimiter := middleware.NewRateLimiter(
		rate.Limit(cfg.RateLimit.AuthPerMin/60.0), // RPM → RPS
		cfg.RateLimit.AuthBurst,
		cfg.RateLimit.IdleTTL,
	)

	api := r.Group("/api")
	api.Use(globalLimiter.Middleware())
	{
		// Жёсткий отдельный лимит на публичные auth-эндпоинты — защита от
		// брутфорса логина и форсированных запросов на сброс пароля.
		authPublic := api.Group("/auth")
		authPublic.Use(authLimiter.Middleware())
		{
			authPublic.POST("/register", authHandler.Register)
			authPublic.POST("/login", authHandler.Login)
			authPublic.POST("/refresh", authHandler.Refresh)
			authPublic.POST("/forgot-password", authHandler.ForgotPassword)
			authPublic.POST("/reset-password", authHandler.ResetPassword)
		}

		api.GET("/categories", categoryHandler.List)
		api.GET("/tags", tagHandler.List)
		api.GET("/public/decks", deckHandler.ListPublicPaginated)
		api.GET("/public/decks/:id", deckHandler.GetPublicByID)

		auth := api.Group("")
		auth.Use(middleware.Auth(jwtManager, userRepo))
		{
			auth.POST("/auth/logout", authHandler.Logout)
			auth.GET("/users/me", userHandler.GetProfile)
			auth.PUT("/users/me", userHandler.UpdateProfile)
			auth.POST("/users/me/avatar", userHandler.UploadAvatar)

			auth.POST("/categories", categoryHandler.Create)
			auth.POST("/tags", tagHandler.Create)

			auth.GET("/decks", deckHandler.ListMine)
			auth.POST("/decks", deckHandler.Create)
			auth.GET("/decks/:id", deckHandler.GetByID)
			auth.PUT("/decks/:id", deckHandler.Update)
			auth.DELETE("/decks/:id", deckHandler.Delete)

			auth.GET("/cards", cardHandler.List)
			auth.POST("/cards", cardHandler.Create)
			auth.GET("/decks/:id/cards", cardHandler.ListByDeck)
			auth.POST("/decks/:id/cards", cardHandler.Create)
			auth.GET("/cards/:id", cardHandler.GetByID)
			auth.PUT("/cards/:id", cardHandler.Update)
			auth.DELETE("/cards/:id", cardHandler.Delete)

			// Админский раздел: управление учётными записями (модуль «Пользователи и доступ»).
			admin := auth.Group("/admin")
			admin.Use(middleware.RequireRole(domain.RoleAdmin))
			{
				admin.GET("/users", adminHandler.ListUsers)
				admin.GET("/users/:id", adminHandler.GetUser)
				admin.PATCH("/users/:id/block", adminHandler.BlockUser)
				admin.PATCH("/users/:id/role", adminHandler.SetUserRole)
				admin.DELETE("/users/:id", adminHandler.DeleteUser)
			}
		}
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
		// TLS: запрещаем legacy версии — TLS 1.0/1.1 уже не используются ни одним
		// поддерживаемым клиентом (iOS 12+, современные браузеры) и считаются
		// небезопасными.
		TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	useTLS := cfg.Server.TLSCertFile != "" && cfg.Server.TLSKeyFile != ""
	go func() {
		logger.Info("server started",
			zap.String("port", cfg.Server.Port),
			zap.Bool("tls", useTLS),
		)
		var err error
		if useTLS {
			err = srv.ListenAndServeTLS(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown", zap.Error(err))
	}
	logger.Info("server stopped")
}
