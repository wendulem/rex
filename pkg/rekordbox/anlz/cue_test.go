package anlz

import (
	"path/filepath"
	"testing"
)

func TestCueEntry_IsHotCue(t *testing.T) {
	tests := []struct {
		hotCueNumber uint32
		expected     bool
	}{
		{0, false},  // Memory point
		{1, true},   // Hot cue A
		{2, true},   // Hot cue B
		{8, true},   // Hot cue H
	}

	for _, tt := range tests {
		entry := CueEntry{HotCueNumber: tt.hotCueNumber}
		if entry.IsHotCue() != tt.expected {
			t.Errorf("HotCueNumber %d: expected IsHotCue=%v, got %v",
				tt.hotCueNumber, tt.expected, entry.IsHotCue())
		}
	}
}

func TestCueEntry_IsLoop(t *testing.T) {
	tests := []struct {
		cueType  uint8
		expected bool
	}{
		{1, false}, // Cue point
		{2, true},  // Loop
	}

	for _, tt := range tests {
		entry := CueEntry{Type: tt.cueType}
		if entry.IsLoop() != tt.expected {
			t.Errorf("Type %d: expected IsLoop=%v, got %v",
				tt.cueType, tt.expected, entry.IsLoop())
		}
	}
}

func TestNewHotCue(t *testing.T) {
	cue := NewHotCue(1, 30000, "Drop", 1)

	if cue.HotCueNumber != 1 {
		t.Errorf("Expected HotCueNumber 1, got %d", cue.HotCueNumber)
	}

	if cue.Type != 1 {
		t.Errorf("Expected Type 1 (cue point), got %d", cue.Type)
	}

	if cue.Time != 30000 {
		t.Errorf("Expected Time 30000, got %d", cue.Time)
	}

	if cue.LoopTime != 0 {
		t.Errorf("Expected LoopTime 0 for cue point, got %d", cue.LoopTime)
	}

	if cue.Comment != "Drop" {
		t.Errorf("Expected Comment 'Drop', got %q", cue.Comment)
	}

	if cue.ColorCode != 1 {
		t.Errorf("Expected ColorCode 1, got %d", cue.ColorCode)
	}

	// Color code 1 should be red (255, 0, 0)
	if cue.ColorRed != 255 || cue.ColorGreen != 0 || cue.ColorBlue != 0 {
		t.Errorf("Expected red color (255,0,0), got (%d,%d,%d)",
			cue.ColorRed, cue.ColorGreen, cue.ColorBlue)
	}
}

func TestNewHotLoop(t *testing.T) {
	loop := NewHotLoop(2, 60000, 64000, "Loop", 4)

	if loop.HotCueNumber != 2 {
		t.Errorf("Expected HotCueNumber 2, got %d", loop.HotCueNumber)
	}

	if loop.Type != 2 {
		t.Errorf("Expected Type 2 (loop), got %d", loop.Type)
	}

	if loop.Time != 60000 {
		t.Errorf("Expected Time 60000, got %d", loop.Time)
	}

	if loop.LoopTime != 64000 {
		t.Errorf("Expected LoopTime 64000, got %d", loop.LoopTime)
	}

	if loop.Comment != "Loop" {
		t.Errorf("Expected Comment 'Loop', got %q", loop.Comment)
	}
}

func TestColorCodeToRGB(t *testing.T) {
	tests := []struct {
		code        uint8
		expectedR   uint8
		expectedG   uint8
		expectedB   uint8
		description string
	}{
		{0, 0, 255, 0, "Default green"},
		{1, 255, 0, 0, "Red"},
		{2, 255, 128, 0, "Orange"},
		{3, 255, 255, 0, "Yellow"},
		{4, 0, 255, 0, "Green"},
		{5, 0, 255, 255, "Cyan"},
		{6, 0, 0, 255, "Blue"},
		{7, 255, 0, 255, "Magenta"},
		{8, 255, 255, 255, "White"},
		{99, 0, 255, 0, "Unknown defaults to green"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			r, g, b := colorCodeToRGB(tt.code)

			if r != tt.expectedR || g != tt.expectedG || b != tt.expectedB {
				t.Errorf("Color code %d: expected RGB(%d,%d,%d), got RGB(%d,%d,%d)",
					tt.code, tt.expectedR, tt.expectedG, tt.expectedB, r, g, b)
			}
		})
	}
}

