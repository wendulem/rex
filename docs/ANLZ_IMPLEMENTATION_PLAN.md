# Analysis File Implementation Plan

## Overview

This document outlines the implementation plan for adding Pioneer DJ analysis file (.DAT/.EXT/.2EX) generation to REX. These files are essential for CDJ hardware to properly utilize tracks exported from Mixxx.

## Current State

REX currently:
- ✅ Generates `export.pdb` database with track metadata
- ✅ Stores `AnalyzePath` field in track rows pointing to analysis files
- ❌ Does NOT generate the actual analysis files

This means CDJs can see the tracks but lack:
- Beat grids (sync/beatmatching won't work)
- Hot cues and loops
- Waveform displays
- Song structure for lighting

## What Are Analysis Files?

Analysis files are binary files that store track analysis data too large for the database:

| File Type | Extension | Purpose | Priority |
|-----------|-----------|---------|----------|
| Basic Analysis | `.DAT` | Beat grid, cues, basic waveform, path | **Critical** |
| Extended Analysis | `.EXT` | Detailed color waveforms for nxs2 | Medium |
| Extended Analysis 2 | `.2EX` | 3-band waveforms for CDJ-3000 | Low |

### File Structure

All analysis files use the same structure:
```
[PMAI Header - 28 bytes]
[Tagged Section 1]
[Tagged Section 2]
[...]
```

Each tagged section has:
```
[FourCC - 4 bytes]    # e.g., "PQTZ" for beat grid
[Header Length - 4 bytes]
[Tag Length - 4 bytes]
[Tag-specific data]
```

**Critical Difference:** Analysis files use **BIG-ENDIAN** byte order (opposite of PDB files which use little-endian).

## Implementation Architecture

### New Package Structure

```
pkg/rekordbox/anlz/
├── README.md                    # Package overview
├── anlz.go                      # Core file I/O
├── anlz_test.go                 # Integration tests
├── header.go                    # PMAI file header
├── header.md                    # Documentation
├── tag.go                       # Base tag interface
├── tag.md                       # Documentation
├── beatgrid.go                  # PQTZ beat grid tag
├── beatgrid.md                  # Documentation
├── cue.go                       # PCOB/PCO2 cue tags
├── cue.md                       # Documentation
├── path.go                      # PPTH path tag
├── path.md                      # Documentation
├── vbr.go                       # PVBR VBR index tag
├── vbr.md                       # Documentation
├── waveform.go                  # All waveform tags
├── waveform.md                  # Documentation
├── structure.go                 # PSSI song structure tag
└── structure.md                 # Documentation
```

### Integration Points

```
cmd/rex/main.go
└── After track copying (line ~200)
    └── Call mediascanner.GenerateAnalysisFiles()

pkg/mediascanner/mediascanner.go
└── New function: GenerateAnalysisFiles()
    ├── Generate beat grid from track BPM
    ├── Extract cues from Mixxx database (if available)
    ├── Generate waveforms from audio file
    └── Write .DAT file using anlz package

pkg/mixxx/
└── New query: GetTrackCues()
    └── Extract hot cues from Mixxx database

pkg/rekordbox/anlz/
└── New package with all analysis file functionality
```

## Implementation Phases

### Phase 1: Foundation (Week 1)
**Goal:** Create package structure and basic file I/O

Files to create:
- `pkg/rekordbox/anlz/README.md`
- `pkg/rekordbox/anlz/anlz.go` + `anlz.md`
- `pkg/rekordbox/anlz/header.go` + `header.md`
- `pkg/rekordbox/anlz/tag.go` + `tag.md`
- `pkg/rekordbox/anlz/anlz_test.go`

**Deliverable:** Can create empty PMAI files with correct structure

### Phase 2: Path & Beat Grid (Week 2)
**Goal:** Minimum viable CDJ functionality

Files to create:
- `pkg/rekordbox/anlz/path.go` + `path.md`
- `pkg/rekordbox/anlz/beatgrid.go` + `beatgrid.md`
- Integration into `cmd/rex/main.go`
- New functions in `pkg/mediascanner/mediascanner.go`

**Deliverable:** CDJs can load tracks and sync BPM

### Phase 3: Cue Points (Week 3)
**Goal:** Full DJ workflow support

Files to create:
- `pkg/rekordbox/anlz/cue.go` + `cue.md`
- New query in `pkg/mixxx/query.sql`
- Run sqlc to generate Go code

**Deliverable:** Hot cues and loops work on CDJs

### Phase 4: Basic Waveforms (Week 4)
**Goal:** Visual navigation

Files to create:
- `pkg/rekordbox/anlz/waveform.go` + `waveform.md`
- Audio analysis helpers in mediascanner

**Deliverable:** Waveform preview displays on CDJs

### Phase 5: VBR Support (Week 5)
**Goal:** Better seeking in VBR tracks

Files to create:
- `pkg/rekordbox/anlz/vbr.go` + `vbr.md`

**Deliverable:** Fast seeking in variable bitrate files

### Phase 6: Enhanced Waveforms (Weeks 6-7)
**Goal:** Modern CDJ feature support

Updates to:
- `pkg/rekordbox/anlz/waveform.go`
- Generate .EXT files

**Deliverable:** Color waveforms on nxs2 players

### Phase 7: Song Structure (Week 8)
**Goal:** Lighting control support

Files to create:
- `pkg/rekordbox/anlz/structure.go` + `structure.md`

**Deliverable:** CDJ-3000 lighting integration

## Testing Strategy

### Unit Tests
Each `.go` file should have corresponding tests in `anlz_test.go`:
- Marshal/unmarshal round-trip tests
- Byte order verification (big-endian)
- Tag length calculations

### Integration Tests
1. Generate analysis files with REX
2. Use `cmd/analyze/main.go` to inspect structure
3. Compare with real rekordbox files (in `testdata/`)
4. Test on actual CDJ hardware

### Test Data
Add to `testdata/`:
- `sample_beatgrid.dat` - Reference PQTZ tag
- `sample_cues.dat` - Reference PCO2 tag
- `sample_complete.dat` - Complete analysis file

## File Naming Convention

Analysis files follow a specific naming pattern:
```
/PIONEER/USBANLZ/{prefix}/{hash}/ANLZ0000.DAT
```

Where:
- `{prefix}` = First 3 characters of track hash (e.g., "P016")
- `{hash}` = 8-character hex hash of track path (e.g., "0000875E")

The hash generation is already implemented in mediascanner for the `AnalyzePath` field.

## Key Technical Considerations

### 1. Byte Order
- **PDB files:** Little-endian (`binary.LittleEndian`)
- **ANLZ files:** Big-endian (`binary.BigEndian`)
- Create constant in `anlz.go`: `var ByteOrder = binary.BigEndian`

### 2. Tag Length Calculation
Every tag must accurately calculate its total length including header:
```go
tagLen := headerLen + dataLen
```

### 3. File Length Calculation
The PMAI header must store total file size:
```go
fileLen := 28 + sum(allTagLengths)
```

### 4. UTF-16 Big-Endian Strings
Path and comment strings use UTF-16BE with trailing NUL:
```go
// Convert string to UTF-16BE
runes := []rune(str)
encoded := utf16.Encode(runes)
bytes := make([]byte, (len(encoded)+1)*2) // +1 for NUL
for i, r := range encoded {
    binary.BigEndian.PutUint16(bytes[i*2:], r)
}
```

### 5. Audio Analysis Options

For generating beat grids and waveforms, choose:

**Option A: Use Mixxx Data** (Recommended for Phase 2)
- Mixxx already analyzed the track
- Query its SQLite tables
- Fastest, no external dependencies

**Option B: FFmpeg** (Recommended for waveforms)
- Shell out to ffmpeg for audio decoding
- Extract amplitude data
- Reliable, widely available

**Option C: Go Audio Libraries**
- Pure Go implementation
- More complex but self-contained
- Consider for future optimization

## Success Criteria

### Phase 2 Complete (Minimum Viable)
- [ ] Can export tracks from REX
- [ ] CDJ loads tracks successfully
- [ ] BPM detection and sync works
- [ ] Beat grid appears on waveform display

### Phase 3 Complete (Full Workflow)
- [ ] Hot cues appear on CDJ
- [ ] Loops work correctly
- [ ] Memory points function

### Phase 4 Complete (Professional Grade)
- [ ] Waveform preview displays
- [ ] Needle drop works
- [ ] Visual navigation functions

### All Phases Complete (Feature Parity)
- [ ] All features match rekordbox export
- [ ] Works on all CDJ models (2000nxs, nxs2, 3000)
- [ ] No rekordbox warnings when imported
- [ ] Performance is acceptable (<2s per track)

## Reference Documentation

All implementation details are sourced from:
https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#analysis-files

Key sections:
- Analysis File Header
- Beat Grid Tag (PQTZ)
- Extended Cue List Tag (PCO2)
- Path Tag (PPTH)
- Waveform Tags (PWAV, PWV3, PWV5, etc.)
- Song Structure Tag (PSSI)

## Next Steps

1. Read through all `.md` files in `pkg/rekordbox/anlz/`
2. Start with Phase 1: Create package structure
3. Implement and test each phase sequentially
4. Use existing PDB generation as a reference pattern
5. Test frequently with real hardware

## Questions & Issues

Track issues and questions in GitHub issues with label `anlz-implementation`.

Common questions:
- **Q:** Why big-endian for ANLZ but little-endian for PDB?
- **A:** Different Pioneer developers/teams, historical reasons

- **Q:** Can we skip waveforms?
- **A:** Yes, but navigation is much harder for DJs

- **Q:** Do we need .EXT and .2EX files?
- **A:** No for basic functionality, yes for modern players

- **Q:** How to test without CDJ hardware?
- **A:** Import into rekordbox software, check for errors

