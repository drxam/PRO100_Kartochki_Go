package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pro100kartochki/mozgoemka/internal/config"
	"github.com/pro100kartochki/mozgoemka/internal/handler"
	"github.com/pro100kartochki/mozgoemka/internal/middleware"
	"github.com/pro100kartochki/mozgoemka/internal/repository"
	"github.com/pro100kartochki/mozgoemka/internal/service"
	"github.com/pro100kartochki/mozgoemka/pkg/jwt"
	"github.com/pro100kartochki/mozgoemka/pkg/validator"
	"go.uber.org/zap"

	_ "github.com/pro100kartochki/mozgoemka/docs"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           МозгоЁмка API
// @version         1.0
// @description     REST API для приложения карточек МозгоЁмка
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

	ctx := context.Background()
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
	categoryRepo := repository.NewCategoryRepository(db)
	tagRepo := repository.NewTagRepository(db)
	deckRepo := repository.NewDeckRepository(db)
	cardRepo := repository.NewCardRepository(db)

	jwtManager := jwt.NewManager(jwt.Config{
		AccessSecret:  cfg.JWT.AccessSecret,
		RefreshSecret: cfg.JWT.RefreshSecret,
		AccessTTL:     cfg.JWT.AccessTTL,
		RefreshTTL:    cfg.JWT.RefreshTTL,
	})

	v := validator.New()

	authSvc := service.NewAuthService(userRepo, tokenRepo, jwtManager)
	userSvc := service.NewUserService(userRepo, deckRepo, cardRepo)
	userSvc.SetUploadConfig(cfg.UploadPath, cfg.BaseURL)
	categorySvc := service.NewCategoryService(categoryRepo)
	tagSvc := service.NewTagService(tagRepo)
	deckSvc := service.NewDeckService(deckRepo, cardRepo, userRepo, categoryRepo, tagRepo)
	cardSvc := service.NewCardService(cardRepo, deckRepo, categoryRepo, tagRepo)

	authHandler := handler.NewAuthHandler(authSvc, v)
	userHandler := handler.NewUserHandler(userSvc, v)
	categoryHandler := handler.NewCategoryHandler(categorySvc, v)
	tagHandler := handler.NewTagHandler(tagSvc, v)
	deckHandler := handler.NewDeckHandler(deckSvc, v)
	cardHandler := handler.NewCardHandler(cardSvc, v)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.CORS())
	r.Use(gin.Recovery())
	r.Use(middleware.Logging(logger))

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.Static("/uploads", cfg.UploadPath)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api")
	{
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/refresh", authHandler.Refresh)
		api.POST("/auth/forgot-password", authHandler.ForgotPassword)

		api.GET("/categories", categoryHandler.List)
		api.GET("/tags", tagHandler.List)
		api.GET("/public/decks", deckHandler.ListPublicPaginated)
		api.GET("/public/decks/:id", deckHandler.GetPublicByID)

		auth := api.Group("")
		auth.Use(middleware.Auth(jwtManager))
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
			auth.GET("/decks/:deck_id/cards", cardHandler.ListByDeck)
			auth.POST("/decks/:deck_id/cards", cardHandler.Create)
			auth.GET("/cards/:id", cardHandler.GetByID)
			auth.PUT("/cards/:id", cardHandler.Update)
			auth.DELETE("/cards/:id", cardHandler.Delete)
		}
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}
	go func() {
		logger.Info("server started", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