func TestCueListExtendedTag_MarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		cueType uint32
		entries []CueEntry
	}{
		{
			name:    "empty hot cue list",
			cueType: 1,
			entries: []CueEntry{},
		},
		{
			name:    "single hot cue",
			cueType: 1,
			entries: []CueEntry{
				NewHotCue(1, 30000, "Intro", 1),
			},
		},
		{
			name:    "multiple hot cues",
			cueType: 1,
			entries: []CueEntry{
				NewHotCue(1, 10000, "Intro", 1),
				NewHotCue(2, 30000, "Drop", 2),
				NewHotCue(3, 60000, "Break", 4),
			},
		},
		{
			name:    "hot cue with loop",
			cueType: 1,
			entries: []CueEntry{
				NewHotCue(1, 5000, "Start", 0),
				NewHotLoop(2, 30000, 34000, "Loop", 5),
			},
		},
		{
			name:    "cue without label",
			cueType: 1,
			entries: []CueEntry{
				{
					HotCueNumber: 1,
					Type:         1,
					Time:         15000,
					ColorCode:    1,
					ColorRed:     255,
					ColorGreen:   0,
					ColorBlue:    0,
				},
			},
		},
		{
			name:    "memory points",
			cueType: 0,
			entries: []CueEntry{
				{
					HotCueNumber: 0,
					Type:         1,
					Time:         5000,
					ColorID:      1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := &CueListExtendedTag{
				Type:    tt.cueType,
				Entries: tt.entries,
			}

			// Marshal
			data, err := original.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary failed: %v", err)
			}

			// Verify minimum size (header is 20 bytes)
			if len(data) < 20 {
				t.Errorf("Data too short: %d bytes", len(data))
			}

			// Verify FourCC
			if string(data[0:4]) != "PCO2" {
				t.Errorf("Wrong FourCC: %q", data[0:4])
			}

			// Unmarshal
			loaded := &CueListExtendedTag{}
			err = loaded.UnmarshalBinary(data)
			if err != nil {
				t.Fatalf("UnmarshalBinary failed: %v", err)
			}

			// Verify type
			if loaded.Type != original.Type {
				t.Errorf("Type mismatch: %d != %d", loaded.Type, original.Type)
			}

			// Verify entry count
			if len(loaded.Entries) != len(original.Entries) {
				t.Errorf("Entry count mismatch: %d != %d", len(loaded.Entries), len(original.Entries))
			}

			// Verify each entry
			for i := range original.Entries {
				orig := &original.Entries[i]
				load := &loaded.Entries[i]

				if load.HotCueNumber != orig.HotCueNumber {
					t.Errorf("Entry %d: HotCueNumber %d != %d", i, load.HotCueNumber, orig.HotCueNumber)
				}
				if load.Type != orig.Type {
					t.Errorf("Entry %d: Type %d != %d", i, load.Type, orig.Type)
				}
				if load.Time != orig.Time {
					t.Errorf("Entry %d: Time %d != %d", i, load.Time, orig.Time)
				}
				if load.LoopTime != orig.LoopTime {
					t.Errorf("Entry %d: LoopTime %d != %d", i, load.LoopTime, orig.LoopTime)
				}
				if load.Comment != orig.Comment {
					t.Errorf("Entry %d: Comment %q != %q", i, load.Comment, orig.Comment)
				}
				if load.ColorCode != orig.ColorCode {
					t.Errorf("Entry %d: ColorCode %d != %d", i, load.ColorCode, orig.ColorCode)
				}
			}
		})
	}
}

