package library

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/stephencjuliano/media-server/internal/db"
)

// ExtrasParseResult holds the parsed metadata from an extras filename
type ExtrasParseResult struct {
	Title         string
	Category      db.ExtraCategory
	SeasonNumber  *int
	EpisodeNumber *int
	DiscNumber    *int
	ParentName    string // For matching to movie/show
}

// ScanExtrasSource scans a source directory for extras content
func (s *Scanner) ScanExtrasSource(source *db.MediaSource) error {
	log.Printf("Scanning extras source: %s (%s)", source.Name, source.Path)

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

	log.Printf("Found %d extra files in %s", len(files), source.Name)

	// Process each file
	for _, file := range files {
		if err := s.processExtraFile(file, source); err != nil {
			log.Printf("Error processing extra %s: %v", file, err)
		}
	}

	// Update last scan time
	s.db.UpdateMediaSourceLastScan(source.ID)

	return nil
}

// processExtraFile processes a single extras file
func (s *Scanner) processExtraFile(filePath string, source *db.MediaSource) error {
	// Check if already in database
	if _, err := s.db.GetExtraByFilePath(filePath); err == nil {
		return nil // Already exists
	}

	// Extract metadata using MetadataExtractor
	mediaFile, err := s.metadataExtractor.ExtractFileMetadata(filePath)
	if err != nil {
		log.Printf("Error extracting metadata for extra %s: %v", filePath, err)
		return err
	}

	// Parse the filename and directory structure
	parseResult := ParseExtrasFilename(filePath, source.Path)

	// Create extra entry
	extra := &db.Extra{
		Title:         parseResult.Title,
		Category:      parseResult.Category,
		MediaFile:     *mediaFile,
		SeasonNumber:  parseResult.SeasonNumber,
		EpisodeNumber: parseResult.EpisodeNumber,
	}
	extra.SourceID = source.ID

	// Try to link to parent content
	s.linkExtraToParent(extra, parseResult, source.Path, filePath)

	_, err = s.db.CreateExtra(extra)
	if err != nil {
		return err
	}

	log.Printf("Added extra: %s [%s]", extra.Title, extra.Category)
	return nil
}

// ParseExtrasFilename parses extras filename to extract metadata and category
func ParseExtrasFilename(filePath, sourcePath string) ExtrasParseResult {
	filename := filepath.Base(filePath)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	result := ExtrasParseResult{
		Title:    filename,
		Category: db.ExtraCategoryOther,
	}

	// Determine parent name from directory structure
	result.ParentName = extractParentName(filePath, sourcePath)

	// Detect category from filename patterns
	result.Category = detectCategory(filename)

	// Parse TV commentary pattern: ComS01E01 - Episode Title
	comRegex := regexp.MustCompile(`(?i)^Com[Ss](\d{1,2})[Ee](\d{1,2})\s*[-–]\s*(.*)$`)
	if match := comRegex.FindStringSubmatch(filename); len(match) == 4 {
		seasonNum, _ := strconv.Atoi(match[1])
		episodeNum, _ := strconv.Atoi(match[2])
		result.SeasonNumber = &seasonNum
		result.EpisodeNumber = &episodeNum
		result.Title = strings.TrimSpace(match[3])
		if result.Title == "" {
			result.Title = filename
		}
		result.Category = db.ExtraCategoryCommentary
		return result
	}

	// Parse TV deleted scene pattern: DSS01E01 - Title
	dsRegex := regexp.MustCompile(`(?i)^[Dd][Ss][Ss](\d{1,2})[Ee](\d{1,2})\s*[-–]\s*(.*)$`)
	if match := dsRegex.FindStringSubmatch(filename); len(match) == 4 {
		seasonNum, _ := strconv.Atoi(match[1])
		episodeNum, _ := strconv.Atoi(match[2])
		result.SeasonNumber = &seasonNum
		result.EpisodeNumber = &episodeNum
		result.Title = strings.TrimSpace(match[3])
		if result.Title == "" {
			result.Title = filename
		}
		result.Category = db.ExtraCategoryDeletedScene
		return result
	}

	// Parse season/disc pattern: S01D01 - Title (show-level extras)
	sdRegex := regexp.MustCompile(`(?i)^[Ss](\d{1,2})[Dd](\d{1,2})\s*[-–]\s*(.*)$`)
	if match := sdRegex.FindStringSubmatch(filename); len(match) == 4 {
		seasonNum, _ := strconv.Atoi(match[1])
		discNum, _ := strconv.Atoi(match[2])
		result.SeasonNumber = &seasonNum
		result.DiscNumber = &discNum
		result.Title = strings.TrimSpace(match[3])
		if result.Title == "" {
			result.Title = filename
		}
		// Category already detected from filename content
		return result
	}

	// Parse movie extras pattern: HP01, HP02, etc. (e.g., "HP02 - Cast Interviews")
	movieExtraRegex := regexp.MustCompile(`(?i)^[A-Z]{2}\d{2}\s*[-–]\s*(.*)$`)
	if match := movieExtraRegex.FindStringSubmatch(filename); len(match) == 2 {
		result.Title = strings.TrimSpace(match[1])
		if result.Title == "" {
			result.Title = filename
		}
		return result
	}

	// Clean up title
	result.Title = cleanExtraTitle(filename)

	return result
}

