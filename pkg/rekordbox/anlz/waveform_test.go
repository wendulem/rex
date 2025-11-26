package anlz

import (
	"path/filepath"
	"testing"
)

func TestWaveformPreviewTag_MarshalUnmarshal(t *testing.T) {
	// Create test waveform data
	tag := &WaveformPreviewTag{}
	for i := 0; i < 400; i++ {
		height := uint8(i % 32)
		whiteness := uint8(i % 8)
		tag.Entries[i] = EncodeWaveformByte(height, whiteness)
	}

	// Marshal
	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Verify size (20 header + 400 data)
	if len(data) != 420 {
		t.Errorf("Expected 420 bytes, got %d", len(data))
	}

	// Verify FourCC
	if string(data[0:4]) != "PWAV" {
		t.Errorf("Wrong FourCC: %q", data[0:4])
	}

	// Unmarshal
	loaded := &WaveformPreviewTag{}
	err = loaded.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	// Verify entries match
	if loaded.Entries != tag.Entries {
		t.Error("Waveform entries mismatch")
	}
}

func TestWaveformPreviewTag_BigEndian(t *testing.T) {
	tag := &WaveformPreviewTag{}

	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Verify header length is big-endian (bytes 4-7)
	if data[4] != 0x00 || data[5] != 0x00 || data[6] != 0x00 || data[7] != 0x14 {
		t.Errorf("HeaderLen not big-endian: %v", data[4:8])
	}

	// Verify tag length is big-endian (bytes 8-11)
	tagLen := ByteOrder.Uint32(data[8:12])
	if tagLen != 420 {
		t.Errorf("TagLen wrong: %d (expected 420)", tagLen)
	}

	// Verify unknown field (bytes 12-15)
	unknown := ByteOrder.Uint32(data[12:16])
	if unknown != 0x00100000 {
		t.Errorf("Unknown field wrong: 0x%08x (expected 0x00100000)", unknown)
	}
}

func TestEncodeDecodeWaveformByte(t *testing.T) {
	tests := []struct {
		height    uint8
		whiteness uint8
	}{
		{0, 0},
		{31, 0},
		{0, 7},
		{31, 7},
		{16, 4},
		{8, 2},
	}

	for _, tt := range tests {
		encoded := EncodeWaveformByte(tt.height, tt.whiteness)
		decodedH, decodedW := DecodeWaveformByte(encoded)

		if decodedH != tt.height {
			t.Errorf("Height mismatch: encoded %d, decoded %d", tt.height, decodedH)
		}

		if decodedW != tt.whiteness {
			t.Errorf("Whiteness mismatch: encoded %d, decoded %d", tt.whiteness, decodedW)
		}
	}
}

func TestGenerateWaveformPreviewFromSamples(t *testing.T) {
	// Create test samples (simulate a simple waveform)
	samples := make([]int16, 1000)
	for i := range samples {
		samples[i] = int16(i * 32) // Gradually increasing amplitude
	}

	tag := GenerateWaveformPreviewFromSamples(samples)

	// Should have 400 entries
	nonZero := 0
	for i := 0; i < 400; i++ {
		if tag.Entries[i] != 0 {
			nonZero++
		}
	}

	// Most entries should be non-zero since we have increasing amplitude
	if nonZero < 300 {
		t.Errorf("Expected mostly non-zero entries, got %d/400", nonZero)
	}
}

func TestGenerateWaveformPreviewFromSamples_SmallInput(t *testing.T) {
	// Test with fewer than 400 samples
	samples := []int16{100, 200, 300, 400, 500}

	tag := GenerateWaveformPreviewFromSamples(samples)

	// First 5 entries should be populated
	for i := 0; i < 5; i++ {
		if tag.Entries[i] == 0 {
			t.Errorf("Entry %d should be non-zero", i)
		}
	}

	// Rest should be zero
	for i := 5; i < 400; i++ {
		if tag.Entries[i] != 0 {
			t.Errorf("Entry %d should be zero, got %d", i, tag.Entries[i])
		}
	}
}

func TestWaveformTinyPreviewTag_MarshalUnmarshal(t *testing.T) {
	// Create test waveform data
	tag := &WaveformTinyPreviewTag{}
	for i := 0; i < 100; i++ {
		height := uint8(i % 16)
		tag.Entries[i] = EncodeTinyWaveformByte(height)
	}

	// Marshal
	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Verify size (20 header + 100 data)
	if len(data) != 120 {
		t.Errorf("Expected 120 bytes, got %d", len(data))
	}

	// Verify FourCC
	if string(data[0:4]) != "PWV2" {
		t.Errorf("Wrong FourCC: %q", data[0:4])
	}

	// Unmarshal
	loaded := &WaveformTinyPreviewTag{}
	err = loaded.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	// Verify entries match
	if loaded.Entries != tag.Entries {
		t.Error("Waveform entries mismatch")
	}
}

