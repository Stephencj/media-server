package library

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
	"github.com/stephencjuliano/media-server/pkg/ffmpeg"
	"github.com/stephencjuliano/media-server/pkg/tmdb"
)

// Scanner handles media library scanning
type Scanner struct {
	db      *db.DB
	cfg     *config.Config
	ffprobe *ffmpeg.FFprobe
	tmdb    *tmdb.Client
	mu      sync.Mutex
	running bool
}

// ScanStatus represents the current scan status
type ScanStatus struct {
	Running     bool   `json:"running"`
	SourceID    int64  `json:"source_id,omitempty"`
	SourceName  string `json:"source_name,omitempty"`
	FilesFound  int    `json:"files_found"`
	FilesScanned int   `json:"files_scanned"`
	CurrentFile string `json:"current_file,omitempty"`
}

// Supported video extensions
var videoExtensions = map[string]bool{
	".mp4":  true,
	".mkv":  true,
	".avi":  true,
	".mov":  true,
	".wmv":  true,
	".m4v":  true,
	".webm": true,
	".flv":  true,
	".ts":   true,
	".m2ts": true,
}

// NewScanner creates a new library scanner
func NewScanner(database *db.DB, cfg *config.Config) *Scanner {
	tmdbClient := tmdb.NewClient(cfg.TMDbAPIKey)
	if tmdbClient.IsConfigured() {
		log.Println("TMDB metadata enrichment enabled")
	} else {
		log.Println("TMDB API key not configured - metadata enrichment disabled")
	}

	return &Scanner{
		db:      database,
		cfg:     cfg,
		ffprobe: ffmpeg.NewFFprobe(cfg.FFmpegPath),
		tmdb:    tmdbClient,
	}
}

// IsRunning returns true if a scan is in progress
func (s *Scanner) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// ScanAll scans all enabled media sources
func (s *Scanner) ScanAll() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	sources, err := s.db.GetAllMediaSources()
	if err != nil {
		return err
	}

	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		if err := s.ScanSource(source); err != nil {
			log.Printf("Error scanning source %s: %v", source.Name, err)
		}
	}

	return nil
}

