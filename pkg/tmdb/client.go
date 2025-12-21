package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	GenreIDs     []int   `json:"genre_ids"`
}

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
	GenreIDs     []int   `json:"genre_ids"`
}

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

type searchResponse struct {
	Results []json.RawMessage `json:"results"`
}

// SearchMovie searches for movies by title and optional year
func (c *Client) SearchMovie(title string, year int) (*MovieResult, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("TMDB API key not configured")
	}

	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("query", title)
	if year > 0 {
		params.Set("year", fmt.Sprintf("%d", year))
	}

	resp, err := c.httpClient.Get(fmt.Sprintf("%s/search/movie?%s", baseURL, params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: %d", resp.StatusCode)
	}

	var searchResp searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	if len(searchResp.Results) == 0 {
		return nil, nil // No results found
	}

	var movie MovieResult
	if err := json.Unmarshal(searchResp.Results[0], &movie); err != nil {
		return nil, err
	}

	return &movie, nil
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

// SearchTV searches for TV shows by title
func (c *Client) SearchTV(title string, year int) (*TVResult, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("TMDB API key not configured")
	}

	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("query", title)
	if year > 0 {
		params.Set("first_air_date_year", fmt.Sprintf("%d", year))
	}

	resp, err := c.httpClient.Get(fmt.Sprintf("%s/search/tv?%s", baseURL, params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API error: %d", resp.StatusCode)
	}

	var searchResp searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	if len(searchResp.Results) == 0 {
		return nil, nil
	}

	var tv TVResult
	if err := json.Unmarshal(searchResp.Results[0], &tv); err != nil {
		return nil, err
	}

	return &tv, nil
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
