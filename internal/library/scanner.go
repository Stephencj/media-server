package library

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/stephencjuliano/media-server/internal/config"
	"github.com/stephencjuliano/media-server/internal/db"
	"github.com/stephencjuliano/media-server/pkg/tmdb"
)

// Scanner handles media library scanning
type Scanner struct {
	db                *db.DB
	cfg               *config.Config
	metadataExtractor *MetadataExtractor
	tmdb              *tmdb.Client
	mu                sync.Mutex
	running           bool
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
		db:                database,
		cfg:               cfg,
		metadataExtractor: NewMetadataExtractor(cfg.FFmpegPath),
		tmdb:              tmdbClient,
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

// isExtrasSource checks if a source path is for extras content
func isExtrasSource(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "special feature") ||
		strings.Contains(lower, "special-feature") ||
		strings.Contains(lower, "specialfeature") ||
		strings.Contains(lower, "commentar") ||
		strings.Contains(lower, "extras") ||
		strings.Contains(lower, "bonus")
}

// ScanSource scans a single media source
func (s *Scanner) ScanSource(source *db.MediaSource) error {
	log.Printf("Scanning source: %s (%s)", source.Name, source.Path)

	// Check if this is an extras source
	if isExtrasSource(source.Path) {
		return s.ScanExtrasSource(source)
	}

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

	// Films and episodes first, so the bonus files kept beside them have
	// something to attach to.
	sort.SliceStable(files, func(a, b int) bool {
		return !isBonusPlacement(PlaceFile(source.Path, files[a])) && isBonusPlacement(PlaceFile(source.Path, files[b]))
	})

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
	// Where a file sits says what it is before its name does. Episodes take
	// their show from the folder, and commentary tracks and bonus clips kept
	// beside a film or in a show's folder are extras, not films.
	switch place := PlaceFile(source.Path, filePath); place.Kind {
	case PlaceEpisode:
		return s.processTVEpisode(filePath, source, place.ShowTitle, 0, place.Season, place.Episode, place.EpisodeTitle)
	case PlaceFilmBonus, PlaceShowBonus:
		return s.processBonusFile(filePath, source, place)
	}

	// Parse filename to extract title, year, and season/episode info
	title, year, mediaType, seasonNum, episodeNum := parseFilename(filePath)

	// If it's a TV episode with season/episode info, use the TV episode processor
	if mediaType == db.MediaTypeTVShow && seasonNum > 0 && episodeNum > 0 {
		return s.processTVEpisode(filePath, source, title, year, seasonNum, episodeNum, "")
	}

	// Check if already in database (for movies)
	if existing, err := s.db.GetMediaByFilePath(filePath); err == nil {
		// Already exists - check if we should refresh metadata
		isFeature := IsMovieFeature(source.Path, filePath)
		if s.tmdb.IsConfigured() && existing.TMDbID == 0 && isFeature {
			// Has no TMDB data yet, refresh it
			s.refreshMetadata(existing)
		}
		if existing.Type == db.MediaTypeMovie && existing.TMDbID != 0 && !isFeature {
			// A recording that is not a film was given one's poster by an
			// earlier scan. It keeps its place but not the borrowed details.
			ClearMovieMatch(existing)
			if err := s.db.UpdateMedia(existing); err != nil {
				log.Printf("Could not clear the match on %s: %v", filePath, err)
			}
		}
		return nil
	}

	// Extract file metadata using the metadata extractor
	mediaFile, err := s.metadataExtractor.ExtractFileMetadata(filePath)
	if err != nil {
		log.Printf("Error extracting metadata for %s: %v", filePath, err)
		return err
	}

	// Create media entry with basic info (for movies)
	media := &db.Media{
		MediaFile: *mediaFile,
		TMDBMetadata: db.TMDBMetadata{
			Title: title,
			Year:  year,
		},
		Type: mediaType,
	}
	media.SourceID = source.ID

	// Enrich with TMDB metadata if available. Episodes that reached here for
	// want of an episode marker, commentary tracks and bonus clips are not
	// films, and matching them only borrows some film's poster.
	if IsMovieFeature(source.Path, filePath) {
		s.enrichWithTMDB(media, title, year, mediaType)
	}

	created, err := s.db.CreateMedia(media)
	if err != nil {
		return err
	}

	// Auto-assign to smart sections
	if err := s.db.AutoAssignMediaToSections(created); err != nil {
		log.Printf("Warning: Failed to auto-assign media to sections: %v", err)
	}

	log.Printf("Added movie: %s (%d)", created.Title, created.Year)
	return nil
}

