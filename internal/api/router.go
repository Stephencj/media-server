package api

import (
	"github.com/gin-gonic/gin"
	"github.com/stephencjuliano/media-server/internal/api/handlers"
	"github.com/stephencjuliano/media-server/internal/api/middleware"
	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
)

// NewRouter creates and configures the Gin router
func NewRouter(database *db.DB, cfg *config.Config) *gin.Engine {
	router := gin.Default()

	// Global middleware
	router.Use(middleware.CORS())
	router.Use(middleware.RequestLogger())

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(database, cfg)
	libraryHandler := handlers.NewLibraryHandler(database, cfg)
	streamHandler := handlers.NewStreamHandler(database, cfg)
	progressHandler := handlers.NewProgressHandler(database)
	sourceHandler := handlers.NewSourceHandler(database)

	// Serve web admin interface
	router.StaticFile("/", "./web/index.html")
	router.StaticFile("/index.html", "./web/index.html")
	router.Static("/assets", "./web/assets")

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	api := router.Group("/api")
	{
		// Authentication (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.JWTAuth(cfg.JWTSecret))
		{
			// Library
			library := protected.Group("/library")
			{
				library.GET("/movies", libraryHandler.GetMovies)
				library.GET("/shows", libraryHandler.GetShows)
				library.GET("/recent", libraryHandler.GetRecent)
				library.POST("/scan", libraryHandler.TriggerScan)
			}

			// Media
			protected.GET("/media/:id", libraryHandler.GetMedia)

			// Streaming
			stream := protected.Group("/stream")
			{
				stream.GET("/:id/manifest.m3u8", streamHandler.GetManifest)
				stream.GET("/:id/segment/:num.ts", streamHandler.GetSegment)
				stream.GET("/:id/subtitles/:lang.vtt", streamHandler.GetSubtitle)
				stream.GET("/:id/direct", streamHandler.DirectPlay)
				stream.DELETE("/:id/transcode", streamHandler.StopTranscode)
			}

			// Progress
			progress := protected.Group("/progress")
			{
				progress.GET("/:mediaId", progressHandler.GetProgress)
				progress.POST("/:mediaId", progressHandler.UpdateProgress)
			}

			// Continue Watching
			protected.GET("/continue-watching", progressHandler.GetContinueWatching)

			// Sources
			sources := protected.Group("/sources")
			{
				sources.GET("", sourceHandler.GetSources)
				sources.POST("", sourceHandler.CreateSource)
				sources.DELETE("/:id", sourceHandler.DeleteSource)
			}
		}
	}

	return router
}
