package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the media server
type Config struct {
	// Server settings
	Host        string `yaml:"host"`
	Port        string `yaml:"port"`
	Environment string `yaml:"environment"`

	// Database
	DatabasePath string `yaml:"database_path"`

	// JWT settings
	JWTSecret     string `yaml:"jwt_secret"`
	JWTExpiration int    `yaml:"jwt_expiration_hours"`

	// Media sources
	MediaSources []MediaSource `yaml:"media_sources"`

	// Transcoding
	FFmpegPath       string `yaml:"ffmpeg_path"`
	TranscodeDir     string `yaml:"transcode_dir"`
	EnableHWAccel    bool   `yaml:"enable_hw_accel"`
	HWAccelType      string `yaml:"hw_accel_type"` // videotoolbox, nvenc, qsv
	DefaultQuality   string `yaml:"default_quality"`
	ThumbnailSeconds int    `yaml:"thumbnail_seconds"`

	// TMDb API
	TMDbAPIKey string `yaml:"tmdb_api_key"`
}

// MediaSource represents a media storage location
type MediaSource struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Path     string `yaml:"path"`
	Type     string `yaml:"type"` // local, smb, nfs
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".media-server")

	return &Config{
		Host:             "0.0.0.0",
		Port:             "8080",
		Environment:      "development",
		DatabasePath:     filepath.Join(dataDir, "media-server.db"),
		JWTSecret:        "", // Must be set by user
		JWTExpiration:    24 * 7,
		MediaSources:     []MediaSource{},
		FFmpegPath:       "ffmpeg",
		TranscodeDir:     filepath.Join(dataDir, "transcode"),
		EnableHWAccel:    true,
		HWAccelType:      "videotoolbox",
		DefaultQuality:   "1080p",
		ThumbnailSeconds: 30,
		TMDbAPIKey:       "",
	}
}

// Load reads configuration from file or environment
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try to load from config file
	configPaths := []string{
		"config.yaml",
		"config.yml",
		filepath.Join(os.Getenv("HOME"), ".media-server", "config.yaml"),
		"/etc/media-server/config.yaml",
	}

	var configFile string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configFile = path
			break
		}
	}

	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Override with environment variables
	if host := os.Getenv("MEDIA_SERVER_HOST"); host != "" {
		cfg.Host = host
	}
	if port := os.Getenv("MEDIA_SERVER_PORT"); port != "" {
		cfg.Port = port
	}
	if env := os.Getenv("MEDIA_SERVER_ENV"); env != "" {
		cfg.Environment = env
	}
	if dbPath := os.Getenv("MEDIA_SERVER_DB_PATH"); dbPath != "" {
		cfg.DatabasePath = dbPath
	}
	if jwtSecret := os.Getenv("MEDIA_SERVER_JWT_SECRET"); jwtSecret != "" {
		cfg.JWTSecret = jwtSecret
	}
	if tmdbKey := os.Getenv("TMDB_API_KEY"); tmdbKey != "" {
		cfg.TMDbAPIKey = tmdbKey
	}

	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.TranscodeDir, 0755); err != nil {
		return nil, err
	}

	return cfg, nil
}
