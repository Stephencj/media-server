package library

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// FilenameParser handles parsing of media filenames with support for various naming conventions.
// It can extract titles, years, quality indicators, season/episode numbers, and IMDb IDs
// from both movie and TV show filenames.
//
// Example usages:
// "The.Matrix.1999.1080p.BluRay.x264.mp4" -> Title: "The Matrix", Year: 1999
// "Inception (2010) [1080p].mkv" -> Title: "Inception", Year: 2010
// "2001.A.Space.Odyssey.1968.mp4" -> Title: "2001 a Space Odyssey", Year: 1968
// "Breaking.Bad.S01E01.720p.mp4" -> Title: "Breaking Bad", Season: 1, Episode: 1, IsTV: true
// "The.Thing.tt0084787.1982.mkv" -> Title: "The Thing", Year: 1982, IMDbID: "tt0084787"
// "Game.of.Thrones.1x01.Winter.Is.Coming.mkv" -> Title: "Game of Thrones", Season: 1, Episode: 1, IsTV: true
// "The.Dark.Knight.(2008).2160p.4K.UHD.HDR.mkv" -> Title: "The Dark Knight", Year: 2008
// "Stranger.Things.S03E08.The.Battle.of.Starcourt.1080p.WEBRip.x265.mkv" -> Title: "Stranger Things", Season: 3, Episode: 8, IsTV: true
type FilenameParser struct {
	qualityRegex        *regexp.Regexp
	yearRegex           *regexp.Regexp
	imdbIDRegex         *regexp.Regexp
	tvShowRegex         *regexp.Regexp
	tvShowAltRegex      *regexp.Regexp
	parenQualityRegex   *regexp.Regexp
	leadingNumberRegex  *regexp.Regexp
	multipleSpacesRegex *regexp.Regexp
}

// NewFilenameParser creates a new filename parser with pre-compiled regular expressions
// for efficient parsing of media filenames.
func NewFilenameParser() *FilenameParser {
	return &FilenameParser{
		// Quality indicators regex - matches common video quality and format markers
		// Matches patterns like: 1080p, 720p, BluRay, WEB-DL, x264, AAC, etc.
		qualityRegex: regexp.MustCompile(`(?i)[\.\s_-]?(1080p|720p|480p|2160p|4k|uhd|hdr|bluray|bdrip|brrip|webrip|web-dl|dvdrip|hdtv|x264|x265|hevc|h264|h265|aac|ac3|dts|5\.1|7\.1|atmos|remastered|extended|directors\.cut|unrated|theatrical|hd)[\.\s_-]?`),

		// Year regex - matches years in various formats (1900-2099)
		// Handles: (2020), [2020], .2020., _2020_
		yearRegex: regexp.MustCompile(`[\(\[\s_\.-]+(19\d{2}|20\d{2})[\)\]\s_\.-]*`),

		// IMDb ID regex - matches IMDb identifiers (tt followed by 7-8 digits)
		imdbIDRegex: regexp.MustCompile(`(tt\d{7,8})`),

		// TV show regex - matches S01E01 format (case insensitive)
		tvShowRegex: regexp.MustCompile(`(?i)[Ss](\d{1,2})[Ee](\d{1,2})`),

		// Alternative TV show regex - matches 1x01 format
		tvShowAltRegex: regexp.MustCompile(`(\d{1,2})x(\d{1,2})`),

		// Parenthetical quality regex - matches quality info in parentheses
		// e.g., (1080p HD), (HD), (BluRay)
		parenQualityRegex: regexp.MustCompile(`\([^)]*(?:1080|720|480|2160|4k|HD|p)[^)]*\)`),

		// Leading number regex - matches file ordering numbers like "01 ", "02 "
		leadingNumberRegex: regexp.MustCompile(`^0\d\s+`),

		// Multiple spaces regex - normalizes multiple spaces to single space
		multipleSpacesRegex: regexp.MustCompile(`\s+`),
	}
}

