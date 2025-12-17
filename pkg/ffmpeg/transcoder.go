package ffmpeg

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// TranscodeProfile defines transcoding settings
type TranscodeProfile struct {
	Name       string
	Width      int
	Height     int
	VideoBitrate string
	AudioBitrate string
	Preset     string
}

// Common transcoding profiles
var Profiles = map[string]TranscodeProfile{
	"1080p": {
		Name:         "1080p",
		Width:        1920,
		Height:       1080,
		VideoBitrate: "8M",
		AudioBitrate: "192k",
		Preset:       "fast",
	},
	"720p": {
		Name:         "720p",
		Width:        1280,
		Height:       720,
		VideoBitrate: "4M",
		AudioBitrate: "128k",
		Preset:       "fast",
	},
	"480p": {
		Name:         "480p",
		Width:        854,
		Height:       480,
		VideoBitrate: "1.5M",
		AudioBitrate: "128k",
		Preset:       "fast",
	},
}

// Transcoder handles video transcoding
type Transcoder struct {
	ffmpegPath    string
	outputDir     string
	enableHWAccel bool
	hwAccelType   string
}

// NewTranscoder creates a new transcoder
func NewTranscoder(ffmpegPath, outputDir string, enableHWAccel bool, hwAccelType string) *Transcoder {
	return &Transcoder{
		ffmpegPath:    ffmpegPath,
		outputDir:     outputDir,
		enableHWAccel: enableHWAccel,
		hwAccelType:   hwAccelType,
	}
}

// TranscodeToHLS transcodes a video to HLS format
func (t *Transcoder) TranscodeToHLS(ctx context.Context, inputPath string, mediaID int64, profile TranscodeProfile) error {
	outputPath := filepath.Join(t.outputDir, fmt.Sprintf("%d", mediaID))

	// Create output directory
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	manifestPath := filepath.Join(outputPath, "manifest.m3u8")
	segmentPath := filepath.Join(outputPath, "segment%d.ts")

	args := []string{}

	// Hardware acceleration
	if t.enableHWAccel {
		switch t.hwAccelType {
		case "videotoolbox":
			args = append(args, "-hwaccel", "videotoolbox")
		case "nvenc":
			args = append(args, "-hwaccel", "cuda")
		case "qsv":
			args = append(args, "-hwaccel", "qsv")
		}
	}

	// Input
	args = append(args, "-i", inputPath)

	// Video encoding
	videoCodec := "libx264"
	if t.enableHWAccel {
		switch t.hwAccelType {
		case "videotoolbox":
			videoCodec = "h264_videotoolbox"
		case "nvenc":
			videoCodec = "h264_nvenc"
		case "qsv":
			videoCodec = "h264_qsv"
		}
	}

	args = append(args,
		"-c:v", videoCodec,
		"-vf", fmt.Sprintf("scale=%d:%d", profile.Width, profile.Height),
		"-b:v", profile.VideoBitrate,
		"-preset", profile.Preset,
	)

	// Audio encoding
	args = append(args,
		"-c:a", "aac",
		"-b:a", profile.AudioBitrate,
		"-ac", "2",
	)

	// HLS settings
	args = append(args,
		"-f", "hls",
		"-hls_time", "10",
		"-hls_list_size", "0",
		"-hls_segment_filename", segmentPath,
		manifestPath,
	)

	cmd := exec.CommandContext(ctx, t.ffmpegPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("Starting transcode for media %d with profile %s", mediaID, profile.Name)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("transcoding failed: %w", err)
	}

	log.Printf("Transcode complete for media %d", mediaID)
	return nil
}

// ExtractSubtitles extracts subtitles from a video file to VTT format
func (t *Transcoder) ExtractSubtitles(inputPath string, mediaID int64, trackIndex int, language string) error {
	outputPath := filepath.Join(t.outputDir, fmt.Sprintf("%d", mediaID))
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return err
	}

	subtitlePath := filepath.Join(outputPath, fmt.Sprintf("subtitle_%s.vtt", language))

	args := []string{
		"-i", inputPath,
		"-map", fmt.Sprintf("0:s:%d", trackIndex),
		"-c:s", "webvtt",
		subtitlePath,
	}

	cmd := exec.Command(t.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("subtitle extraction failed: %w", err)
	}

	return nil
}

// GenerateThumbnail creates a thumbnail image from a video
func (t *Transcoder) GenerateThumbnail(inputPath string, mediaID int64, seekSeconds int) error {
	outputPath := filepath.Join(t.outputDir, fmt.Sprintf("%d", mediaID))
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return err
	}

	thumbnailPath := filepath.Join(outputPath, "thumbnail.jpg")

	args := []string{
		"-ss", fmt.Sprintf("%d", seekSeconds),
		"-i", inputPath,
		"-vframes", "1",
		"-vf", "scale=320:-1",
		"-q:v", "2",
		thumbnailPath,
	}

	cmd := exec.Command(t.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("thumbnail generation failed: %w", err)
	}

	return nil
}

// CanDirectPlay checks if a file can be played directly without transcoding
func CanDirectPlay(metadata *Metadata, targetCodecs []string) bool {
	for _, codec := range targetCodecs {
		if metadata.VideoCodec == codec {
			return true
		}
	}
	return false
}

// DirectPlayCodecs for different platforms
var DirectPlayCodecs = map[string][]string{
	"apple_tv": {"h264", "hevc"},
	"fire_tv":  {"h264", "hevc", "vp9"},
}
