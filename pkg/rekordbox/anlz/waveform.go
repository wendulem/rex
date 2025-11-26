package anlz

import (
	"fmt"
)

// waveformPreviewTagMarshalBinary encodes the WaveformPreviewTag (PWAV) to bytes
func waveformPreviewTagMarshalBinary(t *WaveformPreviewTag) ([]byte, error) {
	const headerLen = uint32(0x14) // 20 bytes
	const dataLen = uint32(400)    // 400-byte waveform
	tagLen := headerLen + dataLen

	buf := make([]byte, tagLen)

	// Write tag header (12 bytes)
	copy(buf[0:12], MarshalTagHeader(t.FourCC(), headerLen, tagLen))

	// Unknown at bytes 12-15 (usually 0x00100000)
	ByteOrder.PutUint32(buf[12:16], 0x00100000)

	// Waveform data at bytes 16-19 is actually part of the header
	// Unknown field, seems to always be zero
	ByteOrder.PutUint32(buf[16:20], 0)

	// Waveform entries starting at byte 20 (400 bytes)
	copy(buf[20:], t.Entries[:])

	return buf, nil
}

// waveformPreviewTagUnmarshalBinary decodes the WaveformPreviewTag from bytes
func waveformPreviewTagUnmarshalBinary(t *WaveformPreviewTag, data []byte) error {
	// Parse tag header
	header, err := UnmarshalTagHeader(data)
	if err != nil {
		return err
	}

	// Verify FourCC
	if header.FourCC != t.FourCC() {
		return fmt.Errorf("wrong fourcc: %s (expected PWAV)", string(header.FourCC[:]))
	}

	// Verify header length
	if header.HeaderLen != 0x14 {
		return fmt.Errorf("unexpected header length: %d (expected 20)", header.HeaderLen)
	}

	// Verify we have all 400 bytes
	if len(data) < 420 { // 20 header + 400 data
		return fmt.Errorf("data too short for waveform: %d bytes", len(data))
	}

	// Read waveform entries
	copy(t.Entries[:], data[20:420])

	return nil
}

// DecodeWaveformByte decodes a PWAV waveform byte
func DecodeWaveformByte(b byte) (height, whiteness uint8) {
	height = b & 0x1F         // 5 low bits
	whiteness = (b >> 5) & 0x07 // 3 high bits
	return
}

// EncodeWaveformByte encodes height and whiteness into a PWAV byte
func EncodeWaveformByte(height, whiteness uint8) byte {
	return (height & 0x1F) | ((whiteness & 0x07) << 5)
}

// GenerateWaveformPreviewFromSamples creates a PWAV tag from PCM samples
func GenerateWaveformPreviewFromSamples(samples []int16) *WaveformPreviewTag {
	tag := &WaveformPreviewTag{}

	// Downsample to 400 samples if needed
	if len(samples) > 400 {
		samples = downsampleTo400(samples)
	}

	// Convert to PWAV format
	for i := 0; i < 400 && i < len(samples); i++ {
		// Get absolute amplitude
		amp := samples[i]
		if amp < 0 {
			amp = -amp
		}

		// Scale to 0-31 range (5 bits for height)
		height := uint8(float64(amp) / 32768.0 * 31.0)

		// Calculate whiteness based on amplitude (louder = whiter)
		whiteness := uint8(0)
		if height > 24 {
			whiteness = 7
		} else if height > 20 {
			whiteness = 5
		} else if height > 16 {
			whiteness = 3
		} else if height > 12 {
			whiteness = 1
		}

		tag.Entries[i] = EncodeWaveformByte(height, whiteness)
	}

	return tag
}

// downsampleTo400 downsamples samples to exactly 400 values using max amplitude
func downsampleTo400(samples []int16) []int16 {
	if len(samples) <= 400 {
		return samples
	}

	result := make([]int16, 400)
	samplesPerPixel := len(samples) / 400

	for i := 0; i < 400; i++ {
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

		result[i] = maxAmp
	}

	return result
}

// waveformTinyPreviewTagMarshalBinary encodes the WaveformTinyPreviewTag (PWV2) to bytes
func waveformTinyPreviewTagMarshalBinary(t *WaveformTinyPreviewTag) ([]byte, error) {
	const headerLen = uint32(0x14) // 20 bytes
	const dataLen = uint32(100)    // 100-byte waveform
	tagLen := headerLen + dataLen

	buf := make([]byte, tagLen)

	// Write tag header (12 bytes)
	copy(buf[0:12], MarshalTagHeader(t.FourCC(), headerLen, tagLen))

	// Unknown at bytes 12-15 (usually 0x00100000)
	ByteOrder.PutUint32(buf[12:16], 0x00100000)

	// Unknown field at bytes 16-19
	ByteOrder.PutUint32(buf[16:20], 0)

	// Waveform entries starting at byte 20 (100 bytes)
	copy(buf[20:], t.Entries[:])

	return buf, nil
}

// waveformTinyPreviewTagUnmarshalBinary decodes the WaveformTinyPreviewTag from bytes
func waveformTinyPreviewTagUnmarshalBinary(t *WaveformTinyPreviewTag, data []byte) error {
	// Parse tag header
	header, err := UnmarshalTagHeader(data)
	if err != nil {
		return err
	}

	// Verify FourCC
	if header.FourCC != t.FourCC() {
		return fmt.Errorf("wrong fourcc: %s (expected PWV2)", string(header.FourCC[:]))
	}

	// Verify header length
	if header.HeaderLen != 0x14 {
		return fmt.Errorf("unexpected header length: %d (expected 20)", header.HeaderLen)
	}

	// Verify we have all 100 bytes
	if len(data) < 120 { // 20 header + 100 data
		return fmt.Errorf("data too short for waveform: %d bytes", len(data))
	}

	// Read waveform entries
	copy(t.Entries[:], data[20:120])

	return nil
}

// DecodeTinyWaveformByte decodes a PWV2 waveform byte
func DecodeTinyWaveformByte(b byte) uint8 {
	return b & 0x0F // 4 low bits for height (0-15)
}

// EncodeTinyWaveformByte encodes height into a PWV2 byte
func EncodeTinyWaveformByte(height uint8) byte {
	return height & 0x0F
}

// GenerateWaveformTinyPreviewFromSamples creates a PWV2 tag from PCM samples
func GenerateWaveformTinyPreviewFromSamples(samples []int16) *WaveformTinyPreviewTag {
	tag := &WaveformTinyPreviewTag{}

	// Downsample to 100 samples if needed
	if len(samples) > 100 {
		samples = downsampleTo100(samples)
	}

	// Convert to PWV2 format
	for i := 0; i < 100 && i < len(samples); i++ {
		// Get absolute amplitude
		amp := samples[i]
		if amp < 0 {
			amp = -amp
		}

		// Scale to 0-15 range (4 bits for height)
		height := uint8(float64(amp) / 32768.0 * 15.0)

		tag.Entries[i] = EncodeTinyWaveformByte(height)
	}

	return tag
}

// downsampleTo100 downsamples samples to exactly 100 values
func downsampleTo100(samples []int16) []int16 {
	if len(samples) <= 100 {
		return samples
	}

	result := make([]int16, 100)
	samplesPerPixel := len(samples) / 100

	for i := 0; i < 100; i++ {
		startIdx := i * samplesPerPixel
		endIdx := startIdx + samplesPerPixel
		if endIdx > len(samples) {
			endIdx = len(samples)
		}

		// Find maximum absolute amplitude
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

		result[i] = maxAmp
	}

	return result
}