// processTVEpisode handles TV show episode files with proper hierarchy.
// fileTitle is the episode's title as its file names it, "" when unknown.
func (s *Scanner) processTVEpisode(filePath string, source *db.MediaSource, showTitle string, year, seasonNum, episodeNum int, fileTitle string) error {
	// Check if episode already exists by file path
	if existing, err := s.db.GetEpisodeByFilePath(filePath); err == nil {
		s.retireStaleMovie(filePath, existing.ID, db.MediaTypeEpisode)
		return nil // Already exists
	}

	mediaFile, err := s.fileMetadata(filePath)
	if err != nil {
		log.Printf("Error extracting metadata for episode %s: %v", filePath, err)
		return err
	}

	// Try to find or create the TV show
	var show *db.TVShow
	var tmdbShowID int

	if s.tmdb.IsConfigured() {
		// Search TMDB for the show
		result, err := s.tmdb.SearchTV(showTitle, year)
		if err != nil {
			log.Printf("TMDB TV search failed for %s: %v", showTitle, err)
		} else if result != nil {
			tmdbShowID = result.ID

			// Check if we already have this show by TMDB ID
			show, err = s.db.GetTVShowByTMDBID(tmdbShowID)
			if err != nil {
				// Show doesn't exist, get full details and create it
				details, err := s.tmdb.GetTVDetails(tmdbShowID)
				if err != nil {
					log.Printf("TMDB TV details failed for %s: %v", showTitle, err)
				} else {
					showYear := 0
					if len(details.FirstAirDate) >= 4 {
						showYear, _ = strconv.Atoi(details.FirstAirDate[:4])
					}

					show = &db.TVShow{
						Title:        details.Name,
						OriginalTitle: details.OriginalName,
						Year:         showYear,
						Overview:     details.Overview,
						PosterPath:   details.PosterPath,
						BackdropPath: details.BackdropPath,
						Rating:       details.VoteAverage,
						Genres:       tmdb.GenresToString(details.Genres),
						TMDbID:       details.ID,
						Status:       details.Status,
					}
					if details.ExternalIDs != nil {
						show.IMDbID = details.ExternalIDs.IMDbID
					}

					show, err = s.db.CreateTVShow(show)
					if err != nil {
						log.Printf("Failed to create TV show %s: %v", showTitle, err)
						return err
					}
					log.Printf("Created TV show: %s (TMDB ID: %d)", show.Title, show.TMDbID)
				}
			}
		}
	}

	// If we couldn't find/create via TMDB, create a basic show entry
	if show == nil {
		// Try to find by title
		show, err = s.db.GetTVShowByTitle(showTitle)
		if err != nil {
			// Create basic show entry
			show = &db.TVShow{
				Title: showTitle,
				Year:  year,
			}
			show, err = s.db.CreateTVShow(show)
			if err != nil {
				log.Printf("Failed to create TV show %s: %v", showTitle, err)
				return err
			}
			log.Printf("Created TV show (no TMDB): %s", show.Title)
		}
	}

	// Find or create the season
	season, err := s.db.GetSeasonByNumber(show.ID, seasonNum)
	if err != nil {
		// Season doesn't exist, try to get details from TMDB
		var seasonName, seasonOverview, seasonPoster, seasonAirDate string
		var seasonEpisodeCount int

		if s.tmdb.IsConfigured() && tmdbShowID > 0 {
			seasonDetails, err := s.tmdb.GetTVSeasonDetails(tmdbShowID, seasonNum)
			if err == nil && seasonDetails != nil {
				seasonName = seasonDetails.Name
				seasonOverview = seasonDetails.Overview
				seasonPoster = seasonDetails.PosterPath
				seasonAirDate = seasonDetails.AirDate
				seasonEpisodeCount = len(seasonDetails.Episodes)
			}
		}

		if seasonName == "" {
			seasonName = "Season " + strconv.Itoa(seasonNum)
		}

		season = &db.Season{
			TVShowID:     show.ID,
			SeasonNumber: seasonNum,
			Name:         seasonName,
			Overview:     seasonOverview,
			PosterPath:   seasonPoster,
			AirDate:      seasonAirDate,
			EpisodeCount: seasonEpisodeCount,
		}
		season, err = s.db.CreateSeason(season)
		if err != nil {
			log.Printf("Failed to create season %d for %s: %v", seasonNum, show.Title, err)
			return err
		}
		log.Printf("Created season: %s S%02d", show.Title, seasonNum)
	}

	// Get episode details from TMDB if available
	var episodeTitle, episodeOverview, episodeStillPath, episodeAirDate string
	var episodeRuntime int
	var episodeRating float64

	if s.tmdb.IsConfigured() && tmdbShowID > 0 {
		episodeDetails, err := s.tmdb.GetTVEpisodeDetails(tmdbShowID, seasonNum, episodeNum)
		if err == nil && episodeDetails != nil {
			episodeTitle = episodeDetails.Name
			episodeOverview = episodeDetails.Overview
			episodeStillPath = episodeDetails.StillPath
			episodeAirDate = episodeDetails.AirDate
			episodeRuntime = episodeDetails.Runtime
			episodeRating = episodeDetails.VoteAverage
		}
	}

	// Hand-numbered libraries don't always count the way TMDB does (a season
	// pass numbers its bonus tracks in with the episodes). When the file
	// names an episode and TMDB's episode of that number is called something
	// else, the file is right and TMDB's details belong to another episode.
	if fileTitle != "" && !tmdb.TitlesAgree(fileTitle, episodeTitle) {
		episodeTitle = fileTitle
		episodeOverview, episodeStillPath, episodeAirDate = "", "", ""
		episodeRuntime, episodeRating = 0, 0
	}

	if episodeTitle == "" {
		episodeTitle = "Episode " + strconv.Itoa(episodeNum)
	}

	// Create the episode record
	episode := &db.Episode{
		MediaFile:     *mediaFile,
		TVShowID:      show.ID,
		SeasonID:      season.ID,
		SeasonNumber:  seasonNum,
		EpisodeNumber: episodeNum,
		Title:         episodeTitle,
		Overview:      episodeOverview,
		StillPath:     episodeStillPath,
		AirDate:       episodeAirDate,
		Runtime:       episodeRuntime,
		Rating:        episodeRating,
	}
	episode.SourceID = source.ID

	created, err := s.db.CreateEpisode(episode)
	if err != nil {
		log.Printf("Failed to create episode S%02dE%02d for %s: %v", seasonNum, episodeNum, show.Title, err)
		return err
	}
	s.retireStaleMovie(filePath, created.ID, db.MediaTypeEpisode)

	log.Printf("Added episode: %s S%02dE%02d - %s", show.Title, seasonNum, episodeNum, episodeTitle)
	return nil
}

