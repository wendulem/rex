package mediascanner

import (
	"context"
	"crypto/md5"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ambientsound/rex/pkg/audioproc"
	"github.com/ambientsound/rex/pkg/library"
	"github.com/ambientsound/rex/pkg/rekordbox/anlz"
)

// GenerateAnalysisFile creates a Pioneer DJ analysis file (.DAT) for a track
func GenerateAnalysisFile(ctx context.Context, track *library.Track, basedir string, cueLibrary *library.CueLibrary) error {
	// Initialize audio analyzer
	analyzer := audioproc.NewAubioAnalyzer(nil)
	
	// Analyze the audio file
	analysis, err := analyzer.Analyze(ctx, track.Path)
	if err != nil {
		return fmt.Errorf("analyze audio: %w", err)
	}
	
	// Update track BPM if it was detected
	if analysis.BPM > 0 && track.Tempo == 0 {
		track.Tempo = analysis.BPM
	}
	
	// Create ANLZ file
	file := &anlz.File{
		Header: anlz.NewFileHeader(),
		Tags:   []anlz.Tag{},
	}
	
	// Add path tag (REQUIRED)
	pathTag := createPathTag(track, basedir)
	file.Tags = append(file.Tags, pathTag)
	
	// Add beat grid tag (CRITICAL for sync)
	beatGridTag := createBeatGridTag(analysis, track)
	file.Tags = append(file.Tags, beatGridTag)
	
	// Add cue list (from cueLibrary or empty)
	cueTag := createCueTag(track, cueLibrary)
	file.Tags = append(file.Tags, cueTag)
	
	// Calculate ANLZ file path
	anlzPath := calculateAnalysisPath(track, basedir)
	
	// Write the file
	if err := file.WriteToFile(anlzPath); err != nil {
		return fmt.Errorf("write ANLZ file: %w", err)
	}
	
	return nil
}

// createPathTag creates the PPTH tag with the track's output path
func createPathTag(track *library.Track, basedir string) *anlz.PathTag {
	// Path should be relative to the USB drive root
	relativePath := track.OutputPath
	
	// Remove basedir prefix to get media-relative path
	basedir = strings.TrimRight(basedir, "/")
	if strings.HasPrefix(relativePath, basedir) {
		relativePath = relativePath[len(basedir):]
	}
	
	// Ensure path starts with /
	if !strings.HasPrefix(relativePath, "/") {
		relativePath = "/" + relativePath
	}
	
	return &anlz.PathTag{
		Path: relativePath,
	}
}

// createBeatGridTag creates the PQTZ tag from analyzed beat data
func createBeatGridTag(analysis *audioproc.Analysis, track *library.Track) *anlz.BeatGridTag {
	// Determine BPM (prefer analyzed, fallback to track metadata)
	bpm := analysis.BPM
	if bpm <= 0 {
		bpm = track.Tempo
	}
	if bpm <= 0 {
		bpm = 120.0 // Default BPM
	}
	
	// If we have detected beats, use them
	if len(analysis.Beats) > 0 {
		return anlz.GenerateFromBeats(convertBeatPositions(analysis.Beats), bpm)
	}
	
	// Otherwise, generate constant BPM grid
	duration := analysis.Duration
	if duration == 0 {
		duration = track.Duration
	}
	
	return anlz.GenerateConstantBPMGrid(bpm, duration)
}

// convertBeatPositions converts audioproc.BeatPosition to time.Duration slice
func convertBeatPositions(beats []audioproc.BeatPosition) []time.Duration {
	result := make([]time.Duration, len(beats))
	for i, beat := range beats {
		result[i] = beat.Time
	}
	return result
}

// calculateAnalysisPath generates the path where the ANLZ file should be stored
// Format: /PIONEER/USBANLZ/{prefix}/{hash}/ANLZ0000.DAT
func calculateAnalysisPath(track *library.Track, basedir string) string {
	// Generate hash from the output path
	hash := hashTrackPath(track.OutputPath)
	
	// First 4 characters for prefix directory
	prefix := hash[0:4]
	
	// Construct ANLZ directory path
	anlzDir := filepath.Join(basedir, "PIONEER", "USBANLZ", prefix, hash)
	
	// ANLZ file is always named ANLZ0000.DAT
	return filepath.Join(anlzDir, "ANLZ0000.DAT")
}

// hashTrackPath generates an 8-character hex hash from a track path
// This matches rekordbox's hashing scheme
func hashTrackPath(path string) string {
	// Use MD5 hash (matches rekordbox behavior)
	h := md5.New()
	h.Write([]byte(path))
	hashBytes := h.Sum(nil)
	
	// Convert to hex string and take first 8 characters
	hashStr := fmt.Sprintf("%x", hashBytes)
	if len(hashStr) >= 8 {
		return strings.ToUpper(hashStr[0:8])
	}
	
	// Pad with zeros if somehow we got a short hash
	return strings.ToUpper(hashStr + strings.Repeat("0", 8-len(hashStr)))
}

