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
)

// Scanner handles media library scanning
type Scanner struct {
	db      *db.DB
	cfg     *config.Config
	ffprobe *ffmpeg.FFprobe
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
	return &Scanner{
		db:      database,
		cfg:     cfg,
		ffprobe: ffmpeg.NewFFprobe(cfg.FFmpegPath),
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
	if _, err := s.db.GetMediaByFilePath(filePath); err == nil {
		return nil // Already exists
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

	// Create media entry
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

	_, err = s.db.CreateMedia(media)
	if err != nil {
		return err
	}

	log.Printf("Added: %s (%d)", title, year)
	return nil
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
