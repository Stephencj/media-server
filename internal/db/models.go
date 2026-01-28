package db

import (
	"time"
)

// MediaFile contains common fields for all playable media files
type MediaFile struct {
	SourceID       int64  `json:"source_id"`
	FilePath       string `json:"file_path"`
	FileSize       int64  `json:"file_size"`
	Duration       int    `json:"duration"`
	VideoCodec     string `json:"video_codec,omitempty"`
	AudioCodec     string `json:"audio_codec,omitempty"`
	Resolution     string `json:"resolution,omitempty"`
	AudioTracks    string `json:"audio_tracks,omitempty"`
	SubtitleTracks string `json:"subtitle_tracks,omitempty"`
}

// TMDBMetadata contains common TMDB metadata fields
type TMDBMetadata struct {
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title,omitempty"`
	Overview      string  `json:"overview,omitempty"`
	PosterPath    string  `json:"poster_path,omitempty"`
	BackdropPath  string  `json:"backdrop_path,omitempty"`
	Rating        float64 `json:"rating,omitempty"`
	Year          int     `json:"year,omitempty"`
	Genres        string  `json:"genres,omitempty"`
	TMDbID        int     `json:"tmdb_id,omitempty"`
	IMDbID        string  `json:"imdb_id,omitempty"`
}

// Timestamps contains common timestamp fields
type Timestamps struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

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
	MediaTypeExtra   MediaType = "extra"
)

// ExtraCategory represents the type of extra content
type ExtraCategory string

const (
	ExtraCategoryCommentary      ExtraCategory = "commentary"
	ExtraCategoryDeletedScene    ExtraCategory = "deleted_scene"
	ExtraCategoryFeaturette      ExtraCategory = "featurette"
	ExtraCategoryInterview       ExtraCategory = "interview"
	ExtraCategoryGagReel         ExtraCategory = "gag_reel"
	ExtraCategoryMusicVideo      ExtraCategory = "music_video"
	ExtraCategoryBehindTheScenes ExtraCategory = "behind_the_scenes"
	ExtraCategoryOther           ExtraCategory = "other"
)