// processBonusFile stores a commentary track or bonus clip found beside a
// film or in a show's folder as an extra of that film or show.
func (s *Scanner) processBonusFile(filePath string, source *db.MediaSource, place Placement) error {
	if existing, err := s.db.GetExtraByFilePath(filePath); err == nil {
		s.retireStaleMovie(filePath, existing.ID, db.MediaTypeExtra)
		return nil
	}

	mediaFile, err := s.fileMetadata(filePath)
	if err != nil {
		log.Printf("Error extracting metadata for extra %s: %v", filePath, err)
		return err
	}

	extra := &db.Extra{
		Title:     place.Title,
		Category:  place.Category,
		MediaFile: *mediaFile,
	}
	extra.SourceID = source.ID

	if place.Kind == PlaceShowBonus {
		if show, err := s.db.GetTVShowByTitle(place.ShowTitle); err == nil && show != nil {
			extra.TVShowID = &show.ID
		}
	} else if film := s.filmForBonus(source.Path, filePath); film != nil {
		extra.MovieID = &film.ID
	}

	created, err := s.db.CreateExtra(extra)
	if err != nil {
		return err
	}
	s.retireStaleMovie(filePath, created.ID, db.MediaTypeExtra)

	log.Printf("Added extra: %s [%s]", created.Title, created.Category)
	return nil
}

