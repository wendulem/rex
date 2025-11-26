package audioproc

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.AubioBin != "aubio" {
		t.Errorf("Expected aubio binary 'aubio', got %q", config.AubioBin)
	}
	
	if config.FFmpegBin != "ffmpeg" {
		t.Errorf("Expected ffmpeg binary 'ffmpeg', got %q", config.FFmpegBin)
	}
	
	if config.FFprobeBin != "ffprobe" {
		t.Errorf("Expected ffprobe binary 'ffprobe', got %q", config.FFprobeBin)
	}
	
	if config.Timeout != 5*time.Minute {
		t.Errorf("Expected timeout 5m, got %v", config.Timeout)
	}
}

func TestNewAubioAnalyzer(t *testing.T) {
	// With config
	config := &Config{
		AubioBin: "/custom/aubio",
		Timeout:  1 * time.Minute,
	}
	
	analyzer := NewAubioAnalyzer(config)
	if analyzer.config.AubioBin != "/custom/aubio" {
		t.Errorf("Config not set correctly")
	}
	
	// Without config (should use defaults)
	analyzer = NewAubioAnalyzer(nil)
	if analyzer.config == nil {
		t.Error("Expected default config, got nil")
	}
	if analyzer.config.AubioBin != "aubio" {
		t.Errorf("Expected default aubio binary, got %q", analyzer.config.AubioBin)
	}
}

func TestResampleToCount(t *testing.T) {
	tests := []struct {
		name        string
		input       []int16
		targetCount int
		expectLen   int
	}{
		{
			name:        "downsample",
			input:       []int16{100, 200, 300, 400, 500, 600},
			targetCount: 3,
			expectLen:   3,
		},
		{
			name:        "upsample",
			input:       []int16{100, 200},
			targetCount: 4,
			expectLen:   4,
		},
		{
			name:        "same size",
			input:       []int16{100, 200, 300},
			targetCount: 3,
			expectLen:   3,
		},
		{
			name:        "empty input",
			input:       []int16{},
			targetCount: 5,
			expectLen:   0,
		},
		{
			name:        "zero target",
			input:       []int16{100, 200},
			targetCount: 0,
			expectLen:   0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resampleToCount(tt.input, tt.targetCount)
			
			if len(result) != tt.expectLen {
				t.Errorf("Expected length %d, got %d", tt.expectLen, len(result))
			}
		})
	}
}

func TestDownsampleForPreview(t *testing.T) {
	// Create test data: 1000 samples
	samples := make([]int16, 1000)
	for i := range samples {
		samples[i] = int16(i % 1000)
	}
	
	// Downsample to 400
	result := DownsampleForPreview(samples, 400)
	
	if len(result) != 400 {
		t.Errorf("Expected 400 samples, got %d", len(result))
	}
	
	// Each sample should be the max from its chunk
	// Verify first sample is max of first ~2-3 samples
	if result[0] < samples[0] {
		t.Errorf("Downsampled value should be >= original")
	}
}

func TestDownsampleForPreview_Empty(t *testing.T) {
	result := DownsampleForPreview([]int16{}, 100)
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d samples", len(result))
	}
}

func TestDownsampleForPreview_SmallerThanTarget(t *testing.T) {
	samples := []int16{100, 200, 300}
	result := DownsampleForPreview(samples, 400)
	
	// Should return original when input is smaller than target
	if len(result) != len(samples) {
		t.Errorf("Expected original length %d, got %d", len(samples), len(result))
	}
}

func TestConvertToPWAVFormat(t *testing.T) {
	samples := []int16{
		0,      // Silent
		8192,   // 25% amplitude
		16384,  // 50% amplitude
		32767,  // 100% amplitude (max)
		-32768, // 100% amplitude (min)
	}
	
	result := ConvertToPWAVFormat(samples)
	
	if len(result) != len(samples) {
		t.Errorf("Expected %d bytes, got %d", len(samples), len(result))
	}
	
	// Verify format: 5 bits height, 3 bits whiteness
	for i, b := range result {
		height := b & 0x1F
		whiteness := (b >> 5) & 0x07
		
		// Height should be in range 0-31
		if height > 31 {
			t.Errorf("Sample %d: height %d exceeds 31", i, height)
		}
		
		// Whiteness should be in range 0-7
		if whiteness > 7 {
			t.Errorf("Sample %d: whiteness %d exceeds 7", i, whiteness)
		}
	}
	
	// First sample (silent) should have height 0
	if (result[0] & 0x1F) != 0 {
		t.Errorf("Silent sample should have height 0, got %d", result[0]&0x1F)
	}
	
	// Last two samples (max amplitude) should have height 31
	maxHeight1 := result[3] & 0x1F
	maxHeight2 := result[4] & 0x1F
	if maxHeight1 != 31 || maxHeight2 != 31 {
		t.Errorf("Max amplitude should have height 31, got %d and %d", maxHeight1, maxHeight2)
	}
}

