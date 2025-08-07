package main

import (
	"log"
	"time"

	"daily-notes-api/internal/config"
	"daily-notes-api/internal/database"
	"daily-notes-api/internal/handlers"
	"daily-notes-api/internal/middleware"
	"daily-notes-api/internal/repository"
	"daily-notes-api/pkg/auth"
	"daily-notes-api/pkg/cache"
	"daily-notes-api/pkg/email"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Connect to database
	db, err := database.NewMySQLConnection(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg)

	// Initialize Email service
	var emailService *email.EmailService
	if cfg.SMTPUsername != "" && cfg.SMTPPassword != "" {
		emailService = email.NewEmailService(cfg)
		log.Printf("Email service initialized with SMTP host: %s", cfg.SMTPHost)
	} else {
		log.Printf("Warning: SMTP credentials not provided, email functionality disabled")
	}

	// Initialize Cache service
	var cacheService *cache.CacheService
	if cfg.CacheEnabled {
		cacheConfig := cache.CacheConfig{
			Host:       cfg.RedisHost,
			Port:       cfg.RedisPort,
			Password:   cfg.RedisPassword,
			DB:         cfg.RedisDB,
			DefaultTTL: time.Duration(cfg.CacheTTLMinutes) * time.Minute,
		}

		cacheService = cache.NewCacheService(cacheConfig)
		if cacheService != nil {
			log.Printf("Cache service initialized with Redis at %s:%s", cfg.RedisHost, cfg.RedisPort)
			// Ensure cache is properly closed on application exit
			defer func() {
				if err := cacheService.Close(); err != nil {
					log.Printf("Failed to close cache service: %v", err)
				}
			}()
		} else {
			log.Printf("Warning: Cache service failed to initialize, running without cache")
		}
	} else {
		log.Printf("Cache service disabled via configuration")
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	noteRepo := repository.NewNoteRepository(db)

	// Initialize handlers (pass cache service to note handler)
	authHandler := handlers.NewAuthHandler(userRepo, jwtManager, emailService, cfg.AppName, cfg.AppURL)
	noteHandler := handlers.NewNoteHandler(noteRepo, cacheService)

	// Setup Gin router
	r := gin.New()

	// Add middleware
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
	r.Use(middleware.ErrorHandler())
	r.Use(gin.Recovery())

	// Health check endpoint (include cache status)
	r.GET("/health", func(c *gin.Context) {
		healthData := gin.H{
			"status":  "ok",
			"service": "daily-notes-api",
			"cache":   "disabled",
		}

		// Check cache health if enabled
		if cacheService != nil {
			if err := cacheService.Health(c); err != nil {
				healthData["cache"] = "error"
				healthData["cache_error"] = err.Error()
			} else {
				healthData["cache"] = "healthy"
			}
		}

		c.JSON(200, healthData)
	})

	// Public API routes (no authentication required)
	api := r.Group("/api/v1")
	{
		// Authentication routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/test-email", authHandler.TestEmail)
		}
	}

	// Protected API routes (authentication required)
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(jwtManager))
	{
		// User profile routes
		user := protected.Group("/user")
		{
			user.GET("/profile", authHandler.GetProfile)
			user.PUT("/profile", authHandler.UpdateProfile)
			user.POST("/change-password", authHandler.ChangePassword)
		}

		// Notes routes
		notes := protected.Group("/notes")
		{
			notes.POST("", noteHandler.CreateNote)
			notes.GET("", noteHandler.GetNotes)
			notes.GET("/:id", noteHandler.GetNote)
			notes.PUT("/:id", noteHandler.UpdateNote)
			notes.DELETE("/:id", noteHandler.DeleteNote)
		}

		// Categories route
		protected.GET("/categories", noteHandler.GetCategories)
	}

	// Start server
	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