// extractParentName gets the parent show/movie name from directory structure
func extractParentName(filePath, sourcePath string) string {
	// Get relative path from source
	relPath, err := filepath.Rel(sourcePath, filePath)
	if err != nil {
		return ""
	}

	// Split into parts
	parts := strings.Split(relPath, string(filepath.Separator))

	// The parent name is typically the first directory after source
	// e.g., TV Show Commentaries/Psych/ComS01E01.m4v -> "Psych"
	// e.g., Movie Special Features/Avatar/Featurette.m4v -> "Avatar"
	if len(parts) >= 2 {
		return parts[0]
	}

	return ""
}

// detectCategory determines the category from filename content
func detectCategory(filename string) db.ExtraCategory {
	lower := strings.ToLower(filename)

	// Commentary patterns
	if strings.HasPrefix(lower, "com") ||
		strings.Contains(lower, "commentary") ||
		strings.Contains(lower, "audio commentary") {
		return db.ExtraCategoryCommentary
	}

	// Deleted scenes
	if strings.HasPrefix(lower, "ds") ||
		strings.Contains(lower, "deleted") ||
		strings.Contains(lower, "alternate") ||
		strings.Contains(lower, "extended") {
		return db.ExtraCategoryDeletedScene
	}

	// Gag reel / bloopers
	if strings.Contains(lower, "blooper") ||
		strings.Contains(lower, "gag") ||
		strings.Contains(lower, "outtake") ||
		strings.Contains(lower, "bloopers") {
		return db.ExtraCategoryGagReel
	}

	// Interviews
	if strings.Contains(lower, "interview") ||
		strings.Contains(lower, "cast") && strings.Contains(lower, "talks") {
		return db.ExtraCategoryInterview
	}

	// Music video
	if strings.Contains(lower, "music video") ||
		strings.Contains(lower, "musicvideo") {
		return db.ExtraCategoryMusicVideo
	}

	// Behind the scenes
	if strings.Contains(lower, "behind") ||
		strings.Contains(lower, "making of") ||
		strings.Contains(lower, "making-of") ||
		strings.Contains(lower, "bts") {
		return db.ExtraCategoryBehindTheScenes
	}

	// Featurettes
	if strings.Contains(lower, "featurette") ||
		strings.Contains(lower, "epk") ||
		strings.Contains(lower, "special feature") {
		return db.ExtraCategoryFeaturette
	}

	return db.ExtraCategoryOther
}

// stripCommonSuffixes removes common extras-related suffixes from a title to get the base show/movie name
// e.g., "Psych Commentary" -> "Psych", "Avatar Special Features" -> "Avatar"
func stripCommonSuffixes(title string) string {
	suffixes := []string{
		" commentary", " commentaries",
		" audio commentary", " audio commentaries",
		" featurette", " featurettes",
		" special feature", " special features",
		" deleted scene", " deleted scenes",
		" behind the scenes",
		" extras", " extra",
		" bonus", " bonus feature", " bonus features",
		" making of", " the making of",
		" interviews", " interview",
		" gag reel", " bloopers", " outtakes",
	}
	lower := strings.ToLower(title)
	for _, suffix := range suffixes {
		if strings.HasSuffix(lower, suffix) {
			return strings.TrimSpace(title[:len(title)-len(suffix)])
		}
	}
	return title
}

// cleanExtraTitle cleans up the extra title
func cleanExtraTitle(filename string) string {
	// Remove common prefixes
	title := filename

	// Remove quality indicators
	qualityRegex := regexp.MustCompile(`(?i)[\.\s_-]?(1080p|720p|480p|2160p|4k|uhd|hdr|bluray|bdrip|webrip|web-dl|hdtv|dvdrip|x264|x265|hevc|h264|h265|aac|ac3|dts|HD)[\.\s_-]?`)
	title = qualityRegex.ReplaceAllString(title, " ")

	// Replace separators
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Clean up multiple spaces
	spaceRegex := regexp.MustCompile(`\s+`)
	title = spaceRegex.ReplaceAllString(title, " ")
	title = strings.TrimSpace(title)

	if title == "" {
		title = filename
	}

	return title
}

