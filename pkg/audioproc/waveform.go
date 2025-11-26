package audioproc

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os/exec"
	"time"
)

// extractWaveformFFmpeg uses FFmpeg to extract waveform amplitude data
func extractWaveformFFmpeg(ctx context.Context, audioPath string, numSamples int, ffmpegBin string, timeout time.Duration) ([]int16, error) {
	// Create context with timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	
	// Calculate target sample rate to get desired number of samples
	// We'll resample the audio to achieve approximately numSamples
	// For a typical track, we want numSamples spread across the duration
	
	// Run FFmpeg to extract raw PCM data
	// -i <input> : input file
	// -ac 1 : convert to mono
	// -ar <rate> : resample to target rate
	// -f s16le : output 16-bit PCM little-endian
	// pipe:1 : output to stdout
	
	cmd := exec.CommandContext(ctx, ffmpegBin,
		"-i", audioPath,
		"-ac", "1",                    // Mono
		"-ar", fmt.Sprintf("%d", numSamples), // Resample to get numSamples (approximately)
		"-f", "s16le",                 // 16-bit PCM
		"pipe:1",                      // Output to stdout
	)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w", err)
	}
	
	// Convert byte stream to int16 samples
	samples := make([]int16, len(output)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(output[i*2 : i*2+2]))
	}
	
	// If we got more or fewer samples than requested, resample to exact count
	if len(samples) != numSamples {
		samples = resampleToCount(samples, numSamples)
	}
	
	return samples, nil
}

// ExtractWaveformPreview extracts exactly 400 samples for PWAV preview tag
func ExtractWaveformPreview(ctx context.Context, audioPath, ffmpegBin string, timeout time.Duration) ([]int16, error) {
	return extractWaveformFFmpeg(ctx, audioPath, 400, ffmpegBin, timeout)
}

// ExtractWaveformDetail extracts waveform at 150 samples/second for detail tags
func ExtractWaveformDetail(ctx context.Context, audioPath, ffmpegBin string, duration time.Duration, timeout time.Duration) ([]int16, error) {
	// 150 samples per second
	numSamples := int(duration.Seconds() * 150)
	return extractWaveformFFmpeg(ctx, audioPath, numSamples, ffmpegBin, timeout)
}

// resampleToCount resamples an array to exactly the target count
func resampleToCount(input []int16, targetCount int) []int16 {
	if len(input) == 0 || targetCount == 0 {
		return []int16{}
	}
	
	if len(input) == targetCount {
		return input
	}
	
	output := make([]int16, targetCount)
	ratio := float64(len(input)) / float64(targetCount)
	
	for i := 0; i < targetCount; i++ {
		// Find the corresponding position in the input
		srcPos := float64(i) * ratio
		srcIdx := int(srcPos)
		
		if srcIdx >= len(input)-1 {
			output[i] = input[len(input)-1]
		} else {
			// Linear interpolation between samples
			frac := srcPos - float64(srcIdx)
			sample1 := float64(input[srcIdx])
			sample2 := float64(input[srcIdx+1])
			output[i] = int16(sample1 + (sample2-sample1)*frac)
		}
	}
	
	return output
}

// DownsampleForPreview downsamples PCM data for waveform preview
// Groups samples into chunks and finds the maximum amplitude in each
func DownsampleForPreview(samples []int16, targetCount int) []int16 {
	if len(samples) == 0 || targetCount == 0 {
		return []int16{}
	}
	
	if len(samples) <= targetCount {
		return samples
	}
	
	output := make([]int16, targetCount)
	samplesPerPixel := len(samples) / targetCount
	
	for i := 0; i < targetCount; i++ {
		startIdx := i * samplesPerPixel
		endIdx := startIdx + samplesPerPixel
		if endIdx > len(samples) {
			endIdx = len(samples)
		}
		
		// Find maximum absolute amplitude in this range
		var maxAmp int16
		for j := startIdx; j < endIdx; j++ {
			amp := samples[j]
			if amp < 0 {
				amp = -amp
			}
			if amp > maxAmp {
				maxAmp = amp
			}
		}
		
		output[i] = maxAmp
	}
	
	return output
}

// ConvertToPWAVFormat converts PCM samples to PWAV waveform format
// PWAV format: 5 bits height (0-31), 3 bits whiteness (0-7)
func ConvertToPWAVFormat(samples []int16) []byte {
	output := make([]byte, len(samples))
	
	for i, sample := range samples {
		// Convert to absolute value
		amp := sample
		if amp < 0 {
			amp = -amp
		}
		
		// Scale to 0-31 range (5 bits for height)
		height := uint8(float64(amp) / 32768.0 * 31.0)
		
		// Calculate whiteness based on amplitude (higher = whiter)
		// For now, simple mapping: louder = whiter
		whiteness := uint8(0)
		if height > 24 {
			whiteness = 7 // Very loud = brightest
		} else if height > 20 {
			whiteness = 5
		} else if height > 16 {
			whiteness = 3
		} else if height > 12 {
			whiteness = 1
		}
		
		// Pack into byte: 5 bits height + 3 bits whiteness
		output[i] = (height & 0x1F) | ((whiteness & 0x07) << 5)
	}
	
	return output
}

// ConvertToPWV5Format converts PCM samples to PWV5 color waveform format
// PWV5 format: 3 bits red, 3 bits green, 3 bits blue, 5 bits height (16 bits total)
func ConvertToPWV5Format(samples []int16) []uint16 {
	output := make([]uint16, len(samples))
	
	for i, sample := range samples {
		// Convert to absolute value
		amp := sample
		if amp < 0 {
			amp = -amp
		}
		
		// Scale to 0-31 range (5 bits for height)
		height := uint16(float64(amp) / 32768.0 * 31.0)
		
		// Simple color mapping based on amplitude
		// Low = blue, mid = green, high = red
		var red, green, blue uint16
		
		if height > 24 {
			// Very loud = red
			red, green, blue = 7, 0, 0
		} else if height > 16 {
			// Loud = yellow (red + green)
			red, green, blue = 7, 7, 0
		} else if height > 8 {
			// Medium = green
			red, green, blue = 0, 7, 0
		} else {
			// Quiet = blue
			red, green, blue = 0, 0, 7
		}
		
		// Pack into 16 bits: RRRgggbbbHHHHH00
		// Bits 15-13: red (3 bits)
		// Bits 12-10: green (3 bits)
		// Bits 9-7: blue (3 bits)
		// Bits 6-2: height (5 bits)
		// Bits 1-0: unused
		output[i] = (red << 13) | (green << 10) | (blue << 7) | (height << 2)
	}
	
	return output
}

// NormalizeAmplitude normalizes audio samples to use full range
func NormalizeAmplitude(samples []int16) []int16 {
	if len(samples) == 0 {
		return samples
	}
	
	// Find maximum amplitude
	var maxAmp float64
	for _, sample := range samples {
		amp := math.Abs(float64(sample))
		if amp > maxAmp {
			maxAmp = amp
		}
	}
	
	if maxAmp == 0 {
		return samples
	}
	
	// Normalize to use full 16-bit range
	ratio := 32767.0 / maxAmp
	output := make([]int16, len(samples))
	
	for i, sample := range samples {
		output[i] = int16(float64(sample) * ratio)
	}
	
	return output
}

