package main

import (
	"log"

	"daily-notes-api/internal/config"
	"daily-notes-api/internal/database"
	"daily-notes-api/internal/handlers"
	"daily-notes-api/internal/middleware"
	"daily-notes-api/internal/repository"
	"daily-notes-api/pkg/auth"

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

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	noteRepo := repository.NewNoteRepository(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, jwtManager)
	noteHandler := handlers.NewNoteHandler(noteRepo)

	// Setup Gin router
	r := gin.New()

	// Add middleware
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
	r.Use(middleware.ErrorHandler())
	r.Use(gin.Recovery())

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "daily-notes-api",
		})
	})

	// Public API routes (no authentication required)
	api := r.Group("/api/v1")
	{
		// Authentication routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
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