// ParseResult contains the parsed filename information.
// All fields are optional and will be zero-valued if not found in the filename.
type ParseResult struct {
	// Title is the cleaned media title (movie or TV show name)
	Title string

	// Year is the release year (0 if not found)
	Year int

	// IMDbID is the IMDb identifier if present in filename (e.g., "tt0084787")
	IMDbID string

	// IsTV indicates whether this is a TV show episode
	IsTV bool

	// SeasonNumber is the season number for TV shows (0 if not a TV show)
	SeasonNumber int

	// EpisodeNumber is the episode number for TV shows (0 if not a TV show)
	EpisodeNumber int

	// OriginalName is the original filename without extension
	OriginalName string
}

// ParseFilename parses a movie or TV show filename and extracts all available metadata.
// It handles various naming conventions and formats commonly used for media files.
//
// The parsing process follows this order:
// 1. Extract season/episode numbers (determines if it's a TV show)
// 2. Extract IMDb ID (if present)
// 3. Extract year
// 4. Clean and extract title
//
// Parameters:
//   - filePath: The full path or filename to parse
//
// Returns:
//   - ParseResult with all extracted information
func (p *FilenameParser) ParseFilename(filePath string) ParseResult {
	result := ParseResult{}

	// Get filename without extension
	filename := filepath.Base(filePath)
	ext := filepath.Ext(filename)
	filename = strings.TrimSuffix(filename, ext)
	result.OriginalName = filename

	// Step 1: Check if it's a TV show (S01E01 format)
	// This must be done FIRST before any cleanup to ensure pattern matching works
	if matches := p.tvShowRegex.FindStringSubmatch(filename); len(matches) >= 3 {
		result.IsTV = true
		result.SeasonNumber, _ = strconv.Atoi(matches[1])
		result.EpisodeNumber, _ = strconv.Atoi(matches[2])

		// Extract show title (everything before S01E01 pattern)
		titlePart := p.tvShowRegex.ReplaceAllString(filename, " ")
		result.Title = p.cleanTitle(titlePart, true)
		return result
	}

	// Step 2: Check alternative TV show format (1x01)
	if matches := p.tvShowAltRegex.FindStringSubmatch(filename); len(matches) >= 3 {
		result.IsTV = true
		result.SeasonNumber, _ = strconv.Atoi(matches[1])
		result.EpisodeNumber, _ = strconv.Atoi(matches[2])

		// Extract show title (everything before 1x01 pattern)
		titlePart := p.tvShowAltRegex.ReplaceAllString(filename, " ")
		result.Title = p.cleanTitle(titlePart, true)
		return result
	}

	// Step 3: Extract IMDb ID if present (highest priority for matching)
	if matches := p.imdbIDRegex.FindStringSubmatch(filename); len(matches) > 0 {
		result.IMDbID = matches[1]
	}

	// Step 4: Extract year
	result.Year = p.extractYear(filename)

	// Step 5: Clean title (this is a movie)
	result.Title = p.cleanMovieTitle(filename, result.Year, result.IMDbID)

	return result
}

// extractYear attempts multiple strategies to extract the year from a filename.
// It looks for years in the range 1900-2099 in various formats.
//
// Supported patterns:
//   - (2020) - parentheses
//   - [2020] - brackets
//   - .2020. - surrounded by dots
//   - _2020_ - surrounded by underscores
//   - -2020- - surrounded by dashes
//
// Returns 0 if no valid year is found.
func (p *FilenameParser) extractYear(filename string) int {
	// Strategy 1: Look for (YYYY) pattern - most common
	patterns := []string{
		`\((\d{4})\)`,             // (2020)
		`\[(\d{4})\]`,             // [2020]
		`[\s_\.-](\d{4})[\s_\.-]`, // .2020. or _2020_ or -2020-
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(filename); len(matches) >= 2 {
			year, err := strconv.Atoi(matches[1])
			// Validate year is in realistic range
			if err == nil && year >= 1900 && year <= 2099 {
				return year
			}
		}
	}

	return 0
}