func TestCueListExtendedTag_BigEndian(t *testing.T) {
	tag := &CueListExtendedTag{
		Type: 1,
		Entries: []CueEntry{
			NewHotCue(1, 30000, "", 0),
		},
	}

	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Verify header length is big-endian (bytes 4-7)
	if data[4] != 0x00 || data[5] != 0x00 || data[6] != 0x00 || data[7] != 0x14 {
		t.Errorf("HeaderLen not big-endian: %v", data[4:8])
	}

	// Verify tag type is big-endian (bytes 12-15)
	tagType := ByteOrder.Uint32(data[12:16])
	if tagType != 1 {
		t.Errorf("Type wrong: %d", tagType)
	}

	// Verify num entries is big-endian (bytes 16-17)
	numEntries := ByteOrder.Uint16(data[16:18])
	if numEntries != 1 {
		t.Errorf("NumEntries wrong: %d", numEntries)
	}
}

func TestCueListExtendedTag_InFile(t *testing.T) {
	// Create a file with path, beat grid, and cues
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
				Type: 1, // Hot cues
				Entries: []CueEntry{
					NewHotCue(1, 30000, "Drop", 1),
					NewHotLoop(2, 60000, 64000, "Loop", 4),
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

	// Verify we have 3 tags
	if len(loaded.Tags) != 3 {
		t.Fatalf("Expected 3 tags, got %d", len(loaded.Tags))
	}

	// Verify cue tag
	cueTag, ok := loaded.Tags[2].(*CueListExtendedTag)
	if !ok {
		t.Fatalf("Third tag is not CueListExtendedTag")
	}

	// Verify cues
	if len(cueTag.Entries) != 2 {
		t.Errorf("Expected 2 cue entries, got %d", len(cueTag.Entries))
	}

	// Verify first cue
	if cueTag.Entries[0].HotCueNumber != 1 {
		t.Errorf("First cue number wrong: %d", cueTag.Entries[0].HotCueNumber)
	}
	if cueTag.Entries[0].Time != 30000 {
		t.Errorf("First cue time wrong: %d", cueTag.Entries[0].Time)
	}
	if cueTag.Entries[0].Comment != "Drop" {
		t.Errorf("First cue comment wrong: %q", cueTag.Entries[0].Comment)
	}

	// Verify second cue (loop)
	if cueTag.Entries[1].Type != 2 {
		t.Errorf("Second cue should be loop (type 2), got %d", cueTag.Entries[1].Type)
	}
	if cueTag.Entries[1].Time != 60000 {
		t.Errorf("Loop start wrong: %d", cueTag.Entries[1].Time)
	}
	if cueTag.Entries[1].LoopTime != 64000 {
		t.Errorf("Loop end wrong: %d", cueTag.Entries[1].LoopTime)
	}
}

func TestCueEntry_UTF16Comment(t *testing.T) {
	tests := []string{
		"",
		"Drop",
		"🎵 Break 🎵",
		"日本語",
		"Very Long Label With Many Characters",
	}

	for _, label := range tests {
		t.Run(label, func(t *testing.T) {
			original := NewHotCue(1, 5000, label, 0)

			tag := &CueListExtendedTag{
				Type:    1,
				Entries: []CueEntry{original},
			}

			// Marshal
			data, err := tag.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary failed: %v", err)
			}

			// Unmarshal
			loaded := &CueListExtendedTag{}
			err = loaded.UnmarshalBinary(data)
			if err != nil {
				t.Fatalf("UnmarshalBinary failed: %v", err)
			}

			if len(loaded.Entries) != 1 {
				t.Fatalf("Expected 1 entry, got %d", len(loaded.Entries))
			}

			if loaded.Entries[0].Comment != label {
				t.Errorf("Comment mismatch: %q != %q", loaded.Entries[0].Comment, label)
			}
		})
	}
}