// Media represents a media item (movie or TV show)
type Media struct {
	ID           int64     `json:"id"`
	MediaFile              // Embedded
	TMDBMetadata           // Embedded
	Timestamps             // Embedded
	Type         MediaType `json:"type"`
	Runtime      int       `json:"runtime,omitempty"`
	SeasonCount  int       `json:"season_count,omitempty"`
	EpisodeCount int       `json:"episode_count,omitempty"`
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
	// Computed fields (populated by queries with JOINs, not stored in DB)
	SeasonCount  int `json:"season_count,omitempty"`
	EpisodeCount int `json:"episode_count,omitempty"`

	// Aggregated technical metadata from episodes
	CommonResolution  string `json:"common_resolution,omitempty"`
	CommonVideoCodec  string `json:"common_video_codec,omitempty"`
	CommonAudioCodec  string `json:"common_audio_codec,omitempty"`
	TotalDuration     int    `json:"total_duration,omitempty"`      // Sum of all episodes
	AvgEpisodeLength  int    `json:"avg_episode_length,omitempty"`   // Average episode duration
	MaxResolution     string `json:"max_resolution,omitempty"`       // Highest resolution available
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
	ID            int64   `json:"id"`
	TVShowID      int64   `json:"tv_show_id"`
	SeasonID      int64   `json:"season_id"`
	SeasonNumber  int     `json:"season_number"`
	EpisodeNumber int     `json:"episode_number"`
	Title         string  `json:"title"`
	Overview      string  `json:"overview,omitempty"`
	StillPath     string  `json:"still_path,omitempty"`
	AirDate       string  `json:"air_date,omitempty"`
	Runtime       int     `json:"runtime,omitempty"`
	Rating        float64 `json:"rating,omitempty"`
	MediaFile               // Embedded
	Timestamps              // Embedded
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
	IsPublic    bool      `json:"is_public"`
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

// Extra represents bonus content (commentaries, deleted scenes, featurettes, etc.)
type Extra struct {
	ID            int64         `json:"id"`
	Title         string        `json:"title"`
	Category      ExtraCategory `json:"category"`

	// Parent references (only one should be set)
	MovieID       *int64 `json:"movie_id,omitempty"`
	TVShowID      *int64 `json:"tv_show_id,omitempty"`
	EpisodeID     *int64 `json:"episode_id,omitempty"`
	SeasonNumber  *int   `json:"season_number,omitempty"`
	EpisodeNumber *int   `json:"episode_number,omitempty"`

	MediaFile  // Embedded
	Timestamps // Embedded

	// Populated by joins (not stored in DB)
	ParentTitle string `json:"parent_title,omitempty"`
}

// Section types
const (
	SectionTypeStandard = "standard" // Manual assignment
	SectionTypeSmart    = "smart"    // Rule-based automatic assignment
	SectionTypeFolder   = "folder"   // Source-based assignment
)

// Rule operators
const (
	OperatorEquals      = "equals"
	OperatorContains    = "contains"
	OperatorGreaterThan = "greater_than"
	OperatorLessThan    = "less_than"
	OperatorInRange     = "in_range"
	OperatorRegex       = "regex"
)

// Section represents a library section (Movies, TV Shows, or custom sections)
type Section struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Icon         string    `json:"icon,omitempty"`
	Description  string    `json:"description,omitempty"`
	SectionType  string    `json:"section_type"` // 'standard', 'smart', 'folder'
	DisplayOrder int       `json:"display_order"`
	IsVisible    bool      `json:"is_visible"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Populated when fetching with rules
	Rules []SectionRule `json:"rules,omitempty"`

	// Populated when fetching with counts
	MediaCount int `json:"media_count,omitempty"`
}

// SectionRule defines a rule for smart sections
type SectionRule struct {
	ID        int64     `json:"id"`
	SectionID int64     `json:"section_id"`
	Field     string    `json:"field"`    // 'type', 'genre', 'year', 'resolution', 'rating', etc.
	Operator  string    `json:"operator"` // 'equals', 'contains', 'greater_than', 'less_than', 'in_range', 'regex'
	Value     string    `json:"value"`    // JSON-encoded value
	CreatedAt time.Time `json:"created_at"`
}

// MediaSection links media items to sections (many-to-many)
type MediaSection struct {
	ID        int64     `json:"id"`
	MediaID   int64     `json:"media_id"`
	MediaType MediaType `json:"media_type"`
	SectionID int64     `json:"section_id"`
	AddedAt   time.Time `json:"added_at"`
}

// SectionWithMedia contains a section and its media items
type SectionWithMedia struct {
	Section
	Media []interface{} `json:"media"` // Can be Media, Episode, Extra, or TVShow
}

// Channel source types
const (
	ChannelSourceSection      = "section"
	ChannelSourcePlaylist     = "playlist"
	ChannelSourceShow         = "show"
	ChannelSourceMovie        = "movie"
	ChannelSourceExtraCategory = "extra_category"
)

// Channel represents a virtual "live TV" channel
type Channel struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Populated when fetching with sources
	Sources []ChannelSource `json:"sources,omitempty"`

	// Computed fields (for display)
	TotalDuration int `json:"total_duration,omitempty"` // Total cycle duration in seconds
	ItemCount     int `json:"item_count,omitempty"`     // Total items in schedule
}

// ChannelSourceOptions contains filtering options for TV show sources
type ChannelSourceOptions struct {
	Seasons           []int    `json:"seasons,omitempty"`            // Specific seasons to include (nil = all)
	IncludeCommentary bool     `json:"include_commentary,omitempty"` // Include commentary tracks
	ExtrasCategories  []string `json:"extras_categories,omitempty"`  // Extra categories to include
}

// ChannelSource defines a content source for a channel
type ChannelSource struct {
	ID          int64                 `json:"id"`
	ChannelID   int64                 `json:"channel_id"`
	SourceType  string                `json:"source_type"` // 'section', 'playlist', 'show', 'movie', 'extra_category'
	SourceID    *int64                `json:"source_id,omitempty"`
	SourceValue string                `json:"source_value,omitempty"` // For extra_category: category name
	Weight      int                   `json:"weight"`                 // Higher = more frequent
	Shuffle     bool                  `json:"shuffle"`                // true=randomize items, false=play in order
	Options     *ChannelSourceOptions `json:"options,omitempty"`      // Filtering options for shows

	// Populated for display
	SourceName string `json:"source_name,omitempty"`
}

// ChannelScheduleItem represents a scheduled item in a channel
type ChannelScheduleItem struct {
	ID                int64     `json:"id"`
	ChannelID         int64     `json:"channel_id"`
	MediaID           int64     `json:"media_id"`
	MediaType         MediaType `json:"media_type"`
	ScheduledPosition int       `json:"scheduled_position"`
	CycleNumber       int       `json:"cycle_number"`
	Duration          int       `json:"duration"`        // in seconds
	CumulativeStart   int       `json:"cumulative_start"` // cumulative seconds from cycle start
	Played            bool      `json:"played"`

	// Populated for display
	Title       string `json:"title,omitempty"`
	PosterPath  string `json:"poster_path,omitempty"`
	BackdropPath string `json:"backdrop_path,omitempty"`
}

// ChannelNowPlaying represents what's currently playing on a channel
type ChannelNowPlaying struct {
	Channel     Channel              `json:"channel"`
	NowPlaying  *ChannelScheduleItem `json:"now_playing"`
	Elapsed     int                  `json:"elapsed"`      // seconds into current item
	UpNext      []ChannelScheduleItem `json:"up_next"`     // next few items
	CycleStart  time.Time            `json:"cycle_start"` // when current cycle started
	StreamURL   string               `json:"stream_url,omitempty"`
}