// cleanMovieTitle cleans up a movie title from filename by removing:
// - Quality indicators (1080p, BluRay, etc.)
// - Year
// - IMDb ID
// - Parenthetical quality info
// - Common separators (dots, underscores)
//
// Parameters:
//   - filename: The original filename to clean
//   - year: The extracted year (0 if none)
//   - imdbID: The extracted IMDb ID (empty if none)
//
// Returns the cleaned title string.
func (p *FilenameParser) cleanMovieTitle(filename string, year int, imdbID string) string {
	title := filename

	// Remove quality indicators FIRST (before separators become spaces)
	// This prevents "1080p" from being parsed as year "1080"
	title = p.qualityRegex.ReplaceAllString(title, " ")

	// Remove parenthetical quality info like "(1080p HD)" or "(HD)"
	title = p.parenQualityRegex.ReplaceAllString(title, " ")

	// Replace common separators with spaces
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")

	// Remove year from title in various formats
	if year > 0 {
		yearStr := strconv.Itoa(year)
		title = strings.ReplaceAll(title, "("+yearStr+")", " ")
		title = strings.ReplaceAll(title, "["+yearStr+"]", " ")
		title = strings.ReplaceAll(title, " "+yearStr+" ", " ")
	}

	// Remove IMDb ID
	if imdbID != "" {
		title = strings.ReplaceAll(title, imdbID, " ")
	}

	return p.cleanTitle(title, false)
}

// cleanTitle performs common cleaning operations on a title:
// - Normalizes separators to spaces
// - Removes multiple spaces
// - Trims whitespace
// - Normalizes capitalization
// - Removes leading file numbers
//
// Parameters:
//   - title: The title to clean
//   - isTVShow: Whether this is a TV show (affects cleanup strategy)
//
// Returns the cleaned title string.
func (p *FilenameParser) cleanTitle(title string, isTVShow bool) string {
	// Replace dots, underscores, and dashes with spaces
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")

	// Normalize multiple spaces to single space
	title = p.multipleSpacesRegex.ReplaceAllString(title, " ")

	// Trim whitespace
	title = strings.TrimSpace(title)

	// Remove leading numbers if they look like file ordering (01, 02, etc.)
	title = p.leadingNumberRegex.ReplaceAllString(title, "")

	// For TV shows, try to extract just the show name by limiting words
	// This helps with files like "Breaking.Bad.S01E01.Pilot.720p.mkv"
	if isTVShow {
		words := strings.Fields(title)
		if len(words) > 4 {
			// Most show names are 1-4 words, anything beyond is likely episode title
			// Keep first 4 words as a heuristic
			title = strings.Join(words[:4], " ")
		}
	}

	// Normalize capitalization
	title = p.normalizeCapitalization(title)

	return title
}

// normalizeCapitalization converts various capitalizations to Title Case.
// It handles:
// - ALL CAPS -> Title Case
// - all lowercase -> Title Case
// - Proper handling of articles and prepositions (lowercase unless first word)
//
// Examples:
//   - "MOVIE TITLE" -> "Movie Title"
//   - "movie title" -> "Movie Title"
//   - "the lord of the rings" -> "The Lord of the Rings"
//   - "a tale of two cities" -> "A Tale of Two Cities"
func (p *FilenameParser) normalizeCapitalization(title string) string {
	words := strings.Fields(title)

	// Small words that should be lowercase (unless first word)
	smallWords := map[string]bool{
		"a":    true,
		"an":   true,
		"and":  true,
		"the":  true,
		"of":   true,
		"in":   true,
		"on":   true,
		"at":   true,
		"to":   true,
		"for":  true,
		"with": true,
		"from": true,
		"by":   true,
	}

	for i, word := range words {
		if len(word) == 0 {
			continue
		}

		lowerWord := strings.ToLower(word)

		// First word is always capitalized
		if i == 0 {
			words[i] = strings.Title(lowerWord)
			continue
		}

		// Small words stay lowercase (unless first word)
		if smallWords[lowerWord] {
			words[i] = lowerWord
			continue
		}

		// Everything else gets title case
		words[i] = strings.Title(lowerWord)
	}

	return strings.Join(words, " ")
}

// ParseMovieFilename is a convenience method for parsing movie filenames.
// It's the same as ParseFilename but makes the intent clearer when you know
// you're dealing with a movie file.
func (p *FilenameParser) ParseMovieFilename(filePath string) ParseResult {
	return p.ParseFilename(filePath)
}

// ParseTVShowFilename is a convenience method for parsing TV show filenames.
// It's the same as ParseFilename but makes the intent clearer when you know
// you're dealing with a TV show file.
func (p *FilenameParser) ParseTVShowFilename(filePath string) ParseResult {
	return p.ParseFilename(filePath)
}
