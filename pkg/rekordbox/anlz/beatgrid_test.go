package anlz

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBeatGridTag_MarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name  string
		beats []Beat
	}{
		{
			name: "single beat",
			beats: []Beat{
				{BeatNumber: 1, Tempo: 12500, Time: 0},
			},
		},
		{
			name: "four beats (one bar)",
			beats: []Beat{
				{BeatNumber: 1, Tempo: 12800, Time: 0},
				{BeatNumber: 2, Tempo: 12800, Time: 468},
				{BeatNumber: 3, Tempo: 12800, Time: 937},
				{BeatNumber: 4, Tempo: 12800, Time: 1406},
			},
		},
		{
			name:  "empty beat grid",
			beats: []Beat{},
		},
		{
			name: "varying tempo",
			beats: []Beat{
				{BeatNumber: 1, Tempo: 12000, Time: 0},
				{BeatNumber: 2, Tempo: 12500, Time: 500},
				{BeatNumber: 3, Tempo: 13000, Time: 980},
				{BeatNumber: 4, Tempo: 13500, Time: 1442},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := &BeatGridTag{
				Beats: tt.beats,
			}

			// Marshal
			data, err := original.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary failed: %v", err)
			}

			// Verify minimum size (header is 24 bytes)
			expectedSize := 24 + (len(tt.beats) * 8)
			if len(data) != expectedSize {
				t.Errorf("Data size mismatch: expected %d, got %d", expectedSize, len(data))
			}

			// Verify FourCC
			if string(data[0:4]) != "PQTZ" {
				t.Errorf("Wrong FourCC: %q", data[0:4])
			}

			// Unmarshal
			loaded := &BeatGridTag{}
			err = loaded.UnmarshalBinary(data)
			if err != nil {
				t.Fatalf("UnmarshalBinary failed: %v", err)
			}

			// Verify beat count
			if len(loaded.Beats) != len(original.Beats) {
				t.Errorf("Beat count mismatch: %d != %d", len(loaded.Beats), len(original.Beats))
			}

			// Verify each beat
			for i := range original.Beats {
				if loaded.Beats[i] != original.Beats[i] {
					t.Errorf("Beat %d mismatch: %+v != %+v", i, loaded.Beats[i], original.Beats[i])
				}
			}
		})
	}
}

func TestBeatGridTag_BigEndian(t *testing.T) {
	tag := &BeatGridTag{
		Beats: []Beat{
			{BeatNumber: 1, Tempo: 12500, Time: 1000},
		},
	}

	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Verify header length is big-endian (bytes 4-7)
	// Should be 0x00000018 (24 decimal)
	if data[4] != 0x00 || data[5] != 0x00 || data[6] != 0x00 || data[7] != 0x18 {
		t.Errorf("HeaderLen not big-endian: %v", data[4:8])
	}

	// Verify tag length is big-endian (bytes 8-11)
	tagLen := ByteOrder.Uint32(data[8:12])
	if tagLen != uint32(len(data)) {
		t.Errorf("TagLen mismatch: %d != %d", tagLen, len(data))
	}

	// Verify Unknown2 is 0x00800000 (bytes 16-19)
	unknown2 := ByteOrder.Uint32(data[16:20])
	if unknown2 != 0x00800000 {
		t.Errorf("Unknown2 wrong: 0x%08x (expected 0x00800000)", unknown2)
	}

	// Verify NumBeats is big-endian (bytes 20-23)
	numBeats := ByteOrder.Uint32(data[20:24])
	if numBeats != 1 {
		t.Errorf("NumBeats wrong: %d (expected 1)", numBeats)
	}

	// Verify beat data is big-endian (starting at byte 24)
	beatNumber := ByteOrder.Uint16(data[24:26])
	if beatNumber != 1 {
		t.Errorf("BeatNumber wrong: %d", beatNumber)
	}

	tempo := ByteOrder.Uint16(data[26:28])
	if tempo != 12500 {
		t.Errorf("Tempo wrong: %d", tempo)
	}

	timeMs := ByteOrder.Uint32(data[28:32])
	if timeMs != 1000 {
		t.Errorf("Time wrong: %d", timeMs)
	}
}

func TestBeatGridTag_BeatStructure(t *testing.T) {
	beat := Beat{
		BeatNumber: 3,
		Tempo:      12800, // 128.00 BPM
		Time:       2000,  // 2 seconds
	}

	tag := &BeatGridTag{Beats: []Beat{beat}}
	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Beat data starts at byte 24
	beatData := data[24:32]

	// Verify beat number (bytes 0-1)
	if beatData[0] != 0x00 || beatData[1] != 0x03 {
		t.Errorf("BeatNumber encoding wrong: %02x %02x", beatData[0], beatData[1])
	}

	// Verify tempo (bytes 2-3)
	// 12800 = 0x3200 in big-endian
	if beatData[2] != 0x32 || beatData[3] != 0x00 {
		t.Errorf("Tempo encoding wrong: %02x %02x", beatData[2], beatData[3])
	}

	// Verify time (bytes 4-7)
	// 2000 = 0x000007D0 in big-endian
	if beatData[4] != 0x00 || beatData[5] != 0x00 || beatData[6] != 0x07 || beatData[7] != 0xD0 {
		t.Errorf("Time encoding wrong: %02x %02x %02x %02x", 
			beatData[4], beatData[5], beatData[6], beatData[7])
	}
}

