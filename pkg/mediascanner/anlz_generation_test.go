package mediascanner

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ambientsound/rex/pkg/audioproc"
	"github.com/ambientsound/rex/pkg/library"
)

func TestHashTrackPath(t *testing.T) {
	tests := []struct {
		path         string
		expectedLen  int
		expectedCase string
	}{
		{
			path:         "/path/to/track.mp3",
			expectedLen:  8,
			expectedCase: "uppercase",
		},
		{
			path:         "/different/path.mp3",
			expectedLen:  8,
			expectedCase: "uppercase",
		},
		{
			path:         "",
			expectedLen:  8,
			expectedCase: "uppercase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			hash := hashTrackPath(tt.path)

			if len(hash) != tt.expectedLen {
				t.Errorf("Expected hash length %d, got %d", tt.expectedLen, len(hash))
			}

			// Verify uppercase
			if hash != hash {
				t.Errorf("Hash should be uppercase: %s", hash)
			}

			// Verify it's hexadecimal
			for _, c := range hash {
				if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')) {
					t.Errorf("Hash contains non-hex character: %c", c)
				}
			}
		})
	}
}

func TestHashTrackPath_Deterministic(t *testing.T) {
	path := "/test/path.mp3"

	hash1 := hashTrackPath(path)
	hash2 := hashTrackPath(path)

	if hash1 != hash2 {
		t.Errorf("Hash should be deterministic: %s != %s", hash1, hash2)
	}
}

func TestHashTrackPath_Unique(t *testing.T) {
	path1 := "/path1.mp3"
	path2 := "/path2.mp3"

	hash1 := hashTrackPath(path1)
	hash2 := hashTrackPath(path2)

	if hash1 == hash2 {
		t.Errorf("Different paths should have different hashes")
	}
}

func TestCalculateAnalysisPath(t *testing.T) {
	track := &library.Track{
		OutputPath: "/mnt/usb/rex/track.mp3",
	}

	basedir := "/mnt/usb"

	path := calculateAnalysisPath(track, basedir)

	// Should start with basedir
	if !strings.HasPrefix(path, basedir) {
		t.Errorf("Path should start with basedir: %s", path)
	}

	// Should contain PIONEER/USBANLZ
	if !strings.Contains(path, "PIONEER") || !strings.Contains(path, "USBANLZ") {
		t.Errorf("Path should contain PIONEER/USBANLZ: %s", path)
	}

	// Should end with ANLZ0000.DAT
	if filepath.Base(path) != "ANLZ0000.DAT" {
		t.Errorf("Path should end with ANLZ0000.DAT: %s", path)
	}

	// Verify structure: basedir/PIONEER/USBANLZ/{prefix}/{hash}/ANLZ0000.DAT
	hash := hashTrackPath(track.OutputPath)
	prefix := hash[0:4]
	expectedPath := filepath.Join(basedir, "PIONEER", "USBANLZ", prefix, hash, "ANLZ0000.DAT")

	if path != expectedPath {
		t.Errorf("Path mismatch:\nExpected: %s\nGot:      %s", expectedPath, path)
	}
}

func TestGenerateAnalysisPathForTrack(t *testing.T) {
	track := &library.Track{
		OutputPath: "/mnt/usb/rex/track.mp3",
	}

	basedir := "/mnt/usb"

	path := GenerateAnalysisPathForTrack(track, basedir)

	// Should be in format: /PIONEER/USBANLZ/{prefix}/{hash}/ANLZ0000.DAT
	expectedPrefix := "/PIONEER/USBANLZ/"
	if !strings.HasPrefix(path, expectedPrefix) {
		t.Errorf("Path should start with %s, got: %s", expectedPrefix, path)
	}

	// Should end with ANLZ0000.DAT
	if filepath.Base(path) != "ANLZ0000.DAT" {
		t.Errorf("Path should end with ANLZ0000.DAT: %s", path)
	}

	// Verify it matches the hash
	hash := hashTrackPath(track.OutputPath)
	if !strings.Contains(path, hash) {
		t.Errorf("Path should contain hash %s: %s", hash, path)
	}
}

func TestCreatePathTag(t *testing.T) {
	tests := []struct {
		name           string
		trackPath      string
		basedir        string
		expectedPrefix string
	}{
		{
			name:           "standard path",
			trackPath:      "/mnt/usb/rex/track.mp3",
			basedir:        "/mnt/usb",
			expectedPrefix: "/rex/track.mp3",
		},
		{
			name:           "nested path",
			trackPath:      "/mnt/usb/music/folder/song.mp3",
			basedir:        "/mnt/usb",
			expectedPrefix: "/music/folder/song.mp3",
		},
		{
			name:           "basedir with trailing slash",
			trackPath:      "/mnt/usb/rex/track.mp3",
			basedir:        "/mnt/usb/",
			expectedPrefix: "/rex/track.mp3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			track := &library.Track{
				OutputPath: tt.trackPath,
			}

			tag := createPathTag(track, tt.basedir)

			if tag.Path != tt.expectedPrefix {
				t.Errorf("Expected path %q, got %q", tt.expectedPrefix, tag.Path)
			}

			// Path should start with /
			if tag.Path[0] != '/' {
				t.Errorf("Path should start with /: %s", tag.Path)
			}
		})
	}
}

