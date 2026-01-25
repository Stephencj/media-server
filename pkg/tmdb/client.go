package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.themoviedb.org/3"

// Client handles TMDB API requests
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new TMDB client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// IsConfigured returns true if API key is set
func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}

// MovieResult represents a movie search result
type MovieResult struct {
	ID           int     `json:"id"`
	Title        string  `json:"title"`
	OriginalTitle string `json:"original_title"`
	Overview     string  `json:"overview"`
	ReleaseDate  string  `json:"release_date"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
	VoteAverage  float64 `json:"vote_average"`
	Popularity   float64 `json:"popularity"`
	GenreIDs     []int   `json:"genre_ids"`
}

// MovieSearchResult is an alias for MovieResult for consistency
type MovieSearchResult = MovieResult

// MovieDetails represents detailed movie info
type MovieDetails struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Overview      string  `json:"overview"`
	ReleaseDate   string  `json:"release_date"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	VoteAverage   float64 `json:"vote_average"`
	Runtime       int     `json:"runtime"`
	IMDbID        string  `json:"imdb_id"`
	Genres        []Genre `json:"genres"`
}

// TVResult represents a TV show search result
type TVResult struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	OriginalName string  `json:"original_name"`
	Overview     string  `json:"overview"`
	FirstAirDate string  `json:"first_air_date"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
	VoteAverage  float64 `json:"vote_average"`
	Popularity   float64 `json:"popularity"`
	GenreIDs     []int   `json:"genre_ids"`
}

// TVSearchResult is an alias for TVResult for consistency
type TVSearchResult = TVResult

// TVDetails represents detailed TV show info
type TVDetails struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	OriginalName    string   `json:"original_name"`
	Overview        string   `json:"overview"`
	FirstAirDate    string   `json:"first_air_date"`
	PosterPath      string   `json:"poster_path"`
	BackdropPath    string   `json:"backdrop_path"`
	VoteAverage     float64  `json:"vote_average"`
	NumberOfSeasons int      `json:"number_of_seasons"`
	NumberOfEpisodes int     `json:"number_of_episodes"`
	Genres          []Genre  `json:"genres"`
	Status          string   `json:"status"` // Returning Series, Ended, Canceled, etc.
	ExternalIDs     *ExternalIDs `json:"external_ids,omitempty"`
}

// Genre represents a genre
type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ExternalIDs contains external IDs like IMDB
type ExternalIDs struct {
	IMDbID string `json:"imdb_id"`
}

// SeasonDetails represents detailed TV season info
type SeasonDetails struct {
	ID           int              `json:"id"`
	SeasonNumber int              `json:"season_number"`
	Name         string           `json:"name"`
	Overview     string           `json:"overview"`
	PosterPath   string           `json:"poster_path"`
	AirDate      string           `json:"air_date"`
	Episodes     []EpisodeSummary `json:"episodes"`
}

// EpisodeSummary represents an episode in season details
type EpisodeSummary struct {
	ID            int     `json:"id"`
	EpisodeNumber int     `json:"episode_number"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	StillPath     string  `json:"still_path"`
	AirDate       string  `json:"air_date"`
	Runtime       int     `json:"runtime"`
	VoteAverage   float64 `json:"vote_average"`
}

// EpisodeDetails represents detailed TV episode info
type EpisodeDetails struct {
	ID            int     `json:"id"`
	EpisodeNumber int     `json:"episode_number"`
	SeasonNumber  int     `json:"season_number"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	StillPath     string  `json:"still_path"`
	AirDate       string  `json:"air_date"`
	Runtime       int     `json:"runtime"`
	VoteAverage   float64 `json:"vote_average"`
}

type searchResponse struct {
	Results []json.RawMessage `json:"results"`
}

// SearchMovieWithResults returns all matching movies for manual selection
func (c *Client) SearchMovieWithResults(title string, year int) ([]MovieSearchResult, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("TMDB API key not configured")
	}

	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("query", title)
	if year > 0 {
		params.Set("year", strconv.Itoa(year))
	}

	resp, err := c.httpClient.Get(fmt.Sprintf("%s/search/movie?%s", baseURL, params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: %d", resp.StatusCode)
	}

	var result struct {
		Results []MovieSearchResult `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Results, nil
}

