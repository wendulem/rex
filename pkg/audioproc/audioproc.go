package audioproc

import (
	"context"
	"time"
)

// Analysis contains all analyzed audio data from a track
type Analysis struct {
	BPM           float64          // Detected beats per minute
	Beats         []BeatPosition   // Precise beat timestamps
	WaveformPCM   []int16          // Raw PCM amplitude samples
	Duration      time.Duration    // Track duration
	SampleRate    int              // Audio sample rate (Hz)
}

// BeatPosition represents a detected beat timestamp
type BeatPosition struct {
	Time       time.Duration // When the beat occurs in the track
	Confidence float64       // Detection confidence (0.0-1.0), if available
}

// Analyzer analyzes audio files to extract BPM, beats, and waveforms
type Analyzer interface {
	// Analyze performs complete audio analysis
	Analyze(ctx context.Context, audioPath string) (*Analysis, error)
	
	// DetectBPM detects the tempo of the track
	DetectBPM(ctx context.Context, audioPath string) (float64, error)
	
	// DetectBeats detects precise beat positions
	DetectBeats(ctx context.Context, audioPath string) ([]BeatPosition, error)
	
	// ExtractWaveform extracts waveform preview data
	ExtractWaveform(ctx context.Context, audioPath string, numSamples int) ([]int16, error)
}

// Config holds configuration for audio analysis
type Config struct {
	// AubioBin is the path to the aubio binary (default: "aubio")
	AubioBin string
	
	// FFmpegBin is the path to the ffmpeg binary (default: "ffmpeg")
	FFmpegBin string
	
	// FFprobeBin is the path to the ffprobe binary (default: "ffprobe")
	FFprobeBin string
	
	// Timeout for analysis operations
	Timeout time.Duration
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		AubioBin:   "aubio",
		FFmpegBin:  "ffmpeg",
		FFprobeBin: "ffprobe",
		Timeout:    5 * time.Minute,
	}
}

