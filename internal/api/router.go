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
	watchlistHandler := handlers.NewWatchlistHandler(database)
	playlistHandler := handlers.NewPlaylistHandler(database)
	sectionHandler := handlers.NewSectionHandler(database)
	templateHandler := handlers.NewSectionTemplateHandler(database)
	showsHandler := handlers.NewShowsHandler(database)
	extrasHandler := handlers.NewExtrasHandler(database)
	metadataHandler := handlers.NewMetadataHandler(database, cfg)
	channelHandler := handlers.NewChannelHandler(database)
	deployHandler := handlers.NewDeployHandler()
	filesHandler := handlers.NewFilesHandler("/media")

	// Serve web admin interface with aggressive no-cache headers
	serveIndex := func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate, max-age=0")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Header("CDN-Cache-Control", "no-store")
		c.Header("Cloudflare-CDN-Cache-Control", "no-store")
		c.Header("Surrogate-Control", "no-store")
		c.Header("X-Content-Version", "2026012711")
		c.File("./web/index.html")
	}
	router.GET("/", serveIndex)
	router.GET("/index.html", serveIndex)
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

		// Deployment status (public - for deploy app monitoring)
		deploy := api.Group("/deploy")
		{
			deploy.GET("/status", deployHandler.GetStatus)
			deploy.GET("/logs", deployHandler.GetLogs)
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
				library.GET("/stats", libraryHandler.GetStats)
				library.POST("/scan", libraryHandler.TriggerScan)
			}

			// Media
			protected.GET("/media/:id", libraryHandler.GetMedia)

			// Metadata management
			protected.POST("/media/:id/metadata/search", metadataHandler.SearchTMDB)
			protected.PUT("/media/:id/metadata/apply", metadataHandler.ApplyMetadata)
			protected.POST("/media/:id/metadata/refresh", metadataHandler.RefreshMetadata)

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

			// File Browser (for configuring sources)
			files := protected.Group("/files")
			{
				files.GET("", filesHandler.ListDirectory)
				files.GET("/roots", filesHandler.GetRoots)
			}

			// Watchlist
			watchlist := protected.Group("/watchlist")
			{
				watchlist.GET("", watchlistHandler.GetWatchlist)
				watchlist.POST("/:mediaId", watchlistHandler.AddToWatchlist)
				watchlist.DELETE("/:mediaId", watchlistHandler.RemoveFromWatchlist)
				watchlist.GET("/:mediaId/check", watchlistHandler.CheckWatchlist)
			}

			// Mark as watched
			protected.POST("/media/:id/watched", watchlistHandler.MarkAsWatched)

			// Playlists
			playlists := protected.Group("/playlists")
			{
				playlists.GET("", playlistHandler.GetPlaylists)
				playlists.POST("", playlistHandler.CreatePlaylist)
				playlists.GET("/:playlistId", playlistHandler.GetPlaylist)
				playlists.PUT("/:playlistId", playlistHandler.UpdatePlaylist)
				playlists.DELETE("/:playlistId", playlistHandler.DeletePlaylist)
				playlists.POST("/:playlistId/items/:mediaId", playlistHandler.AddToPlaylist)
				playlists.DELETE("/:playlistId/items/:mediaId", playlistHandler.RemoveFromPlaylist)
				playlists.PUT("/:playlistId/reorder", playlistHandler.ReorderPlaylist)
			}

			// Sections
			sections := protected.Group("/sections")
			{
				// Section CRUD
				sections.GET("", sectionHandler.ListSections)
				sections.POST("", sectionHandler.CreateSection)
				sections.PUT("/reorder", sectionHandler.ReorderSections)

				// Section templates
				sections.GET("/templates", templateHandler.GetTemplates)
				sections.POST("/from-template", templateHandler.CreateFromTemplate)

				// Section by slug (must come before :id routes)
				sections.GET("/slug/:slug", sectionHandler.GetSectionBySlug)
				sections.GET("/slug/:slug/media", sectionHandler.GetSectionMediaBySlug)

				// Section by ID
				sections.GET("/:id", sectionHandler.GetSection)
				sections.PUT("/:id", sectionHandler.UpdateSection)
				sections.DELETE("/:id", sectionHandler.DeleteSection)

				// Section media
				sections.GET("/:id/media", sectionHandler.GetSectionMedia)
				sections.POST("/:id/media", sectionHandler.AddMediaToSection)
				sections.DELETE("/:id/media/:mediaId", sectionHandler.RemoveMediaFromSection)

				// Section rules
				sections.GET("/:id/rules", sectionHandler.GetSectionRules)
				sections.POST("/:id/rules", sectionHandler.AddSectionRule)
				sections.DELETE("/:id/rules/:ruleId", sectionHandler.DeleteSectionRule)
			}

			// TV Shows (hierarchical)
			shows := protected.Group("/shows")
			{
				shows.GET("", showsHandler.GetShows)
				shows.GET("/:showId", showsHandler.GetShow)
				shows.GET("/:showId/seasons", showsHandler.GetSeasons)
				shows.GET("/:showId/seasons/:seasonNum", showsHandler.GetSeason)
				shows.GET("/:showId/seasons/:seasonNum/episodes", showsHandler.GetEpisodes)
				shows.GET("/:showId/episodes", showsHandler.GetAllEpisodes)
				shows.GET("/:showId/random", showsHandler.GetRandomEpisode)
				shows.GET("/:showId/seasons/:seasonNum/random", showsHandler.GetRandomEpisodeFromSeason)
			}

			// Episodes (direct access)
			protected.GET("/episodes/:episodeId", showsHandler.GetEpisode)

			// Extras (browsable library)
			extras := protected.Group("/extras")
			{
				extras.GET("", extrasHandler.GetExtras)
				extras.GET("/categories", extrasHandler.GetExtraCategories)
				extras.GET("/category/:category", extrasHandler.GetExtrasByCategory)
				extras.GET("/random", extrasHandler.GetRandomExtra)
				extras.GET("/:id", extrasHandler.GetExtra)
			}

			// Extras by parent media
			protected.GET("/media/:id/extras", extrasHandler.GetMovieExtras)
			shows.GET("/:showId/extras", extrasHandler.GetShowExtras)
			protected.GET("/episodes/:episodeId/extras", extrasHandler.GetEpisodeExtras)

			// Channels (virtual live TV)
			channels := protected.Group("/channels")
			{
				channels.GET("", channelHandler.ListChannels)
				channels.POST("", channelHandler.CreateChannel)
				channels.GET("/:id", channelHandler.GetChannel)
				channels.PUT("/:id", channelHandler.UpdateChannel)
				channels.DELETE("/:id", channelHandler.DeleteChannel)
				channels.GET("/:id/now", channelHandler.GetNowPlaying)
				channels.GET("/:id/schedule", channelHandler.GetSchedule)
				channels.POST("/:id/regenerate", channelHandler.RegenerateSchedule)
				channels.GET("/:id/sources", channelHandler.GetSources)
				channels.POST("/:id/sources", channelHandler.AddSource)
				channels.DELETE("/:id/sources/:sourceId", channelHandler.DeleteSource)
			channels.GET("/:id/show-options/:showId", channelHandler.GetShowOptions)
			}
		}
	}

	return router
}
