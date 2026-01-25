package library

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/stephencjuliano/media-server/internal/db"
	"github.com/stephencjuliano/media-server/pkg/ffmpeg"
)

// MetadataExtractor handles extraction of technical metadata from media files
type MetadataExtractor struct {
	ffprobe *ffmpeg.FFprobe
}

// NewMetadataExtractor creates a new metadata extractor
func NewMetadataExtractor(ffprobePath string) *MetadataExtractor {
	return &MetadataExtractor{
		ffprobe: ffmpeg.NewFFprobe(ffprobePath),
	}
}

// ExtractFileMetadata extracts technical metadata from a media file
func (m *MetadataExtractor) ExtractFileMetadata(filePath string) (*db.MediaFile, error) {
	// Get file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Get video metadata via ffprobe
	metadata, err := m.ffprobe.GetMetadata(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get ffprobe metadata: %w", err)
	}

	mediaFile := &db.MediaFile{
		FilePath: filePath,
		FileSize: fileInfo.Size(),
	}

	// Extract video metadata
	mediaFile.Duration = metadata.Duration

	if metadata.VideoCodec != "" {
		mediaFile.VideoCodec = metadata.VideoCodec
	}

	if metadata.Resolution != "" {
		mediaFile.Resolution = metadata.Resolution
	}

	if metadata.AudioCodec != "" {
		mediaFile.AudioCodec = metadata.AudioCodec
	}

	// Marshal audio and subtitle tracks to JSON
	if len(metadata.AudioTracks) > 0 {
		mediaFile.AudioTracks = marshalAudioTracks(metadata.AudioTracks)
	}

	if len(metadata.SubtitleTracks) > 0 {
		mediaFile.SubtitleTracks = marshalSubtitleTracks(metadata.SubtitleTracks)
	}

	return mediaFile, nil
}

// marshalAudioTracks converts audio tracks to JSON string
// This matches the format used in pkg/ffmpeg/ffprobe.go lines 171-175
func marshalAudioTracks(tracks []ffmpeg.AudioTrack) string {
	if len(tracks) == 0 {
		return ""
	}

	data, err := json.Marshal(tracks)
	if err != nil {
		return ""
	}

	return string(data)
}

// marshalSubtitleTracks converts subtitle tracks to JSON string
// This matches the format used in pkg/ffmpeg/ffprobe.go lines 176-180
func marshalSubtitleTracks(tracks []ffmpeg.SubtitleTrack) string {
	if len(tracks) == 0 {
		return ""
	}

	data, err := json.Marshal(tracks)
	if err != nil {
		return ""
	}

	return string(data)
}