func TestBeatGridTag_InvalidFourCC(t *testing.T) {
	// Create data with wrong FourCC
	data := make([]byte, 32)
	copy(data[0:4], []byte("XXXX"))
	ByteOrder.PutUint32(data[4:8], 0x18)
	ByteOrder.PutUint32(data[8:12], 32)

	tag := &BeatGridTag{}
	err := tag.UnmarshalBinary(data)
	if err == nil {
		t.Error("Expected error for wrong FourCC, got nil")
	}
}

func TestBeatGridTag_ShortBuffer(t *testing.T) {
	data := make([]byte, 10) // Too short

	tag := &BeatGridTag{}
	err := tag.UnmarshalBinary(data)
	if err == nil {
		t.Error("Expected error for short buffer, got nil")
	}
}

func TestBeatGridTag_InFile(t *testing.T) {
	// Create a file with a beat grid
	file := &File{
		Header: NewFileHeader(),
		Tags: []Tag{
			&BeatGridTag{
				Beats: []Beat{
					{BeatNumber: 1, Tempo: 12500, Time: 0},
					{BeatNumber: 2, Tempo: 12500, Time: 480},
					{BeatNumber: 3, Tempo: 12500, Time: 960},
					{BeatNumber: 4, Tempo: 12500, Time: 1440},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.dat")

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

	// Verify we have one tag
	if len(loaded.Tags) != 1 {
		t.Fatalf("Expected 1 tag, got %d", len(loaded.Tags))
	}

	// Verify it's a beat grid tag
	beatGridTag, ok := loaded.Tags[0].(*BeatGridTag)
	if !ok {
		t.Fatalf("Tag is not a BeatGridTag")
	}

	// Verify beats
	if len(beatGridTag.Beats) != 4 {
		t.Errorf("Expected 4 beats, got %d", len(beatGridTag.Beats))
	}
}

func TestGenerateConstantBPMGrid(t *testing.T) {
	tests := []struct {
		name           string
		bpm            float64
		duration       time.Duration
		expectedBeats  int
		expectedTempo  uint16
		firstBeatTime  uint32
		secondBeatTime uint32
	}{
		{
			name:           "120 BPM, 10 seconds",
			bpm:            120.0,
			duration:       10 * time.Second,
			expectedBeats:  20,  // 120 BPM = 2 beats/sec × 10 sec
			expectedTempo:  12000,
			firstBeatTime:  0,
			secondBeatTime: 500, // 60000ms / 120bpm = 500ms/beat
		},
		{
			name:           "128 BPM, 5 seconds",
			bpm:            128.0,
			duration:       5 * time.Second,
			expectedBeats:  10,  // 128 BPM / 60 * 5
			expectedTempo:  12800,
			firstBeatTime:  0,
			secondBeatTime: 468, // 60000ms / 128bpm ≈ 468ms/beat
		},
		{
			name:           "140 BPM, 3 minutes",
			bpm:            140.0,
			duration:       3 * time.Minute,
			expectedBeats:  420, // 140 BPM × 3 min
			expectedTempo:  14000,
			firstBeatTime:  0,
			secondBeatTime: 428, // 60000ms / 140bpm ≈ 428ms/beat
		},
		{
			name:           "default BPM (negative input)",
			bpm:            -1,
			duration:       2 * time.Second,
			expectedBeats:  4,   // 120 default × 2 sec / 60
			expectedTempo:  12000,
			firstBeatTime:  0,
			secondBeatTime: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := GenerateConstantBPMGrid(tt.bpm, tt.duration)

			if len(tag.Beats) != tt.expectedBeats {
				t.Errorf("Beat count mismatch: expected %d, got %d", tt.expectedBeats, len(tag.Beats))
			}

			if len(tag.Beats) == 0 {
				return
			}

			// Verify first beat
			if tag.Beats[0].BeatNumber != 1 {
				t.Errorf("First beat number: expected 1, got %d", tag.Beats[0].BeatNumber)
			}
			if tag.Beats[0].Tempo != tt.expectedTempo {
				t.Errorf("Tempo mismatch: expected %d, got %d", tt.expectedTempo, tag.Beats[0].Tempo)
			}
			if tag.Beats[0].Time != tt.firstBeatTime {
				t.Errorf("First beat time: expected %d, got %d", tt.firstBeatTime, tag.Beats[0].Time)
			}

			// Verify second beat (if exists)
			if len(tag.Beats) > 1 {
				if tag.Beats[1].BeatNumber != 2 {
					t.Errorf("Second beat number: expected 2, got %d", tag.Beats[1].BeatNumber)
				}
				if tag.Beats[1].Time != tt.secondBeatTime {
					t.Errorf("Second beat time: expected %d, got %d", tt.secondBeatTime, tag.Beats[1].Time)
				}
			}

			// Verify beat numbers cycle 1-4
			for i, beat := range tag.Beats {
				expectedNum := uint16((i % 4) + 1)
				if beat.BeatNumber != expectedNum {
					t.Errorf("Beat %d: expected number %d, got %d", i, expectedNum, beat.BeatNumber)
				}
			}
		})
	}
}

func TestGenerateFromBeats(t *testing.T) {
	beatTimes := []time.Duration{
		0,
		500 * time.Millisecond,
		1000 * time.Millisecond,
		1500 * time.Millisecond,
		2000 * time.Millisecond,
	}

	tag := GenerateFromBeats(beatTimes, 120.0)

	if len(tag.Beats) != 5 {
		t.Fatalf("Expected 5 beats, got %d", len(tag.Beats))
	}

	// Verify each beat
	for i, beat := range tag.Beats {
		expectedNum := uint16((i % 4) + 1)
		if beat.BeatNumber != expectedNum {
			t.Errorf("Beat %d: expected number %d, got %d", i, expectedNum, beat.BeatNumber)
		}

		if beat.Tempo != 12000 {
			t.Errorf("Beat %d: expected tempo 12000, got %d", i, beat.Tempo)
		}

		expectedTime := uint32(beatTimes[i].Milliseconds())
		if beat.Time != expectedTime {
			t.Errorf("Beat %d: expected time %d, got %d", i, expectedTime, beat.Time)
		}
	}
}

func TestGenerateFromBeats_Empty(t *testing.T) {
	tag := GenerateFromBeats([]time.Duration{}, 120.0)

	if len(tag.Beats) != 0 {
		t.Errorf("Expected 0 beats, got %d", len(tag.Beats))
	}
}

func TestGenerateFromBeats_DefaultBPM(t *testing.T) {
	beatTimes := []time.Duration{0, 500 * time.Millisecond}
	
	// Negative BPM should use default
	tag := GenerateFromBeats(beatTimes, -1)

	if len(tag.Beats) != 2 {
		t.Fatalf("Expected 2 beats, got %d", len(tag.Beats))
	}

	// Should use default 120 BPM (12000)
	if tag.Beats[0].Tempo != 12000 {
		t.Errorf("Expected default tempo 12000, got %d", tag.Beats[0].Tempo)
	}
}

func TestBeatGridTag_RealWorldScenario(t *testing.T) {
	// Simulate a 5-minute track at 128 BPM
	duration := 5 * time.Minute
	bpm := 128.0

	tag := GenerateConstantBPMGrid(bpm, duration)

	// 128 BPM × 5 minutes = 640 beats
	expectedBeats := 640
	if len(tag.Beats) != expectedBeats {
		t.Errorf("Expected ~%d beats, got %d", expectedBeats, len(tag.Beats))
	}

	// Verify tempo is consistent
	for i, beat := range tag.Beats {
		if beat.Tempo != 12800 {
			t.Errorf("Beat %d: tempo mismatch %d", i, beat.Tempo)
		}
	}

	// Verify beat spacing is consistent (approximately 468ms at 128 BPM)
	msPerBeat := 60000.0 / bpm
	for i := 1; i < len(tag.Beats); i++ {
		timeDiff := tag.Beats[i].Time - tag.Beats[i-1].Time
		expectedDiff := uint32(msPerBeat)
		
		// Allow 1ms variance due to rounding
		if timeDiff < expectedDiff-1 || timeDiff > expectedDiff+1 {
			t.Errorf("Beat %d: time spacing %d, expected ~%d", i, timeDiff, expectedDiff)
			break
		}
	}
}

func TestBeatGridTag_CompleteFile(t *testing.T) {
	// Create a complete file with path and beat grid
	file := &File{
		Header: NewFileHeader(),
		Tags: []Tag{
			&PathTag{Path: "/B/rex/track.mp3"},
			GenerateConstantBPMGrid(125.0, 3*time.Minute),
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

	// Verify we have 2 tags
	if len(loaded.Tags) != 2 {
		t.Fatalf("Expected 2 tags, got %d", len(loaded.Tags))
	}

	// Verify path tag
	pathTag, ok := loaded.Tags[0].(*PathTag)
	if !ok {
		t.Fatalf("First tag is not PathTag")
	}
	if pathTag.Path != "/B/rex/track.mp3" {
		t.Errorf("Path mismatch: %q", pathTag.Path)
	}

	// Verify beat grid tag
	beatGridTag, ok := loaded.Tags[1].(*BeatGridTag)
	if !ok {
		t.Fatalf("Second tag is not BeatGridTag")
	}

	// Should have 125 BPM × 3 minutes = 375 beats
	if len(beatGridTag.Beats) != 375 {
		t.Errorf("Expected 375 beats, got %d", len(beatGridTag.Beats))
	}
}

