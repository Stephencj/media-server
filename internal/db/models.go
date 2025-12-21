package db

import (
	"time"
)

// User represents a user account
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// MediaType represents the type of media
type MediaType string

const (
	MediaTypeMovie   MediaType = "movie"
	MediaTypeTVShow  MediaType = "tvshow"
	MediaTypeEpisode MediaType = "episode"
)

// Media represents a media item (movie or TV show)
type Media struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	OriginalTitle string  `json:"original_title,omitempty"`
	Type        MediaType `json:"type"`
	Year        int       `json:"year,omitempty"`
	Overview    string    `json:"overview,omitempty"`
	PosterPath  string    `json:"poster_path,omitempty"`
	BackdropPath string   `json:"backdrop_path,omitempty"`
	Rating      float64   `json:"rating,omitempty"`
	Runtime     int       `json:"runtime,omitempty"` // in minutes
	Genres      string    `json:"genres,omitempty"`  // comma-separated

	// External IDs
	TMDbID  int    `json:"tmdb_id,omitempty"`
	IMDbID  string `json:"imdb_id,omitempty"`

	// For TV Shows
	SeasonCount  int `json:"season_count,omitempty"`
	EpisodeCount int `json:"episode_count,omitempty"`

	// File info
	SourceID  int64  `json:"source_id"`
	FilePath  string `json:"file_path"`
	FileSize  int64  `json:"file_size"`

	// Video info
	Duration       int    `json:"duration"` // in seconds
	VideoCodec     string `json:"video_codec,omitempty"`
	AudioCodec     string `json:"audio_codec,omitempty"`
	Resolution     string `json:"resolution,omitempty"`
	AudioTracks    string `json:"audio_tracks,omitempty"`    // JSON array
	SubtitleTracks string `json:"subtitle_tracks,omitempty"` // JSON array

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TVShow represents a TV series (parent of episodes)
type TVShow struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	OriginalTitle string   `json:"original_title,omitempty"`
	Year         int       `json:"year,omitempty"`
	Overview     string    `json:"overview,omitempty"`
	PosterPath   string    `json:"poster_path,omitempty"`
	BackdropPath string    `json:"backdrop_path,omitempty"`
	Rating       float64   `json:"rating,omitempty"`
	Genres       string    `json:"genres,omitempty"`
	TMDbID       int       `json:"tmdb_id,omitempty"`
	IMDbID       string    `json:"imdb_id,omitempty"`
	Status       string    `json:"status,omitempty"` // Returning Series, Ended, etc.
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Season represents a TV season
type Season struct {
	ID           int64     `json:"id"`
	TVShowID     int64     `json:"tv_show_id"`
	SeasonNumber int       `json:"season_number"`
	Name         string    `json:"name,omitempty"`
	Overview     string    `json:"overview,omitempty"`
	PosterPath   string    `json:"poster_path,omitempty"`
	AirDate      string    `json:"air_date,omitempty"`
	EpisodeCount int       `json:"episode_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// Episode represents a TV episode
type Episode struct {
	ID            int64     `json:"id"`
	TVShowID      int64     `json:"tv_show_id"`
	SeasonID      int64     `json:"season_id"`
	SeasonNumber  int       `json:"season_number"`
	EpisodeNumber int       `json:"episode_number"`
	Title         string    `json:"title"`
	Overview      string    `json:"overview,omitempty"`
	StillPath     string    `json:"still_path,omitempty"`
	AirDate       string    `json:"air_date,omitempty"`
	Runtime       int       `json:"runtime,omitempty"`
	Rating        float64   `json:"rating,omitempty"`

	// File info
	SourceID       int64  `json:"source_id"`
	FilePath       string `json:"file_path"`
	FileSize       int64  `json:"file_size"`
	Duration       int    `json:"duration"`
	VideoCodec     string `json:"video_codec,omitempty"`
	AudioCodec     string `json:"audio_codec,omitempty"`
	Resolution     string `json:"resolution,omitempty"`
	AudioTracks    string `json:"audio_tracks,omitempty"`
	SubtitleTracks string `json:"subtitle_tracks,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MediaSource represents a configured media source
type MediaSource struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Type      string    `json:"type"` // local, smb, nfs
	Username  string    `json:"username,omitempty"`
	Password  string    `json:"-"`
	Enabled   bool      `json:"enabled"`
	LastScan  time.Time `json:"last_scan,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WatchProgress represents viewing progress for a user
type WatchProgress struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	MediaID   int64     `json:"media_id"`
	MediaType MediaType `json:"media_type"`
	Position  int       `json:"position"`  // in seconds
	Duration  int       `json:"duration"`  // in seconds
	Completed bool      `json:"completed"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Watchlist represents a user's saved items
type Watchlist struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	MediaID   int64     `json:"media_id"`
	MediaType MediaType `json:"media_type"`
	AddedAt   time.Time `json:"added_at"`
}

// Playlist represents a user-created playlist
type Playlist struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	ItemCount   int       `json:"item_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PlaylistItem represents an item in a playlist
type PlaylistItem struct {
	ID         int64     `json:"id"`
	PlaylistID int64     `json:"playlist_id"`
	MediaID    int64     `json:"media_id"`
	MediaType  MediaType `json:"media_type"`
	Position   int       `json:"position"`
	AddedAt    time.Time `json:"added_at"`
}

// PlaylistItemWithMedia combines PlaylistItem with Media details for display
type PlaylistItemWithMedia struct {
	ID           int64     `json:"id"`
	PlaylistID   int64     `json:"playlist_id"`
	MediaID      int64     `json:"media_id"`
	MediaType    MediaType `json:"media_type"`
	Position     int       `json:"position"`
	AddedAt      time.Time `json:"added_at"`
	Title        string    `json:"title"`
	Year         int       `json:"year,omitempty"`
	PosterPath   string    `json:"poster_path,omitempty"`
	Duration     int       `json:"duration,omitempty"`
	Overview     string    `json:"overview,omitempty"`
	Rating       float64   `json:"rating,omitempty"`
	Resolution   string    `json:"resolution,omitempty"`
}