func TestEncodeDecodeTinyWaveformByte(t *testing.T) {
	for height := uint8(0); height <= 15; height++ {
		encoded := EncodeTinyWaveformByte(height)
		decoded := DecodeTinyWaveformByte(encoded)

		if decoded != height {
			t.Errorf("Height mismatch: encoded %d, decoded %d", height, decoded)
		}

		// Verify only 4 bits are used
		if encoded > 0x0F {
			t.Errorf("Encoded byte uses more than 4 bits: 0x%02x", encoded)
		}
	}
}

func TestGenerateWaveformTinyPreviewFromSamples(t *testing.T) {
	// Create test samples
	samples := make([]int16, 500)
	for i := range samples {
		samples[i] = int16(i * 64)
	}

	tag := GenerateWaveformTinyPreviewFromSamples(samples)

	// Should have 100 entries
	nonZero := 0
	for i := 0; i < 100; i++ {
		if tag.Entries[i] != 0 {
			nonZero++
		}
	}

	// Most entries should be non-zero
	if nonZero < 80 {
		t.Errorf("Expected mostly non-zero entries, got %d/100", nonZero)
	}

	// Each byte should only use 4 bits
	for i := 0; i < 100; i++ {
		if tag.Entries[i] > 0x0F {
			t.Errorf("Entry %d exceeds 4 bits: 0x%02x", i, tag.Entries[i])
		}
	}
}

func TestWaveformTag_InCompleteFile(t *testing.T) {
	// Create samples for waveform
	samples := make([]int16, 800)
	for i := range samples {
		samples[i] = int16(16000) // Mid amplitude
	}

	// Create a complete file with all implemented tags
	file := &File{
		Header: NewFileHeader(),
		Tags: []Tag{
			&PathTag{Path: "/B/rex/track.mp3"},
			&BeatGridTag{
				Beats: []Beat{
					{BeatNumber: 1, Tempo: 12800, Time: 0},
				},
			},
			&CueListExtendedTag{
				Type: 1,
				Entries: []CueEntry{
					NewHotCue(1, 30000, "Drop", 1),
				},
			},
			GenerateWaveformPreviewFromSamples(samples),
			GenerateWaveformTinyPreviewFromSamples(samples),
		},
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "complete.dat")

	// Write
	err := file.WriteToFile(tmpFile)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	// Read back
	loaded, err := LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify we have 5 tags
	if len(loaded.Tags) != 5 {
		t.Fatalf("Expected 5 tags, got %d", len(loaded.Tags))
	}

	// Verify waveform preview tag
	wavePreview, ok := loaded.Tags[3].(*WaveformPreviewTag)
	if !ok {
		t.Fatalf("Fourth tag is not WaveformPreviewTag")
	}

	// Should have waveform data
	hasData := false
	for i := 0; i < 400; i++ {
		if wavePreview.Entries[i] != 0 {
			hasData = true
			break
		}
	}
	if !hasData {
		t.Error("Waveform preview has no data")
	}

	// Verify tiny waveform tag
	waveTiny, ok := loaded.Tags[4].(*WaveformTinyPreviewTag)
	if !ok {
		t.Fatalf("Fifth tag is not WaveformTinyPreviewTag")
	}

	// Should have waveform data
	hasData = false
	for i := 0; i < 100; i++ {
		if waveTiny.Entries[i] != 0 {
			hasData = true
			break
		}
	}
	if !hasData {
		t.Error("Tiny waveform preview has no data")
	}
}

func TestDownsampleTo400(t *testing.T) {
	// Test with larger input
	samples := make([]int16, 8000)
	for i := range samples {
		samples[i] = int16(i)
	}

	result := downsampleTo400(samples)

	if len(result) != 400 {
		t.Errorf("Expected 400 samples, got %d", len(result))
	}

	// Each result should be the max from its chunk
	// First chunk should have max from samples[0:20]
	if result[0] < 19 {
		t.Errorf("First downsampled value should be ~19, got %d", result[0])
	}
}

func TestDownsampleTo400_SmallInput(t *testing.T) {
	samples := []int16{100, 200, 300}

	result := downsampleTo400(samples)

	// Should return original when smaller
	if len(result) != 3 {
		t.Errorf("Expected 3 samples (unchanged), got %d", len(result))
	}
}

func TestDownsampleTo100(t *testing.T) {
	// Test with larger input
	samples := make([]int16, 5000)
	for i := range samples {
		samples[i] = int16(i)
	}

	result := downsampleTo100(samples)

	if len(result) != 100 {
		t.Errorf("Expected 100 samples, got %d", len(result))
	}
}

