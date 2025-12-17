package ffmpeg

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// FFprobe wraps ffprobe commands
type FFprobe struct {
	path string
}

// Metadata contains video file metadata
type Metadata struct {
	Duration          int    `json:"duration"` // in seconds
	VideoCodec        string `json:"video_codec"`
	AudioCodec        string `json:"audio_codec"`
	Resolution        string `json:"resolution"`
	Width             int    `json:"width"`
	Height            int    `json:"height"`
	Bitrate           int    `json:"bitrate"`
	AudioTracks       []AudioTrack
	SubtitleTracks    []SubtitleTrack
	AudioTracksJSON   string `json:"audio_tracks"`
	SubtitleTracksJSON string `json:"subtitle_tracks"`
}

// AudioTrack represents an audio stream
type AudioTrack struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Codec    string `json:"codec"`
	Channels int    `json:"channels"`
	Title    string `json:"title,omitempty"`
}

// SubtitleTrack represents a subtitle stream
type SubtitleTrack struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Codec    string `json:"codec"`
	Title    string `json:"title,omitempty"`
	Forced   bool   `json:"forced"`
}

// ffprobeOutput represents the JSON output from ffprobe
type ffprobeOutput struct {
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		Index         int    `json:"index"`
		CodecType     string `json:"codec_type"`
		CodecName     string `json:"codec_name"`
		Width         int    `json:"width,omitempty"`
		Height        int    `json:"height,omitempty"`
		Channels      int    `json:"channels,omitempty"`
		Tags          map[string]string `json:"tags,omitempty"`
		Disposition   map[string]int    `json:"disposition,omitempty"`
	} `json:"streams"`
}

// NewFFprobe creates a new FFprobe instance
func NewFFprobe(ffmpegPath string) *FFprobe {
	// Derive ffprobe path from ffmpeg path
	probePath := strings.Replace(ffmpegPath, "ffmpeg", "ffprobe", 1)
	if probePath == ffmpegPath {
		probePath = "ffprobe"
	}
	return &FFprobe{path: probePath}
}

// GetMetadata extracts metadata from a video file
func (f *FFprobe) GetMetadata(filePath string) (*Metadata, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	}

	cmd := exec.Command(f.path, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe error: %w", err)
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	metadata := &Metadata{}

	// Parse duration
	if probe.Format.Duration != "" {
		if duration, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
			metadata.Duration = int(duration)
		}
	}

	// Parse bitrate
	if probe.Format.BitRate != "" {
		if bitrate, err := strconv.Atoi(probe.Format.BitRate); err == nil {
			metadata.Bitrate = bitrate
		}
	}

	// Process streams
	audioIndex := 0
	subtitleIndex := 0

	for _, stream := range probe.Streams {
		switch stream.CodecType {
		case "video":
			if metadata.VideoCodec == "" {
				metadata.VideoCodec = stream.CodecName
				metadata.Width = stream.Width
				metadata.Height = stream.Height
				metadata.Resolution = fmt.Sprintf("%dx%d", stream.Width, stream.Height)
			}

		case "audio":
			track := AudioTrack{
				Index:    audioIndex,
				Codec:    stream.CodecName,
				Channels: stream.Channels,
			}
			if lang, ok := stream.Tags["language"]; ok {
				track.Language = lang
			} else {
				track.Language = "und"
			}
			if title, ok := stream.Tags["title"]; ok {
				track.Title = title
			}
			metadata.AudioTracks = append(metadata.AudioTracks, track)

			if audioIndex == 0 {
				metadata.AudioCodec = stream.CodecName
			}
			audioIndex++

		case "subtitle":
			track := SubtitleTrack{
				Index: subtitleIndex,
				Codec: stream.CodecName,
			}
			if lang, ok := stream.Tags["language"]; ok {
				track.Language = lang
			} else {
				track.Language = "und"
			}
			if title, ok := stream.Tags["title"]; ok {
				track.Title = title
			}
			if forced, ok := stream.Disposition["forced"]; ok && forced == 1 {
				track.Forced = true
			}
			metadata.SubtitleTracks = append(metadata.SubtitleTracks, track)
			subtitleIndex++
		}
	}

	// Convert tracks to JSON strings for storage
	if len(metadata.AudioTracks) > 0 {
		if data, err := json.Marshal(metadata.AudioTracks); err == nil {
			metadata.AudioTracksJSON = string(data)
		}
	}
	if len(metadata.SubtitleTracks) > 0 {
		if data, err := json.Marshal(metadata.SubtitleTracks); err == nil {
			metadata.SubtitleTracksJSON = string(data)
		}
	}

	return metadata, nil
}