// GenerateAnalysisPathForTrack generates the AnalyzePath field for the PDB track row
// This is the path that goes in the export.pdb database
func GenerateAnalysisPathForTrack(track *library.Track, basedir string) string {
	hash := hashTrackPath(track.OutputPath)
	prefix := hash[0:4]
	
	// Format: /PIONEER/USBANLZ/{prefix}/{hash}/ANLZ0000.DAT
	return fmt.Sprintf("/PIONEER/USBANLZ/%s/%s/ANLZ0000.DAT", prefix, hash)
}

// GenerateAnalysisFilesForLibrary generates ANLZ files for all tracks in a library
func GenerateAnalysisFilesForLibrary(ctx context.Context, lib *library.Library, basedir string, cueLibrary *library.CueLibrary) error {
	tracks := lib.Tracks().All()
	
	for i, track := range tracks {
		// Log progress
		fmt.Printf("\r[%6d/%6d] Generating analysis for %s", i+1, len(tracks), track.Title)
		
		// Generate analysis file
		err := GenerateAnalysisFile(ctx, track, basedir, cueLibrary)
		if err != nil {
			// Log warning but continue with other tracks
			fmt.Printf("\nWarning: analysis for %q failed: %v\n", track.Title, err)
			continue
		}
	}
	
	fmt.Printf("\r[%6d/%6d] Analysis files generated\n", len(tracks), len(tracks))
	return nil
}

// createCueTag creates a PCO2 cue list tag from the cue library
func createCueTag(track *library.Track, cueLibrary *library.CueLibrary) *anlz.CueListExtendedTag {
	// Get cues for this track
	var cues []library.CuePoint
	if cueLibrary != nil {
		cues = cueLibrary.GetCuesForTrack(track.Path)
	}
	
	// Convert to ANLZ cue entries
	entries := make([]anlz.CueEntry, 0, len(cues))
	
	for _, cue := range cues {
		entry := anlz.CueEntry{
			HotCueNumber:    uint32(cue.Number),
			Time:            uint32(cue.Time.Milliseconds()),
			Comment:         cue.Label,
			LoopNumerator:   0,
			LoopDenominator: 0,
		}
		
		// Set type (cue point or loop)
		if cue.Type == library.CueTypeLoop {
			entry.Type = 2 // Loop
			entry.LoopTime = uint32(cue.LoopEnd.Milliseconds())
		} else {
			entry.Type = 1 // Cue point
			entry.LoopTime = 0
		}
		
		// Set color
		entry.ColorCode = uint8(cue.ColorCode)
		entry.ColorRed, entry.ColorGreen, entry.ColorBlue = getRGBForColorCode(uint8(cue.ColorCode))
		
		entries = append(entries, entry)
	}
	
	// Determine type: hot cues vs memory points
	// If any cue has number > 0, it's hot cues
	tagType := uint32(1) // Default to hot cues
	if len(entries) > 0 && entries[0].HotCueNumber == 0 {
		tagType = 0 // Memory points
	}
	
	return &anlz.CueListExtendedTag{
		Type:    tagType,
		Entries: entries,
	}
}

// getRGBForColorCode converts a color code to RGB values
func getRGBForColorCode(code uint8) (uint8, uint8, uint8) {
	// Default green
	if code == 0 {
		return 0, 255, 0
	}

	// Basic rekordbox palette
	colors := [][3]uint8{
		{255, 0, 0},     // 1: Red
		{255, 128, 0},   // 2: Orange
		{255, 255, 0},   // 3: Yellow
		{0, 255, 0},     // 4: Green
		{0, 255, 255},   // 5: Cyan
		{0, 0, 255},     // 6: Blue
		{255, 0, 255},   // 7: Magenta
		{255, 255, 255}, // 8: White
	}

	if code > 0 && int(code) <= len(colors) {
		return colors[code-1][0], colors[code-1][1], colors[code-1][2]
	}

	// Default to green for unknown codes
	return 0, 255, 0
}

// ValidateAnalysisFile checks if an ANLZ file exists and is valid
func ValidateAnalysisFile(track *library.Track, basedir string) error {
	anlzPath := calculateAnalysisPath(track, basedir)
	
	// Try to load the file
	file, err := anlz.LoadFromFile(anlzPath)
	if err != nil {
		return fmt.Errorf("load ANLZ file: %w", err)
	}
	
	// Verify it has required tags
	hasPath := false
	hasBeatGrid := false
	
	for _, tag := range file.Tags {
		switch tag.TagType() {
		case anlz.TagTypePath:
			hasPath = true
		case anlz.TagTypeBeatGrid:
			hasBeatGrid = true
		}
	}
	
	if !hasPath {
		return fmt.Errorf("missing PPTH (path) tag")
	}
	
	if !hasBeatGrid {
		return fmt.Errorf("missing PQTZ (beat grid) tag")
	}
	
	return nil
}