// SearchMovie searches for movies by title and optional year, returning the best match
func (c *Client) SearchMovie(title string, year int) (*MovieResult, error) {
	results, err := c.SearchMovieWithResults(title, year)
	if err != nil {
		return nil, err
	}

	return c.FindBestMovieMatch(results, title, year), nil
}

// GetMovieDetails fetches detailed movie info by TMDB ID
func (c *Client) GetMovieDetails(tmdbID int) (*MovieDetails, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("TMDB API key not configured")
	}

	resp, err := c.httpClient.Get(fmt.Sprintf("%s/movie/%d?api_key=%s", baseURL, tmdbID, c.apiKey))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: %d", resp.StatusCode)
	}

	var details MovieDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, err
	}

	return &details, nil
}

// SearchTVWithResults returns all matching TV shows for manual selection
func (c *Client) SearchTVWithResults(title string, year int) ([]TVSearchResult, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("TMDB API key not configured")
	}

	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("query", title)
	if year > 0 {
		params.Set("first_air_date_year", strconv.Itoa(year))
	}

	resp, err := c.httpClient.Get(fmt.Sprintf("%s/search/tv?%s", baseURL, params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: %d", resp.StatusCode)
	}

	var result struct {
		Results []TVSearchResult `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Results, nil
}

// SearchTV searches for TV shows by title, returning the best match
func (c *Client) SearchTV(title string, year int) (*TVResult, error) {
	results, err := c.SearchTVWithResults(title, year)
	if err != nil {
		return nil, err
	}

	return c.FindBestTVMatch(results, title, year), nil
}

// GetTVDetails fetches detailed TV show info by TMDB ID
func (c *Client) GetTVDetails(tmdbID int) (*TVDetails, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("TMDB API key not configured")
	}

	resp, err := c.httpClient.Get(fmt.Sprintf("%s/tv/%d?api_key=%s&append_to_response=external_ids", baseURL, tmdbID, c.apiKey))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: %d", resp.StatusCode)
	}

	var details TVDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, err
	}

	return &details, nil
}

// GetTVSeasonDetails fetches detailed season info by TMDB show ID and season number
func (c *Client) GetTVSeasonDetails(showID int, seasonNum int) (*SeasonDetails, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("TMDB API key not configured")
	}

	resp, err := c.httpClient.Get(fmt.Sprintf("%s/tv/%d/season/%d?api_key=%s", baseURL, showID, seasonNum, c.apiKey))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: %d", resp.StatusCode)
	}

	var details SeasonDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, err
	}

	return &details, nil
}

// GetTVEpisodeDetails fetches detailed episode info by TMDB show ID, season and episode number
func (c *Client) GetTVEpisodeDetails(showID int, seasonNum int, episodeNum int) (*EpisodeDetails, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("TMDB API key not configured")
	}

	resp, err := c.httpClient.Get(fmt.Sprintf("%s/tv/%d/season/%d/episode/%d?api_key=%s", baseURL, showID, seasonNum, episodeNum, c.apiKey))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: %d", resp.StatusCode)
	}

	var details EpisodeDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, err
	}

	return &details, nil
}

// FindBestMovieMatch scores and ranks search results to find the best match
// Returns nil if no match meets the minimum confidence threshold (50.0)
func (c *Client) FindBestMovieMatch(results []MovieSearchResult, searchTitle string, searchYear int) *MovieSearchResult {
	if len(results) == 0 {
		return nil
	}

	var bestMatch *MovieSearchResult
	bestScore := 0.0

	for i := range results {
		score := calculateMovieMatchScore(&results[i], searchTitle, searchYear)
		if score > bestScore {
			bestScore = score
			bestMatch = &results[i]
		}
	}

	// Only return if score is above threshold (50.0)
	if bestScore >= 50.0 {
		return bestMatch
	}

	return nil
}

// FindBestTVMatch scores and ranks search results to find the best match
// Returns nil if no match meets the minimum confidence threshold (50.0)
func (c *Client) FindBestTVMatch(results []TVSearchResult, searchTitle string, searchYear int) *TVSearchResult {
	if len(results) == 0 {
		return nil
	}

	var bestMatch *TVSearchResult
	bestScore := 0.0

	for i := range results {
		score := calculateTVMatchScore(&results[i], searchTitle, searchYear)
		if score > bestScore {
			bestScore = score
			bestMatch = &results[i]
		}
	}

	// Only return if score is above threshold (50.0)
	if bestScore >= 50.0 {
		return bestMatch
	}

	return nil
}

// calculateMovieMatchScore calculates a confidence score (0-100) for a movie match
// Scoring breakdown:
// - Title similarity: 0-50 points (exact match = 50, contains = 40, partial = 25)
// - Year proximity: 0-30 points (exact = 30, ±1 year = 20, ±2 years = 10, ±3-5 years = 5)
// - Popularity boost: 0-20 points (higher popularity results get slight preference)
func calculateMovieMatchScore(result *MovieSearchResult, searchTitle string, searchYear int) float64 {
	score := 0.0

	// Title similarity (0-50 points)
	titleScore := titleSimilarity(result.Title, searchTitle)
	score += titleScore * 50.0

	// Year proximity (0-30 points)
	if searchYear > 0 && result.ReleaseDate != "" {
		resultYear := extractYearFromDate(result.ReleaseDate)
		yearDiff := abs(resultYear - searchYear)

		if yearDiff == 0 {
			score += 30.0
		} else if yearDiff == 1 {
			score += 20.0
		} else if yearDiff == 2 {
			score += 10.0
		} else if yearDiff <= 5 {
			score += 5.0
		}
	}

	// Popularity boost (0-20 points)
	// More popular results get slight preference to help disambiguate
	popularityScore := min(result.Popularity/100.0*20.0, 20.0)
	score += popularityScore

	return score
}

// calculateTVMatchScore calculates a confidence score (0-100) for a TV show match
// Uses the same scoring logic as movies but with FirstAirDate instead of ReleaseDate
func calculateTVMatchScore(result *TVSearchResult, searchTitle string, searchYear int) float64 {
	score := 0.0

	// Title similarity (0-50 points)
	titleScore := titleSimilarity(result.Name, searchTitle)
	score += titleScore * 50.0

	// Year proximity (0-30 points)
	if searchYear > 0 && result.FirstAirDate != "" {
		resultYear := extractYearFromDate(result.FirstAirDate)
		yearDiff := abs(resultYear - searchYear)

		if yearDiff == 0 {
			score += 30.0
		} else if yearDiff == 1 {
			score += 20.0
		} else if yearDiff == 2 {
			score += 10.0
		} else if yearDiff <= 5 {
			score += 5.0
		}
	}

	// Popularity boost (0-20 points)
	popularityScore := min(result.Popularity/100.0*20.0, 20.0)
	score += popularityScore

	return score
}

// titleSimilarity compares two titles and returns a similarity score (0.0-1.0)
// 1.0 = exact match after normalization
// 0.8 = one title contains the other
// 0.5 = fallback for any other case (potential partial match)
func titleSimilarity(title1, title2 string) float64 {
	// Normalize titles for comparison
	t1 := normalizeTitle(title1)
	t2 := normalizeTitle(title2)

	// Exact match
	if t1 == t2 {
		return 1.0
	}

	// Contains match (one title is substring of other)
	if strings.Contains(t1, t2) || strings.Contains(t2, t1) {
		return 0.8
	}

	// Fallback - could implement Levenshtein distance or other fuzzy matching
	// For now, use simple approach with moderate score
	return 0.5
}

// normalizeTitle normalizes a title for comparison by:
// - Converting to lowercase
// - Removing common articles (the, a, an)
// - Removing special characters
// - Normalizing whitespace
func normalizeTitle(title string) string {
	// Lowercase
	t := strings.ToLower(title)

	// Remove common articles at start
	t = strings.TrimPrefix(t, "the ")
	t = strings.TrimPrefix(t, "a ")
	t = strings.TrimPrefix(t, "an ")

	// Remove special characters, keep only alphanumeric and spaces
	t = regexp.MustCompile(`[^a-z0-9\s]`).ReplaceAllString(t, "")

	// Normalize whitespace (remove extra spaces)
	t = strings.Join(strings.Fields(t), " ")

	return strings.TrimSpace(t)
}

// extractYearFromDate extracts the year from a date string (expects YYYY-MM-DD format)
func extractYearFromDate(date string) int {
	if len(date) >= 4 {
		year, _ := strconv.Atoi(date[:4])
		return year
	}
	return 0
}

// abs returns the absolute value of an integer
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// GenresToString converts genre slice to comma-separated string
func GenresToString(genres []Genre) string {
	if len(genres) == 0 {
		return ""
	}
	result := ""
	for i, g := range genres {
		if i > 0 {
			result += ", "
		}
		result += g.Name
	}
	return result
}