func TestConvertToPWV5Format(t *testing.T) {
	samples := []int16{
		0,      // Silent
		4096,   // Low
		16384,  // Medium
		28672,  // High
		32767,  // Max
	}
	
	result := ConvertToPWV5Format(samples)
	
	if len(result) != len(samples) {
		t.Errorf("Expected %d values, got %d", len(samples), len(result))
	}
	
	// Verify each sample is properly packed
	for i, val := range result {
		red := (val >> 13) & 0x07
		green := (val >> 10) & 0x07
		blue := (val >> 7) & 0x07
		height := (val >> 2) & 0x1F
		
		// All components should be in valid range
		if red > 7 {
			t.Errorf("Sample %d: red %d exceeds 7", i, red)
		}
		if green > 7 {
			t.Errorf("Sample %d: green %d exceeds 7", i, green)
		}
		if blue > 7 {
			t.Errorf("Sample %d: blue %d exceeds 7", i, blue)
		}
		if height > 31 {
			t.Errorf("Sample %d: height %d exceeds 31", i, height)
		}
		
		// At least one color should be set (except for silent)
		if i > 0 && red == 0 && green == 0 && blue == 0 {
			t.Errorf("Sample %d: no color set", i)
		}
	}
}

func TestNormalizeAmplitude(t *testing.T) {
	// Test with samples that don't use full range
	samples := []int16{100, 200, 300, 400}
	
	result := NormalizeAmplitude(samples)
	
	if len(result) != len(samples) {
		t.Errorf("Expected %d samples, got %d", len(samples), len(result))
	}
	
	// Find max in result - should be close to 32767
	var maxResult int16
	for _, s := range result {
		if s > maxResult {
			maxResult = s
		}
	}
	
	// Should be normalized to use full range
	if maxResult < 30000 {
		t.Errorf("Normalization should maximize amplitude, got %d", maxResult)
	}
	
	// Relative amplitudes should be preserved
	// result[1] should be ~2x result[0]
	ratio := float64(result[1]) / float64(result[0])
	if ratio < 1.9 || ratio > 2.1 {
		t.Errorf("Relative amplitudes not preserved: ratio %f", ratio)
	}
}

func TestNormalizeAmplitude_Empty(t *testing.T) {
	result := NormalizeAmplitude([]int16{})
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d samples", len(result))
	}
}

func TestNormalizeAmplitude_AllZeros(t *testing.T) {
	samples := []int16{0, 0, 0, 0}
	result := NormalizeAmplitude(samples)
	
	// Should return unchanged when all zeros
	for i, s := range result {
		if s != 0 {
			t.Errorf("Sample %d should be 0, got %d", i, s)
		}
	}
}

func TestBeatPosition(t *testing.T) {
	beat := BeatPosition{
		Time:       2500 * time.Millisecond,
		Confidence: 0.95,
	}
	
	if beat.Time != 2500*time.Millisecond {
		t.Errorf("Time mismatch: %v", beat.Time)
	}
	
	if beat.Confidence != 0.95 {
		t.Errorf("Confidence mismatch: %f", beat.Confidence)
	}
}

func TestAnalysis(t *testing.T) {
	analysis := &Analysis{
		BPM: 128.5,
		Beats: []BeatPosition{
			{Time: 0, Confidence: 1.0},
			{Time: 468 * time.Millisecond, Confidence: 1.0},
		},
		WaveformPCM: []int16{100, 200, 300},
		Duration:    3 * time.Minute,
		SampleRate:  44100,
	}
	
	if analysis.BPM != 128.5 {
		t.Errorf("BPM mismatch: %f", analysis.BPM)
	}
	
	if len(analysis.Beats) != 2 {
		t.Errorf("Expected 2 beats, got %d", len(analysis.Beats))
	}
	
	if len(analysis.WaveformPCM) != 3 {
		t.Errorf("Expected 3 waveform samples, got %d", len(analysis.WaveformPCM))
	}
	
	if analysis.Duration != 3*time.Minute {
		t.Errorf("Duration mismatch: %v", analysis.Duration)
	}
	
	if analysis.SampleRate != 44100 {
		t.Errorf("Sample rate mismatch: %d", analysis.SampleRate)
	}
}

// Integration tests (require aubio and ffmpeg to be installed)
// These are skipped by default but can be run with actual audio files

func TestAubioAnalyzer_Integration(t *testing.T) {
	t.Skip("Integration test - requires aubio installation and test audio file")
	
	analyzer := NewAubioAnalyzer(nil)
	ctx := context.Background()
	
	// This would require a real audio file
	audioPath := "testdata/test_track.mp3"
	
	analysis, err := analyzer.Analyze(ctx, audioPath)
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}
	
	if analysis.BPM <= 0 {
		t.Errorf("Invalid BPM: %f", analysis.BPM)
	}
	
	if len(analysis.Beats) == 0 {
		t.Error("No beats detected")
	}
}

