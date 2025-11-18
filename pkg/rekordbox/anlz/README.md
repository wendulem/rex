# Package anlz - Pioneer DJ Analysis File Generation

## Overview

The `anlz` package provides functionality for generating Pioneer DJ analysis files (`.DAT`, `.EXT`, `.2EX`) that contain track analysis data used by CDJ hardware and rekordbox software.

## Purpose

Analysis files store data that is too large to fit in the `export.pdb` database, including:
- **Beat grids** - Timing and tempo information for beatmatching
- **Cue points** - Hot cues, loops, and memory points
- **Waveforms** - Visual track representations for navigation
- **Song structure** - Phrase analysis for lighting control
- **VBR indices** - Fast seeking in variable bitrate files

Without these files, CDJs can load tracks but lack essential DJ functionality like sync, hot cues, and waveform displays.

## File Types

### ANLZ0000.DAT (Basic Analysis)
**Priority: Critical**

Contains essential data for basic CDJ operation:
- PPTH - Path to audio file
- PQTZ - Beat grid
- PCOB/PCO2 - Cue points and loops
- PWAV - Monochrome waveform preview (400 bytes)
- PWV2 - Tiny waveform preview (100 bytes)
- PVBR - VBR seeking index

### ANLZ0000.EXT (Extended Analysis)
**Priority: Medium**

Contains enhanced waveforms for nexus 2 players:
- PWV3 - Detailed monochrome waveform (variable length)
- PWV4 - Color waveform preview (1,200 entries)
- PWV5 - Detailed color waveform (variable length)

### ANLZ0000.2EX (Extended Analysis 2)
**Priority: Low**

Contains 3-band waveforms for CDJ-3000:
- PWV6 - 3-band waveform preview (1,200 entries)
- PWV7 - Detailed 3-band waveform (variable length)
- PSSI - Song structure for lighting control

## File Structure

All analysis files share the same structure:

```
┌─────────────────────────────────────┐
│ PMAI Header (28 bytes)              │
│  - Magic: "PMAI"                    │
│  - Header length: 0x1c              │
│  - File length: total bytes         │
│  - Unknown padding: 16 bytes        │
├─────────────────────────────────────┤
│ Tagged Section 1                    │
│  ┌─────────────────────────────────┐│
│  │ FourCC (4 bytes) e.g., "PQTZ"   ││
│  │ Header length (4 bytes)         ││
│  │ Tag length (4 bytes)            ││
│  │ Tag-specific data...            ││
│  └─────────────────────────────────┘│
├─────────────────────────────────────┤
│ Tagged Section 2                    │
│  ┌─────────────────────────────────┐│
│  │ FourCC (4 bytes) e.g., "PPTH"   ││
│  │ Header length (4 bytes)         ││
│  │ Tag length (4 bytes)            ││
│  │ Tag-specific data...            ││
│  └─────────────────────────────────┘│
├─────────────────────────────────────┤
│ ... more sections ...               │
└─────────────────────────────────────┘
```

## Key Technical Details

### Byte Order
**CRITICAL:** Analysis files use **BIG-ENDIAN** byte order, which is the **opposite** of the PDB database files (which use little-endian).

```go
// Always use this for ANLZ files
var ByteOrder = binary.BigEndian
```

### String Encoding
Strings in analysis files use UTF-16 Big-Endian with trailing NUL (0000):

```go
// Path and comment strings
encoded := utf16.Encode([]rune(str))
bytes := make([]byte, (len(encoded)+1)*2) // +1 for NUL
for i, r := range encoded {
    binary.BigEndian.PutUint16(bytes[i*2:], r)
}
// bytes[len(bytes)-2:] are 0x00 0x00 (trailing NUL)
```

### Tag Length Calculation
Every tag must accurately report its total length:

```go
headerLen := uint32(0x18)  // Tag header size
dataLen := uint32(len(tagData))
tagLen := headerLen + dataLen  // Total including header

// Write to tag
ByteOrder.PutUint32(buf[4:8], headerLen)
ByteOrder.PutUint32(buf[8:12], tagLen)
```

### File Path Structure
Analysis files are stored in a hashed directory structure:

```
/PIONEER/USBANLZ/{prefix}/{hash}/ANLZ0000.DAT
                    ↓        ↓
                  P016   0000875E
```

- `{prefix}` = First 3 chars of 8-char hex hash
- `{hash}` = Full 8-char hex hash of track path

This path is already generated and stored in the track's `AnalyzePath` field in the PDB database.

## Package Files

### Core Infrastructure
- **`anlz.go`** - Main file I/O, File struct, tag orchestration
- **`header.go`** - PMAI file header structure
- **`tag.go`** - Base Tag interface, common tag functionality

### Tag Implementations
- **`path.go`** - PPTH tag (audio file path)
- **`beatgrid.go`** - PQTZ tag (beat grid with timing/tempo)
- **`cue.go`** - PCOB/PCO2 tags (cue points and loops)
- **`waveform.go`** - All waveform tags (PWAV, PWV2-7)
- **`vbr.go`** - PVBR tag (VBR seeking index)
- **`structure.go`** - PSSI tag (song structure for lighting)

### Testing
- **`anlz_test.go`** - Unit and integration tests

## Usage Example