// ScanSource scans a single media source
func (s *Scanner) ScanSource(source *db.MediaSource) error {
	log.Printf("Scanning source: %s (%s)", source.Name, source.Path)

	// Verify path exists
	info, err := os.Stat(source.Path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return os.ErrInvalid
	}

	// Find all video files
	var files []string
	err = filepath.Walk(source.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if videoExtensions[ext] {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("Found %d video files in %s", len(files), source.Name)

	// Process each file
	for _, file := range files {
		if err := s.processFile(file, source); err != nil {
			log.Printf("Error processing %s: %v", file, err)
		}
	}

	// Update last scan time
	s.db.UpdateMediaSourceLastScan(source.ID)

	return nil
}

func (s *Scanner) processFile(filePath string, source *db.MediaSource) error {
	// Check if already in database
	if existing, err := s.db.GetMediaByFilePath(filePath); err == nil {
		// Already exists - check if we should refresh metadata
		if s.tmdb.IsConfigured() && existing.TMDbID == 0 {
			// Has no TMDB data yet, refresh it
			s.refreshMetadata(existing)
		}
		return nil
	}

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// Parse filename to extract title and year
	title, year, mediaType := parseFilename(filePath)

	// Get video metadata using ffprobe
	metadata, err := s.ffprobe.GetMetadata(filePath)
	if err != nil {
		log.Printf("Warning: Could not get metadata for %s: %v", filePath, err)
		metadata = &ffmpeg.Metadata{}
	}

	// Create media entry with basic info
	media := &db.Media{
		Title:          title,
		Type:           mediaType,
		Year:           year,
		SourceID:       source.ID,
		FilePath:       filePath,
		FileSize:       info.Size(),
		Duration:       metadata.Duration,
		VideoCodec:     metadata.VideoCodec,
		AudioCodec:     metadata.AudioCodec,
		Resolution:     metadata.Resolution,
		AudioTracks:    metadata.AudioTracksJSON,
		SubtitleTracks: metadata.SubtitleTracksJSON,
	}

	// Enrich with TMDB metadata if available
	s.enrichWithTMDB(media, title, year, mediaType)

	_, err = s.db.CreateMedia(media)
	if err != nil {
		return err
	}

	log.Printf("Added: %s (%d)", media.Title, media.Year)
	return nil
}

// refreshMetadata updates an existing media item with TMDB data
func (s *Scanner) refreshMetadata(media *db.Media) {
	if !s.tmdb.IsConfigured() {
		return
	}

	// Parse title to get clean search term
	title, year, _ := parseFilename(media.FilePath)
	if media.Title != "" {
		title = media.Title // Use existing title if available
	}
	if media.Year > 0 {
		year = media.Year
	}

	log.Printf("Refreshing metadata for: %s", title)

	// Create a copy to update
	updated := *media

	if media.Type == db.MediaTypeMovie {
		result, err := s.tmdb.SearchMovie(title, year)
		if err != nil || result == nil {
			return
		}

		details, err := s.tmdb.GetMovieDetails(result.ID)
		if err != nil {
			return
		}

		updated.Title = details.Title
		updated.OriginalTitle = details.OriginalTitle
		updated.Overview = details.Overview
		updated.PosterPath = details.PosterPath
		updated.BackdropPath = details.BackdropPath
		updated.Rating = details.VoteAverage
		updated.Runtime = details.Runtime
		updated.TMDbID = details.ID
		updated.IMDbID = details.IMDbID
		updated.Genres = tmdb.GenresToString(details.Genres)

		if len(details.ReleaseDate) >= 4 {
			if y, err := strconv.Atoi(details.ReleaseDate[:4]); err == nil {
				updated.Year = y
			}
		}

	} else if media.Type == db.MediaTypeTVShow {
		result, err := s.tmdb.SearchTV(title, year)
		if err != nil || result == nil {
			return
		}

		details, err := s.tmdb.GetTVDetails(result.ID)
		if err != nil {
			return
		}

		updated.Title = details.Name
		updated.OriginalTitle = details.OriginalName
		updated.Overview = details.Overview
		updated.PosterPath = details.PosterPath
		updated.BackdropPath = details.BackdropPath
		updated.Rating = details.VoteAverage
		updated.SeasonCount = details.NumberOfSeasons
		updated.EpisodeCount = details.NumberOfEpisodes
		updated.TMDbID = details.ID
		updated.Genres = tmdb.GenresToString(details.Genres)

		if details.ExternalIDs != nil {
			updated.IMDbID = details.ExternalIDs.IMDbID
		}

		if len(details.FirstAirDate) >= 4 {
			if y, err := strconv.Atoi(details.FirstAirDate[:4]); err == nil {
				updated.Year = y
			}
		}
	}

	// Update in database
	if err := s.db.UpdateMedia(&updated); err != nil {
		log.Printf("Failed to update metadata for %s: %v", title, err)
	} else {
		log.Printf("Updated metadata for: %s (%d)", updated.Title, updated.Year)
	}
}

// enrichWithTMDB fetches and applies metadata from TMDB
func (s *Scanner) enrichWithTMDB(media *db.Media, title string, year int, mediaType db.MediaType) {
	if !s.tmdb.IsConfigured() {
		return
	}

	if mediaType == db.MediaTypeMovie {
		// Search for movie
		result, err := s.tmdb.SearchMovie(title, year)
		if err != nil {
			log.Printf("TMDB search failed for %s: %v", title, err)
			return
		}
		if result == nil {
			return
		}

		// Get detailed info
		details, err := s.tmdb.GetMovieDetails(result.ID)
		if err != nil {
			log.Printf("TMDB details failed for %s: %v", title, err)
			return
		}

		// Apply metadata
		media.Title = details.Title
		media.OriginalTitle = details.OriginalTitle
		media.Overview = details.Overview
		media.PosterPath = details.PosterPath
		media.BackdropPath = details.BackdropPath
		media.Rating = details.VoteAverage
		media.Runtime = details.Runtime
		media.TMDbID = details.ID
		media.IMDbID = details.IMDbID
		media.Genres = tmdb.GenresToString(details.Genres)

		// Extract year from release date
		if len(details.ReleaseDate) >= 4 {
			if y, err := strconv.Atoi(details.ReleaseDate[:4]); err == nil {
				media.Year = y
			}
		}

	} else if mediaType == db.MediaTypeTVShow {
		// Search for TV show
		result, err := s.tmdb.SearchTV(title, year)
		if err != nil {
			log.Printf("TMDB search failed for %s: %v", title, err)
			return
		}
		if result == nil {
			return
		}

		// Get detailed info
		details, err := s.tmdb.GetTVDetails(result.ID)
		if err != nil {
			log.Printf("TMDB details failed for %s: %v", title, err)
			return
		}

		// Apply metadata
		media.Title = details.Name
		media.OriginalTitle = details.OriginalName
		media.Overview = details.Overview
		media.PosterPath = details.PosterPath
		media.BackdropPath = details.BackdropPath
		media.Rating = details.VoteAverage
		media.SeasonCount = details.NumberOfSeasons
		media.EpisodeCount = details.NumberOfEpisodes
		media.TMDbID = details.ID
		media.Genres = tmdb.GenresToString(details.Genres)

		if details.ExternalIDs != nil {
			media.IMDbID = details.ExternalIDs.IMDbID
		}

		// Extract year from first air date
		if len(details.FirstAirDate) >= 4 {
			if y, err := strconv.Atoi(details.FirstAirDate[:4]); err == nil {
				media.Year = y
			}
		}
	}
}

// parseFilename extracts title, year, and type from filename
func parseFilename(filePath string) (title string, year int, mediaType db.MediaType) {
	filename := filepath.Base(filePath)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	// Replace common separators with spaces
	filename = strings.ReplaceAll(filename, ".", " ")
	filename = strings.ReplaceAll(filename, "_", " ")
	filename = strings.ReplaceAll(filename, "-", " ")

	// Look for year pattern (4 digits in parentheses or standalone)
	yearRegex := regexp.MustCompile(`\((\d{4})\)|(\d{4})`)
	yearMatch := yearRegex.FindStringSubmatch(filename)
	if len(yearMatch) > 0 {
		for _, m := range yearMatch[1:] {
			if m != "" {
				year, _ = strconv.Atoi(m)
				// Remove year from title
				filename = yearRegex.ReplaceAllString(filename, "")
				break
			}
		}
	}

	// Check for TV show patterns (S01E01, 1x01, etc.)
	tvRegex := regexp.MustCompile(`(?i)S\d{1,2}E\d{1,2}|\d{1,2}x\d{1,2}`)
	if tvRegex.MatchString(filename) {
		mediaType = db.MediaTypeTVShow
	} else {
		mediaType = db.MediaTypeMovie
	}

	// Clean up title
	title = strings.TrimSpace(filename)

	// Remove quality indicators
	qualityRegex := regexp.MustCompile(`(?i)\b(1080p|720p|480p|2160p|4k|uhd|hdr|bluray|bdrip|webrip|web-dl|hdtv|dvdrip|x264|x265|hevc|h264|h265|aac|ac3|dts)\b`)
	title = qualityRegex.ReplaceAllString(title, "")

	// Clean up multiple spaces
	spaceRegex := regexp.MustCompile(`\s+`)
	title = spaceRegex.ReplaceAllString(title, " ")
	title = strings.TrimSpace(title)

	if title == "" {
		title = filepath.Base(filePath)
	}

	return
}