// filmForBonus finds the film a bonus file belongs to: the one in the same
// folder, or in a folder that holds several, the one the file is named after.
func (s *Scanner) filmForBonus(sourcePath, filePath string) *db.Media {
	movies, err := s.db.GetMoviesInFolder(filepath.Dir(filePath))
	if err != nil {
		return nil
	}

	bonusName := lettersOnly(strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)))
	var named, only *db.Media
	namedLength, distinct := 0, map[string]bool{}
	for _, movie := range movies {
		// Other bonus files in the folder may not have been moved yet.
		if !IsMovieFeature(sourcePath, movie.FilePath) {
			continue
		}
		title, _ := MovieNameFromPath(movie.FilePath)
		key := lettersOnly(title)
		if key == "" {
			continue
		}
		if !distinct[key] {
			distinct[key] = true
			only = movie
		}
		if strings.Contains(bonusName, key) && len(key) > namedLength {
			named, namedLength = movie, len(key)
		}
	}

	if named != nil {
		return named
	}
	if len(distinct) == 1 {
		return only
	}
	return nil
}

// fileMetadata returns a file's technical details. A file already in the
// library as a movie keeps the details read then, which saves probing it
// again when it is only being moved to where it belongs.
func (s *Scanner) fileMetadata(filePath string) (*db.MediaFile, error) {
	if stale, err := s.db.GetMediaByFilePath(filePath); err == nil && stale.Type == db.MediaTypeMovie {
		mediaFile := stale.MediaFile
		return &mediaFile, nil
	}
	return s.metadataExtractor.ExtractFileMetadata(filePath)
}

// retireStaleMovie removes the movie row an earlier scan made for a file that
// is really an episode or an extra, carrying over whatever pointed at it.
func (s *Scanner) retireStaleMovie(filePath string, newID int64, newType db.MediaType) {
	stale, err := s.db.GetMediaByFilePath(filePath)
	if err != nil || stale.Type != db.MediaTypeMovie {
		return
	}
	if err := s.db.RetireMovieRow(stale.ID, newID, newType); err != nil {
		log.Printf("Could not retire movie row for %s: %v", filePath, err)
		return
	}
	log.Printf("Moved out of Movies: %s (now %s)", filePath, newType)
}

func isBonusPlacement(place Placement) bool {
	return place.Kind == PlaceFilmBonus || place.Kind == PlaceShowBonus
}