// linkExtraToParent attempts to link an extra to its parent movie/show/episode
func (s *Scanner) linkExtraToParent(extra *db.Extra, parseResult ExtrasParseResult, sourcePath, filePath string) {
	parentName := parseResult.ParentName
	if parentName == "" {
		return
	}

	// Strip common suffixes for better matching
	// e.g., "Psych Commentary" -> "Psych", "Avatar Special Features" -> "Avatar"
	cleanParentName := stripCommonSuffixes(parentName)

	// Determine source type from path
	sourcePathLower := strings.ToLower(sourcePath)
	isTV := strings.Contains(sourcePathLower, "tv") ||
		strings.Contains(sourcePathLower, "show") ||
		strings.Contains(sourcePathLower, "series")
	isMovie := strings.Contains(sourcePathLower, "movie") ||
		strings.Contains(sourcePathLower, "film")

	// Try to link to TV show
	if isTV || (!isMovie && parseResult.SeasonNumber != nil) {
		// Try exact match with cleaned name first
		show, err := s.db.GetTVShowByTitle(cleanParentName)
		if err == nil && show != nil {
			extra.TVShowID = &show.ID

			// If we have episode info, try to link to specific episode
			if parseResult.SeasonNumber != nil && parseResult.EpisodeNumber != nil {
				episode, err := s.db.GetEpisodeByNumber(show.ID, *parseResult.SeasonNumber, *parseResult.EpisodeNumber)
				if err == nil && episode != nil {
					extra.EpisodeID = &episode.ID
					extra.SeasonNumber = parseResult.SeasonNumber
					extra.EpisodeNumber = parseResult.EpisodeNumber
					log.Printf("Linked extra to episode: %s S%02dE%02d", show.Title, *parseResult.SeasonNumber, *parseResult.EpisodeNumber)
				}
			} else {
				log.Printf("Linked extra to TV show: %s", show.Title)
			}
			return
		}

		// Fuzzy match with bidirectional search using cleaned name
		shows, err := s.db.SearchTVShowsFuzzy(cleanParentName, 5)
		if err == nil && len(shows) > 0 {
			// Use first match (ordered by title length DESC, so longest/most specific match first)
			extra.TVShowID = &shows[0].ID
			if parseResult.SeasonNumber != nil && parseResult.EpisodeNumber != nil {
				episode, err := s.db.GetEpisodeByNumber(shows[0].ID, *parseResult.SeasonNumber, *parseResult.EpisodeNumber)
				if err == nil && episode != nil {
					extra.EpisodeID = &episode.ID
					extra.SeasonNumber = parseResult.SeasonNumber
					extra.EpisodeNumber = parseResult.EpisodeNumber
				}
			}
			log.Printf("Fuzzy matched extra to TV show: %s (from '%s')", shows[0].Title, parentName)
			return
		}

		// If cleaned name differs from original, also try with original name
		if cleanParentName != parentName {
			shows, err := s.db.SearchTVShowsFuzzy(parentName, 5)
			if err == nil && len(shows) > 0 {
				extra.TVShowID = &shows[0].ID
				if parseResult.SeasonNumber != nil && parseResult.EpisodeNumber != nil {
					episode, err := s.db.GetEpisodeByNumber(shows[0].ID, *parseResult.SeasonNumber, *parseResult.EpisodeNumber)
					if err == nil && episode != nil {
						extra.EpisodeID = &episode.ID
						extra.SeasonNumber = parseResult.SeasonNumber
						extra.EpisodeNumber = parseResult.EpisodeNumber
					}
				}
				log.Printf("Fuzzy matched extra to TV show (original): %s (from '%s')", shows[0].Title, parentName)
				return
			}
		}
	}

	// Try to link to movie
	if isMovie || !isTV {
		// Try with cleaned name first using fuzzy search
		movies, err := s.db.SearchMediaFuzzy(cleanParentName, db.MediaTypeMovie, 5)
		if err == nil && len(movies) > 0 {
			extra.MovieID = &movies[0].ID
			log.Printf("Linked extra to movie: %s (from '%s')", movies[0].Title, parentName)
			return
		}

		// Try without year suffix (e.g., "Avatar (2009)" -> "Avatar")
		cleanName := regexp.MustCompile(`\s*\(\d{4}\)\s*$`).ReplaceAllString(cleanParentName, "")
		if cleanName != cleanParentName {
			movies, err := s.db.SearchMediaFuzzy(cleanName, db.MediaTypeMovie, 5)
			if err == nil && len(movies) > 0 {
				extra.MovieID = &movies[0].ID
				log.Printf("Linked extra to movie (cleaned): %s (from '%s')", movies[0].Title, parentName)
				return
			}
		}
	}

	log.Printf("Could not link extra to parent: %s (cleaned: %s)", parentName, cleanParentName)
}