```go
package main

import (
    "github.com/ambientsound/rex/pkg/rekordbox/anlz"
)

func generateAnalysisFile(track *library.Track) error {
    // Create new file
    file := &anlz.File{
        Header: anlz.NewFileHeader(),
        Tags:   []anlz.Tag{},
    }
    
    // Add path tag
    pathTag := &anlz.PathTag{
        Path: track.OutputPath,
    }
    file.Tags = append(file.Tags, pathTag)
    
    // Add beat grid
    beatGrid := &anlz.BeatGridTag{
        Beats: generateBeats(track),
    }
    file.Tags = append(file.Tags, beatGrid)
    
    // Write to disk
    return file.WriteToFile(track.AnalyzePath)
}
```

## Integration with REX

The analysis file generation is integrated into the main export flow:

```
cmd/rex/main.go
  ├─> Read Mixxx database
  ├─> Copy/transcode audio files
  ├─> Generate PDB database
  └─> Generate analysis files ← NEW
        └─> mediascanner.GenerateAnalysisFiles()
              ├─> Generate beat grid from BPM
              ├─> Extract cues from Mixxx
              ├─> Generate waveforms from audio
              └─> Write .DAT/.EXT/.2EX files
```

## Data Sources

### From Mixxx Database
- Track BPM → Beat grid tempo
- Cue points → Hot cue entries
- Beat positions → Beat grid timing
- Track duration → Waveform length

### From Audio File
- Amplitude analysis → Waveform heights
- RMS values → Waveform whiteness/color
- Frequency bands → 3-band waveforms

### From Track Metadata
- File path → Path tag
- Duration → Beat grid extent
- Hash → Analysis file path

## Implementation Status

### ✅ Planned (Documentation Complete)
- [x] Package structure defined
- [x] All .md documentation files created
- [x] Integration points identified

### 🚧 In Progress
- [ ] Core infrastructure (`anlz.go`, `header.go`, `tag.go`)
- [ ] Path tag (`path.go`)
- [ ] Beat grid tag (`beatgrid.go`)

### ⏳ Not Started
- [ ] Cue tag (`cue.go`)
- [ ] Waveform tags (`waveform.go`)
- [ ] VBR tag (`vbr.go`)
- [ ] Structure tag (`structure.go`)
- [ ] Integration into main export flow

## Testing Strategy

### Unit Tests
Each tag type has marshal/unmarshal tests:
```go
func TestBeatGridTag_Marshal(t *testing.T) {
    tag := &BeatGridTag{ /* ... */ }
    data, err := tag.MarshalBinary()
    // Verify structure, byte order, lengths
}
```

### Integration Tests
```go
func TestCompleteAnalysisFile(t *testing.T) {
    file := createTestFile()
    tmpPath := "/tmp/test.dat"
    
    // Write and read back
    file.WriteToFile(tmpPath)
    loaded, err := anlz.LoadFromFile(tmpPath)
    
    // Verify all tags present and correct
}
```

### Hardware Validation
1. Generate files with REX
2. Copy to USB drive
3. Test on CDJ-2000nxs2
4. Verify:
   - Track loads without errors
   - BPM displays correctly
   - Sync works
   - Hot cues appear
   - Waveform displays

## Reference Documentation

Implementation based on Deep Symmetry's reverse engineering work:
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#analysis-files

Each file's `.md` documentation includes specific byte structure diagrams from this source.

## Performance Considerations

### File Size
- Basic .DAT: ~5-50 KB
- Extended .EXT: ~100-500 KB (due to detailed waveforms)
- Extended .2EX: ~50-200 KB

### Generation Time (Target)
- Path tag: < 1ms
- Beat grid: < 10ms (constant BPM) / < 100ms (from Mixxx analysis)
- Cues: < 5ms (from Mixxx)
- Waveforms: < 500ms (FFmpeg decode) / < 2s (pure Go analysis)

**Target total:** < 2 seconds per track for full analysis

### Memory Usage
- Keep file in memory until complete: ~1 MB max
- Stream audio analysis: don't load entire file
- Reuse buffers where possible

## Error Handling

### Graceful Degradation
If analysis generation fails for a track:
1. Log warning (don't fail entire export)
2. Skip analysis file (track still works, just missing features)
3. Continue with remaining tracks

```go
if err := GenerateAnalysisFiles(track); err != nil {
    log.Printf("Warning: analysis for %q: %v", track.Title, err)
    // Continue - PDB already has track, CDJ can load it
}
```

### Required vs Optional Tags
- **Required:** PPTH (path) - Without this, CDJ can't find audio
- **Critical:** PQTZ (beat grid) - Without this, sync doesn't work
- **Important:** PCO2 (cues) - Without this, hot cues don't work
- **Nice-to-have:** Waveforms - CDJ works but navigation is harder

## Future Enhancements

### Phase 1 (Current)
- Basic .DAT file generation
- Path, beat grid, cue tags
- Integration into export flow

### Phase 2
- Waveform generation
- .EXT file creation
- Color waveforms

### Phase 3
- Song structure analysis
- .2EX file creation
- 3-band waveforms

### Phase 4
- Optimize performance
- Parallel processing
- Caching/incremental generation

## Contributing

When adding new tags or features:

1. Add tag structure to appropriate `.go` file
2. Implement `Tag` interface methods
3. Update corresponding `.md` documentation
4. Add unit tests to `anlz_test.go`
5. Test with real hardware
6. Update this README

## License

Same as REX project license.