// refreshMetadata updates an existing media item with TMDB data
func (s *Scanner) refreshMetadata(media *db.Media) {
	if !s.tmdb.IsConfigured() {
		return
	}

	// Parse title to get clean search term
	title, year, _, _, _ := parseFilename(media.FilePath)
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
		// Match from the file itself. The stored title and year are whatever
		// an earlier scan made of the name, which is not evidence.
		details, err := MatchMovieFile(s.tmdb, FactsForMovie(media))
		if err != nil || details == nil {
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
		details, err := MatchMovieFile(s.tmdb, FactsForMovie(media))
		if err != nil {
			log.Printf("TMDB match failed for %s: %v", title, err)
			return
		}
		if details == nil {
			log.Printf("No confident TMDB match for %s (%d), keeping file name", title, year)
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

// parseFilename extracts title, year, type, and season/episode numbers from filename
func parseFilename(filePath string) (title string, year int, mediaType db.MediaType, seasonNum int, episodeNum int) {
	filename := filepath.Base(filePath)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	// Extract season/episode FIRST before any cleanup
	// Match S01E01 format (case insensitive)
	tvRegex := regexp.MustCompile(`(?i)[Ss](\d{1,2})[Ee](\d{1,2})`)
	tvMatch := tvRegex.FindStringSubmatch(filename)
	if len(tvMatch) == 3 {
		mediaType = db.MediaTypeTVShow
		seasonNum, _ = strconv.Atoi(tvMatch[1])
		episodeNum, _ = strconv.Atoi(tvMatch[2])
		// Remove pattern from filename for title extraction
		filename = tvRegex.ReplaceAllString(filename, " ")
	}

	// Also support 1x01 format
	if seasonNum == 0 {
		altRegex := regexp.MustCompile(`(\d{1,2})x(\d{1,2})`)
		altMatch := altRegex.FindStringSubmatch(filename)
		if len(altMatch) == 3 {
			mediaType = db.MediaTypeTVShow
			seasonNum, _ = strconv.Atoi(altMatch[1])
			episodeNum, _ = strconv.Atoi(altMatch[2])
			filename = altRegex.ReplaceAllString(filename, " ")
		}
	}

	// Movies have their own parser: the cleanup below is tuned to pulling a
	// show name out from in front of an episode marker.
	if mediaType == "" {
		title, year = MovieNameFromPath(filePath)
		return title, year, db.MediaTypeMovie, 0, 0
	}

	// Remove quality indicators FIRST (before separators become spaces)
	// This prevents "1080p" from being parsed as year "1080"
	qualityRegex := regexp.MustCompile(`(?i)[\.\s_-]?(1080p|720p|480p|2160p|4k|uhd|hdr|bluray|bdrip|webrip|web-dl|hdtv|dvdrip|x264|x265|hevc|h264|h265|aac|ac3|dts|HD)[\.\s_-]?`)
	filename = qualityRegex.ReplaceAllString(filename, " ")

	// Replace common separators with spaces
	filename = strings.ReplaceAll(filename, ".", " ")
	filename = strings.ReplaceAll(filename, "_", " ")
	filename = strings.ReplaceAll(filename, "-", " ")

	// Remove parenthetical quality/format info like "(1080p HD)" or "(HD)"
	parenQualityRegex := regexp.MustCompile(`\([^)]*(?:1080|720|480|2160|HD|p)[^)]*\)`)
	filename = parenQualityRegex.ReplaceAllString(filename, "")

	// Look for year pattern - only match realistic movie years (1900-2099)
	yearRegex := regexp.MustCompile(`\(?(19\d{2}|20\d{2})\)?`)
	yearMatch := yearRegex.FindStringSubmatch(filename)
	if len(yearMatch) > 0 {
		year, _ = strconv.Atoi(yearMatch[1])
		// Remove year from title
		filename = yearRegex.ReplaceAllString(filename, "")
	}

	// Clean up multiple spaces and trim
	spaceRegex := regexp.MustCompile(`\s+`)
	title = spaceRegex.ReplaceAllString(filename, " ")
	title = strings.TrimSpace(title)

	// Remove leading numbers if they look like file ordering (01, 02, etc.)
	leadingNumRegex := regexp.MustCompile(`^0\d\s+`)
	title = leadingNumRegex.ReplaceAllString(title, "")

	// Remove trailing episode titles for cleaner show name extraction
	// e.g., "Breaking Bad Pilot" -> "Breaking Bad"
	if mediaType == db.MediaTypeTVShow && seasonNum > 0 {
		// Try to get just the show name by looking for common patterns
		// This helps with files like "Breaking.Bad.S01E01.Pilot.mkv"
		words := strings.Fields(title)
		if len(words) > 2 {
			// Keep first few words as show title (heuristic)
			// Most show names are 1-4 words
			title = strings.Join(words[:min(len(words), 4)], " ")
		}
	}

	if title == "" {
		title = filepath.Base(filePath)
	}

	return
}

// runtimeMinutes converts a file duration in seconds to whole minutes, the
// unit TMDB reports running times in.
func runtimeMinutes(seconds int) int {
	return (seconds + 30) / 60
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
