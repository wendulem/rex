package audioproc

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// AubioAnalyzer uses the aubio command-line tool for audio analysis
type AubioAnalyzer struct {
	config *Config
}

// NewAubioAnalyzer creates a new analyzer using aubio
func NewAubioAnalyzer(config *Config) *AubioAnalyzer {
	if config == nil {
		config = DefaultConfig()
	}
	return &AubioAnalyzer{config: config}
}

// Analyze performs complete audio analysis using aubio and FFmpeg
func (a *AubioAnalyzer) Analyze(ctx context.Context, audioPath string) (*Analysis, error) {
	// Detect BPM
	bpm, err := a.DetectBPM(ctx, audioPath)
	if err != nil {
		return nil, fmt.Errorf("detect BPM: %w", err)
	}
	
	// Detect beat positions
	beats, err := a.DetectBeats(ctx, audioPath)
	if err != nil {
		return nil, fmt.Errorf("detect beats: %w", err)
	}
	
	// Get duration (using ffprobe via helper)
	duration, err := getAudioDuration(ctx, audioPath, a.config.FFprobeBin)
	if err != nil {
		return nil, fmt.Errorf("get duration: %w", err)
	}
	
	return &Analysis{
		BPM:      bpm,
		Beats:    beats,
		Duration: duration,
	}, nil
}

// DetectBPM uses aubio to detect the tempo of a track
func (a *AubioAnalyzer) DetectBPM(ctx context.Context, audioPath string) (float64, error) {
	// Create context with timeout
	if a.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}
	
	// Run: aubio tempo <audioPath>
	cmd := exec.CommandContext(ctx, a.config.AubioBin, "tempo", audioPath)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("aubio tempo failed: %w", err)
	}
	
	// Parse output: "128.000000 bpm"
	outputStr := strings.TrimSpace(string(output))
	parts := strings.Fields(outputStr)
	
	if len(parts) < 2 {
		return 0, fmt.Errorf("unexpected aubio output: %q", outputStr)
	}
	
	bpm, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("parse BPM: %w", err)
	}
	
	return bpm, nil
}

// DetectBeats uses aubio to detect precise beat positions
func (a *AubioAnalyzer) DetectBeats(ctx context.Context, audioPath string) ([]BeatPosition, error) {
	// Create context with timeout
	if a.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}
	
	// Run: aubio onset -i <audioPath> -O beats
	cmd := exec.CommandContext(ctx, a.config.AubioBin, "onset", "-i", audioPath, "-O", "beats")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aubio onset failed: %w", err)
	}
	
	// Parse output: one timestamp per line (in seconds)
	var beats []BeatPosition
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		seconds, err := strconv.ParseFloat(line, 64)
		if err != nil {
			continue // Skip unparseable lines
		}
		
		beats = append(beats, BeatPosition{
			Time:       time.Duration(seconds * float64(time.Second)),
			Confidence: 1.0, // aubio doesn't provide confidence scores
		})
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse aubio output: %w", err)
	}
	
	return beats, nil
}

// ExtractWaveform extracts waveform data (delegated to waveform.go)
func (a *AubioAnalyzer) ExtractWaveform(ctx context.Context, audioPath string, numSamples int) ([]int16, error) {
	return extractWaveformFFmpeg(ctx, audioPath, numSamples, a.config.FFmpegBin, a.config.Timeout)
}

// getAudioDuration uses ffprobe to get the track duration
func getAudioDuration(ctx context.Context, audioPath, ffprobeBin string) (time.Duration, error) {
	// Run: ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 <file>
	cmd := exec.CommandContext(ctx, ffprobeBin,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		audioPath,
	)
	
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}
	
	durationStr := strings.TrimSpace(string(output))
	seconds, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("parse duration: %w", err)
	}
	
	return time.Duration(seconds * float64(time.Second)), nil
}