func TestCueListExtendedTag_MultipleEntries(t *testing.T) {
	tag := &CueListExtendedTag{
		Type: 1,
		Entries: []CueEntry{
			NewHotCue(1, 5000, "A", 1),
			NewHotCue(2, 10000, "B", 2),
			NewHotCue(3, 15000, "C", 3),
			NewHotCue(4, 20000, "D", 4),
			NewHotCue(5, 25000, "E", 5),
			NewHotCue(6, 30000, "F", 6),
			NewHotCue(7, 35000, "G", 7),
			NewHotCue(8, 40000, "H", 8),
		},
	}

	// Marshal
	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Unmarshal
	loaded := &CueListExtendedTag{}
	err = loaded.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	// Verify all 8 cues
	if len(loaded.Entries) != 8 {
		t.Fatalf("Expected 8 entries, got %d", len(loaded.Entries))
	}

	for i := 0; i < 8; i++ {
		if loaded.Entries[i].HotCueNumber != uint32(i+1) {
			t.Errorf("Entry %d: wrong hot cue number %d", i, loaded.Entries[i].HotCueNumber)
		}
	}
}

func TestCueListExtendedTag_MemoryPoints(t *testing.T) {
	tag := &CueListExtendedTag{
		Type: 0, // Memory points
		Entries: []CueEntry{
			{
				HotCueNumber: 0, // Memory point
				Type:         1,
				Time:         5000,
				ColorID:      1,
			},
		},
	}

	// Marshal
	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Verify type is 0 (memory points)
	tagType := ByteOrder.Uint32(data[12:16])
	if tagType != 0 {
		t.Errorf("Expected type 0 (memory points), got %d", tagType)
	}

	// Unmarshal
	loaded := &CueListExtendedTag{}
	err = loaded.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	if loaded.Type != 0 {
		t.Errorf("Type should be 0, got %d", loaded.Type)
	}

	if len(loaded.Entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(loaded.Entries))
	}

	if loaded.Entries[0].HotCueNumber != 0 {
		t.Errorf("Entry should be memory point (0), got %d", loaded.Entries[0].HotCueNumber)
	}
}

func TestCueEntry_LoopQuantization(t *testing.T) {
	// Test quantized loop (e.g., 4-beat loop)
	entry := CueEntry{
		HotCueNumber:    1,
		Type:            2, // Loop
		Time:            30000,
		LoopTime:        34000,
		LoopNumerator:   4, // 4-beat loop
		LoopDenominator: 1,
		ColorCode:       1,
		ColorRed:        255,
		ColorGreen:      0,
		ColorBlue:       0,
	}

	tag := &CueListExtendedTag{
		Type:    1,
		Entries: []CueEntry{entry},
	}

	// Marshal
	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Unmarshal
	loaded := &CueListExtendedTag{}
	err = loaded.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	if len(loaded.Entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(loaded.Entries))
	}

	if loaded.Entries[0].LoopNumerator != 4 {
		t.Errorf("LoopNumerator wrong: %d", loaded.Entries[0].LoopNumerator)
	}

	if loaded.Entries[0].LoopDenominator != 1 {
		t.Errorf("LoopDenominator wrong: %d", loaded.Entries[0].LoopDenominator)
	}
}

func TestCueListExtendedTag_RealWorldScenario(t *testing.T) {
	// Create a realistic set of hot cues
	tag := &CueListExtendedTag{
		Type: 1, // Hot cues
		Entries: []CueEntry{
			NewHotCue(1, 15000, "Intro", 0),     // Default green
			NewHotCue(2, 45000, "Build", 3),     // Yellow
			NewHotCue(3, 75000, "Drop", 1),      // Red
			NewHotLoop(4, 105000, 109000, "Break Loop", 6), // Blue loop
		},
	}

	// Marshal and write to file
	file := &File{
		Header: NewFileHeader(),
		Tags: []Tag{
			&PathTag{Path: "/B/rex/track.mp3"},
			&BeatGridTag{Beats: []Beat{{1, 12800, 0}}},
			tag,
		},
	}

	tmpFile := filepath.Join(t.TempDir(), "test.dat")
	err := file.WriteToFile(tmpFile)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	// Read back and verify
	loaded, err := LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	cueTag := loaded.Tags[2].(*CueListExtendedTag)
	if len(cueTag.Entries) != 4 {
		t.Errorf("Expected 4 cues, got %d", len(cueTag.Entries))
	}
}