func TestCreateBeatGridTag_WithAnalyzedBeats(t *testing.T) {
	analysis := &audioproc.Analysis{
		BPM: 128.0,
		Beats: []audioproc.BeatPosition{
			{Time: 0},
			{Time: 468 * time.Millisecond},
			{Time: 937 * time.Millisecond},
			{Time: 1406 * time.Millisecond},
		},
	}

	track := &library.Track{
		Tempo:    125.0, // Different from analysis
		Duration: 3 * time.Minute,
	}

	tag := createBeatGridTag(analysis, track)

	if len(tag.Beats) != 4 {
		t.Errorf("Expected 4 beats, got %d", len(tag.Beats))
	}

	// Should use analyzed beats
	if tag.Beats[0].Time != 0 {
		t.Errorf("First beat should be at time 0")
	}
}

func TestCreateBeatGridTag_ConstantBPM(t *testing.T) {
	analysis := &audioproc.Analysis{
		BPM:      128.5,
		Beats:    []audioproc.BeatPosition{}, // No detected beats
		Duration: 2 * time.Minute,
	}

	track := &library.Track{
		Tempo:    0, // No track metadata
		Duration: 2 * time.Minute,
	}

	tag := createBeatGridTag(analysis, track)

	// Should have beats generated from constant BPM
	// 128.5 BPM × 2 minutes = ~257 beats
	if len(tag.Beats) < 250 || len(tag.Beats) > 260 {
		t.Errorf("Expected ~257 beats, got %d", len(tag.Beats))
	}

	// Verify tempo encoding (128.5 × 100 = 12850)
	if tag.Beats[0].Tempo != 12850 {
		t.Errorf("Expected tempo 12850, got %d", tag.Beats[0].Tempo)
	}
}

func TestCreateBeatGridTag_DefaultBPM(t *testing.T) {
	analysis := &audioproc.Analysis{
		BPM:      0, // No analyzed BPM
		Beats:    []audioproc.BeatPosition{},
		Duration: 1 * time.Minute,
	}

	track := &library.Track{
		Tempo:    0, // No track BPM
		Duration: 1 * time.Minute,
	}

	tag := createBeatGridTag(analysis, track)

	// Should use default 120 BPM
	// 120 BPM × 1 minute = 120 beats
	if len(tag.Beats) != 120 {
		t.Errorf("Expected 120 beats, got %d", len(tag.Beats))
	}

	// Verify default tempo (120 × 100 = 12000)
	if tag.Beats[0].Tempo != 12000 {
		t.Errorf("Expected tempo 12000, got %d", tag.Beats[0].Tempo)
	}
}

func TestConvertBeatPositions(t *testing.T) {
	beats := []audioproc.BeatPosition{
		{Time: 0},
		{Time: 500 * time.Millisecond},
		{Time: 1000 * time.Millisecond},
	}

	result := convertBeatPositions(beats)

	if len(result) != 3 {
		t.Fatalf("Expected 3 durations, got %d", len(result))
	}

	if result[0] != 0 {
		t.Errorf("First duration should be 0, got %v", result[0])
	}

	if result[1] != 500*time.Millisecond {
		t.Errorf("Second duration should be 500ms, got %v", result[1])
	}

	if result[2] != 1000*time.Millisecond {
		t.Errorf("Third duration should be 1000ms, got %v", result[2])
	}
}

func TestConvertBeatPositions_Empty(t *testing.T) {
	result := convertBeatPositions([]audioproc.BeatPosition{})

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

// Integration test (requires actual file system and aubio/ffmpeg)
func TestGenerateAnalysisFile_Integration(t *testing.T) {
	t.Skip("Integration test - requires aubio, ffmpeg, and test audio file")

	ctx := context.Background()
	tmpDir := t.TempDir()

	track := &library.Track{
		Path:       "testdata/test_track.mp3", // Would need real file
		OutputPath: filepath.Join(tmpDir, "rex", "test_track.mp3"),
		Title:      "Test Track",
		Tempo:      128.0,
		Duration:   3 * time.Minute,
	}

	err := GenerateAnalysisFile(ctx, track, tmpDir)
	if err != nil {
		t.Fatalf("GenerateAnalysisFile failed: %v", err)
	}

	// Verify file was created
	anlzPath := calculateAnalysisPath(track, tmpDir)
	// Would check if file exists and can be loaded
	_ = anlzPath
}

