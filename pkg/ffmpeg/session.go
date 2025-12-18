package ffmpeg

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// TranscodeSession represents an active transcoding session
type TranscodeSession struct {
	MediaID    int64
	InputPath  string
	OutputDir  string
	Profile    TranscodeProfile
	StartTime  time.Time
	Cmd        *exec.Cmd
	Cancel     context.CancelFunc
	Done       chan struct{}
	Error      error
	mu         sync.RWMutex
}

// SessionManager manages active transcoding sessions
type SessionManager struct {
	sessions      map[int64]*TranscodeSession
	mu            sync.RWMutex
	ffmpegPath    string
	outputDir     string
	enableHWAccel bool
	hwAccelType   string
}

// NewSessionManager creates a new session manager
func NewSessionManager(ffmpegPath, outputDir string, enableHWAccel bool, hwAccelType string) *SessionManager {
	return &SessionManager{
		sessions:      make(map[int64]*TranscodeSession),
		ffmpegPath:    ffmpegPath,
		outputDir:     outputDir,
		enableHWAccel: enableHWAccel,
		hwAccelType:   hwAccelType,
	}
}

// GetOrStartSession returns an existing session or starts a new one
func (sm *SessionManager) GetOrStartSession(mediaID int64, inputPath string, profile TranscodeProfile) (*TranscodeSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check for existing session
	if session, exists := sm.sessions[mediaID]; exists {
		return session, nil
	}

	// Check if transcode already completed
	outputPath := filepath.Join(sm.outputDir, fmt.Sprintf("%d", mediaID))
	manifestPath := filepath.Join(outputPath, "manifest.m3u8")

	// If manifest exists and has ENDLIST, transcode is complete
	if data, err := os.ReadFile(manifestPath); err == nil {
		if containsEndList(string(data)) {
			return nil, nil // Already complete, no active session needed
		}
	}

	// Start new session
	session, err := sm.startSession(mediaID, inputPath, profile)
	if err != nil {
		return nil, err
	}

	sm.sessions[mediaID] = session
	return session, nil
}

func (sm *SessionManager) startSession(mediaID int64, inputPath string, profile TranscodeProfile) (*TranscodeSession, error) {
	outputPath := filepath.Join(sm.outputDir, fmt.Sprintf("%d", mediaID))

	// Create output directory
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	manifestPath := filepath.Join(outputPath, "manifest.m3u8")
	segmentPath := filepath.Join(outputPath, "segment%d.ts")

	args := []string{}

	// Hardware acceleration
	if sm.enableHWAccel {
		switch sm.hwAccelType {
		case "videotoolbox":
			args = append(args, "-hwaccel", "videotoolbox")
		case "nvenc":
			args = append(args, "-hwaccel", "cuda")
		case "vaapi":
			args = append(args, "-hwaccel", "vaapi", "-hwaccel_output_format", "vaapi")
		case "qsv":
			args = append(args, "-hwaccel", "qsv")
		}
	}

	// Input
	args = append(args, "-i", inputPath)

	// Video encoding
	videoCodec := "libx264"
	scaleFilter := fmt.Sprintf("scale=%d:%d", profile.Width, profile.Height)

	if sm.enableHWAccel {
		switch sm.hwAccelType {
		case "videotoolbox":
			videoCodec = "h264_videotoolbox"
		case "nvenc":
			videoCodec = "h264_nvenc"
		case "vaapi":
			videoCodec = "h264_vaapi"
			scaleFilter = fmt.Sprintf("scale_vaapi=w=%d:h=%d", profile.Width, profile.Height)
		case "qsv":
			videoCodec = "h264_qsv"
		}
	}

	args = append(args,
		"-c:v", videoCodec,
		"-vf", scaleFilter,
		"-b:v", profile.VideoBitrate,
	)

	// Add preset for software encoding
	if !sm.enableHWAccel || sm.hwAccelType == "" {
		args = append(args, "-preset", profile.Preset)
	}

	// Audio encoding
	args = append(args,
		"-c:a", "aac",
		"-b:a", profile.AudioBitrate,
		"-ac", "2",
	)

	// HLS settings for live/progressive output
	args = append(args,
		"-f", "hls",
		"-hls_time", "4",           // 4 second segments for faster start
		"-hls_list_size", "0",       // Keep all segments in playlist
		"-hls_flags", "independent_segments+append_list",
		"-hls_segment_type", "mpegts",
		"-hls_segment_filename", segmentPath,
		"-y", // Overwrite
		manifestPath,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, sm.ffmpegPath, args...)

	// Capture stderr for debugging
	cmd.Stderr = os.Stderr

	session := &TranscodeSession{
		MediaID:   mediaID,
		InputPath: inputPath,
		OutputDir: outputPath,
		Profile:   profile,
		StartTime: time.Now(),
		Cmd:       cmd,
		Cancel:    cancel,
		Done:      make(chan struct{}),
	}

	// Start transcoding in background
	go func() {
		defer close(session.Done)
		defer func() {
			sm.mu.Lock()
			delete(sm.sessions, mediaID)
			sm.mu.Unlock()
		}()

		log.Printf("Starting live transcode for media %d with profile %s", mediaID, profile.Name)

		if err := cmd.Run(); err != nil {
			session.mu.Lock()
			session.Error = err
			session.mu.Unlock()
			log.Printf("Transcode error for media %d: %v", mediaID, err)
			return
		}

		log.Printf("Transcode complete for media %d", mediaID)
	}()

	return session, nil
}

// WaitForSegments waits for initial segments to be available
func (sm *SessionManager) WaitForSegments(mediaID int64, minSegments int, timeout time.Duration) error {
	outputPath := filepath.Join(sm.outputDir, fmt.Sprintf("%d", mediaID))
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		count := 0
		for i := 0; i < minSegments+5; i++ {
			segmentPath := filepath.Join(outputPath, fmt.Sprintf("segment%d.ts", i))
			if _, err := os.Stat(segmentPath); err == nil {
				count++
			}
		}

		if count >= minSegments {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for segments")
}

// GetSession returns an active session if one exists
func (sm *SessionManager) GetSession(mediaID int64) *TranscodeSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[mediaID]
}

// StopSession stops a transcoding session
func (sm *SessionManager) StopSession(mediaID int64) {
	sm.mu.Lock()
	session, exists := sm.sessions[mediaID]
	if exists {
		delete(sm.sessions, mediaID)
	}
	sm.mu.Unlock()

	if session != nil {
		session.Cancel()
	}
}

// StopAllSessions stops all active sessions
func (sm *SessionManager) StopAllSessions() {
	sm.mu.Lock()
	sessions := make([]*TranscodeSession, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	sm.sessions = make(map[int64]*TranscodeSession)
	sm.mu.Unlock()

	for _, s := range sessions {
		s.Cancel()
	}
}

// IsTranscoding checks if a media item is currently being transcoded
func (sm *SessionManager) IsTranscoding(mediaID int64) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	_, exists := sm.sessions[mediaID]
	return exists
}

// GetAvailableSegments returns the count of available segments
func (sm *SessionManager) GetAvailableSegments(mediaID int64) int {
	outputPath := filepath.Join(sm.outputDir, fmt.Sprintf("%d", mediaID))
	count := 0

	for i := 0; i < 10000; i++ {
		segmentPath := filepath.Join(outputPath, fmt.Sprintf("segment%d.ts", i))
		if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
			break
		}
		count++
	}

	return count
}

func containsEndList(manifest string) bool {
	return len(manifest) > 0 &&
		(contains(manifest, "#EXT-X-ENDLIST") || contains(manifest, "#EXT-X-ENDLIST\n"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
