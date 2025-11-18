# beatgrid.go - PQTZ Beat Grid Tag

## Purpose

This file implements the PQTZ (Beat Grid) tag, which stores beat timing and tempo information for a track. This is one of the **most critical** tags - without it, CDJs cannot sync tracks or display accurate beat information.

## Source Documentation

From: https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#beat-grid-tag

> This kind of section holds a list of all beats found within the track, recording their bar position, the time at which they occur, and the tempo at that point.

The tag identifier **PQTZ** may stand for "Pioneer Quantization".

### Tag Structure

```
Byte Offset
0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
┌──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┐
│P │Q │T │Z │ len_header  │   len_tag   │unk1 │ 00
├──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┤
│     unknown2    │  len_beats  │  Beat entries │ 10
│                                                │ 20
│ (8 bytes per beat)                            │
└────────────────────────────────────────────────┘
```

### Beat Entry Structure

```
Byte Offset
0  1  2  3  4  5  6  7
┌──┬──┬──┬──┬──┬──┬──┬──┐
│bnum │ tempo │   time    │
└──┴──┴──┴──┴──┴──┴──┴──┘
```

### Field Descriptions

**Tag Header:**
| Offset | Size | Name | Description |
|--------|------|------|-------------|
| 0x00 | 4 | FourCC | "PQTZ" identifier |
| 0x04 | 4 | HeaderLen | Always 0x18 (24 bytes) |
| 0x08 | 4 | TagLen | Total length (header + all beats) |
| 0x0c | 4 | Unknown1 | Purpose unknown |
| 0x10 | 4 | Unknown2 | Seems to always be 0x00800000 |
| 0x14 | 4 | NumBeats | Number of beat entries following |

**Beat Entry (8 bytes each):**
| Offset | Size | Name | Description |
|--------|------|------|-------------|
| 0x00 | 2 | BeatNumber | Position in measure: 1, 2, 3, or 4 |
| 0x02 | 2 | Tempo | BPM × 100 (e.g., 12500 = 125.00 BPM) |
| 0x04 | 4 | Time | Milliseconds from track start (at normal speed) |

**Key Points:**
- BeatNumber cycles 1, 2, 3, 4 for each bar
- Tempo allows 0.01 BPM precision
- Time is always at normal playback speed
- Tempo can vary throughout track (for tempo changes)

## Implementation

### BeatGridTag Struct

```go
type BeatGridTag struct {
    Beats []Beat
}

type Beat struct {
    BeatNumber uint16  // 1, 2, 3, or 4 (position in bar)
    Tempo      uint16  // BPM × 100 (12500 = 125.00 BPM)
    Time       uint32  // Milliseconds from start
}
```

### Interface Implementation

```go
func (t *BeatGridTag) FourCC() [4]byte {
    return [4]byte{'P', 'Q', 'T', 'Z'}
}

func (t *BeatGridTag) TagType() TagType {
    return TagTypeBeatGrid
}
```

### Marshal (Encode to Bytes)

```go
func (t *BeatGridTag) MarshalBinary() ([]byte, error) {
    headerLen := uint32(0x18)  // 24 bytes
    numBeats := uint32(len(t.Beats))
    beatDataLen := numBeats * 8  // 8 bytes per beat
    tagLen := headerLen + beatDataLen
    
    buf := make([]byte, tagLen)
    
    // Write tag header
    copy(buf[0:4], t.FourCC()[:])
    ByteOrder.PutUint32(buf[4:8], headerLen)
    ByteOrder.PutUint32(buf[8:12], tagLen)
    
    // Unknown fields
    ByteOrder.PutUint32(buf[12:16], 0)  // Unknown1
    ByteOrder.PutUint32(buf[16:20], 0x00800000)  // Unknown2
    
    // Number of beats
    ByteOrder.PutUint32(buf[20:24], numBeats)
    
    // Write beat entries
    offset := 24
    for _, beat := range t.Beats {
        ByteOrder.PutUint16(buf[offset:offset+2], beat.BeatNumber)
        ByteOrder.PutUint16(buf[offset+2:offset+4], beat.Tempo)
        ByteOrder.PutUint32(buf[offset+4:offset+8], beat.Time)
        offset += 8
    }
    
    return buf, nil
}
```

### Unmarshal (Decode from Bytes)

```go
func (t *BeatGridTag) UnmarshalBinary(data []byte) error {
    // Parse header
    header := &TagHeader{}
    if err := header.UnmarshalBinary(data); err != nil {
        return err
    }
    
    // Verify FourCC
    if header.FourCC != t.FourCC() {
        return fmt.Errorf("wrong fourcc: %s (expected PQTZ)", header.FourCC)
    }
    
    // Verify header length
    if header.HeaderLen != 0x18 {
        return fmt.Errorf("unexpected header length: %d", header.HeaderLen)
    }
    
    // Read number of beats
    if len(data) < 24 {
        return fmt.Errorf("data too short for beat count")
    }
    numBeats := ByteOrder.Uint32(data[20:24])
    
    // Read beat entries
    t.Beats = make([]Beat, numBeats)
    offset := 24
    
    for i := uint32(0); i < numBeats; i++ {
        if len(data) < offset+8 {
            return fmt.Errorf("data too short for beat %d", i)
        }
        
        t.Beats[i] = Beat{
            BeatNumber: ByteOrder.Uint16(data[offset : offset+2]),
            Tempo:      ByteOrder.Uint16(data[offset+2 : offset+4]),
            Time:       ByteOrder.Uint32(data[offset+4 : offset+8]),
        }
        offset += 8
    }
    
    return nil
}
```

## Beat Grid Generation

### From Constant BPM

```go
func GenerateConstantBPMGrid(bpm float64, durationMs uint32) *BeatGridTag {
    beatsPerMinute := bpm
    beatsPerSecond := beatsPerMinute / 60.0
    msPerBeat := 1000.0 / beatsPerSecond
    
    beats := []Beat{}
    beatNumber := uint16(1)
    currentTime := uint32(0)
    tempoBPM100 := uint16(bpm * 100)
    
    for currentTime < durationMs {
        beats = append(beats, Beat{
            BeatNumber: beatNumber,
            Tempo:      tempoBPM100,
            Time:       currentTime,
        })
        
        // Advance beat
        beatNumber++
        if beatNumber > 4 {
            beatNumber = 1
        }
        
        currentTime += uint32(msPerBeat)
    }
    
    return &BeatGridTag{Beats: beats}
}
```

### From Mixxx Beat Analysis

```go
// Assuming Mixxx stores beats in its database
func GenerateFromMixxxBeats(mixxxBeats []MixxxBeat, trackBPM float64) *BeatGridTag {
    beats := make([]Beat, len(mixxxBeats))
    tempoBPM100 := uint16(trackBPM * 100)
    
    for i, mb := range mixxxBeats {
        beats[i] = Beat{
            BeatNumber: uint16((i % 4) + 1),  // Cycle 1-4
            Tempo:      tempoBPM100,
            Time:       uint32(mb.PositionMs),
        }
    }
    
    return &BeatGridTag{Beats: beats}
}
```

### From Audio Analysis

```go
// Using external beat detection tool (e.g., aubio)
func GenerateFromAudioFile(audioPath string) (*BeatGridTag, error) {
    // Run beat detection
    cmd := exec.Command("aubio", "beat", "-i", audioPath, "-O", "json")
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    
    // Parse results
    var aubioBeats []float64
    json.Unmarshal(output, &aubioBeats)
    
    // Convert to Beat structs
    beats := make([]Beat, len(aubioBeats))
    for i, timeSeconds := range aubioBeats {
        timeMs := uint32(timeSeconds * 1000)
        
        // Calculate tempo from beat spacing
        var tempo uint16
        if i > 0 {
            beatSpacing := timeMs - beats[i-1].Time
            bpm := 60000.0 / float64(beatSpacing)
            tempo = uint16(bpm * 100)
        }
        
        beats[i] = Beat{
            BeatNumber: uint16((i % 4) + 1),
            Tempo:      tempo,
            Time:       timeMs,
        }
    }
    
    return &BeatGridTag{Beats: beats}, nil
}
```

## Integration Points

### Called By
- `pkg/mediascanner/mediascanner.go::GenerateAnalysisFiles()`

### Data Sources
1. **Track BPM** (from Mixxx database)
2. **Track Duration** (from track metadata)
3. **Beat positions** (from Mixxx analysis or audio analysis)

### Example Usage in Mediascanner

```go
func GenerateAnalysisFiles(track *library.Track, lib *library.Library) error {
    file := &anlz.File{
        Header: anlz.NewFileHeader(),
        Tags:   []anlz.Tag{},
    }
    
    // Generate beat grid from track BPM
    beatGrid := anlz.GenerateConstantBPMGrid(
        track.BPM,
        uint32(track.Duration*1000),
    )
    
    file.Tags = append(file.Tags, beatGrid)
    // ... add other tags
    
    return file.WriteToFile(track.AnalyzePath)
}
```

## Testing Requirements

### Test Cases

1. **Simple Beat Grid**
```go
func TestBeatGridTag_Simple(t *testing.T) {
    tag := &BeatGridTag{
        Beats: []Beat{
            {BeatNumber: 1, Tempo: 12000, Time: 0},
            {BeatNumber: 2, Tempo: 12000, Time: 500},
            {BeatNumber: 3, Tempo: 12000, Time: 1000},
            {BeatNumber: 4, Tempo: 12000, Time: 1500},
        },
    }
    
    data, err := tag.MarshalBinary()
    assert.NoError(t, err)
    
    // Verify FourCC
    assert.Equal(t, []byte("PQTZ"), data[0:4])
    
    // Verify beat count
    numBeats := binary.BigEndian.Uint32(data[20:24])
    assert.Equal(t, uint32(4), numBeats)
}
```

2. **Round-trip**
```go
func TestBeatGridTag_RoundTrip(t *testing.T) {
    original := &BeatGridTag{
        Beats: []Beat{
            {1, 12500, 0},
            {2, 12500, 480},
            {3, 12500, 960},
            {4, 12500, 1440},
        },
    }
    
    data, _ := original.MarshalBinary()
    
    loaded := &BeatGridTag{}
    err := loaded.UnmarshalBinary(data)
    assert.NoError(t, err)
    assert.Equal(t, original.Beats, loaded.Beats)
}
```

3. **Empty Beat Grid**
```go
func TestBeatGridTag_Empty(t *testing.T) {
    tag := &BeatGridTag{Beats: []Beat{}}
    
    data, err := tag.MarshalBinary()
    assert.NoError(t, err)
    
    // Should have header only (24 bytes)
    assert.Equal(t, 24, len(data))
}
```

4. **Tempo Precision**
```go
func TestBeatGridTag_TempoPrecision(t *testing.T) {
    tag := &BeatGridTag{
        Beats: []Beat{
            {1, 12534, 0},  // 125.34 BPM
        },
    }
    
    data, _ := tag.MarshalBinary()
    loaded := &BeatGridTag{}
    loaded.UnmarshalBinary(data)
    
    // Verify precision preserved
    assert.Equal(t, uint16(12534), loaded.Beats[0].Tempo)
    
    // Convert back to BPM
    bpm := float64(loaded.Beats[0].Tempo) / 100.0
    assert.Equal(t, 125.34, bpm)
}
```

5. **Constant BPM Generation**
```go
func TestGenerateConstantBPMGrid(t *testing.T) {
    // Generate 2 seconds at 120 BPM
    tag := GenerateConstantBPMGrid(120.0, 2000)
    
    // 120 BPM = 2 beats/second = 4 beats in 2 seconds
    assert.Equal(t, 4, len(tag.Beats))
    
    // Verify beat numbers cycle
    assert.Equal(t, uint16(1), tag.Beats[0].BeatNumber)
    assert.Equal(t, uint16(2), tag.Beats[1].BeatNumber)
    assert.Equal(t, uint16(3), tag.Beats[2].BeatNumber)
    assert.Equal(t, uint16(4), tag.Beats[3].BeatNumber)
    
    // Verify tempo
    assert.Equal(t, uint16(12000), tag.Beats[0].Tempo)
}
```

## Usage Examples

### Creating Beat Grid

```go
// From constant BPM
beatGrid := anlz.GenerateConstantBPMGrid(125.5, 180000) // 3 minutes at 125.5 BPM

// Manually
beatGrid := &anlz.BeatGridTag{
    Beats: []anlz.Beat{
        {BeatNumber: 1, Tempo: 12550, Time: 0},
        {BeatNumber: 2, Tempo: 12550, Time: 477},
        {BeatNumber: 3, Tempo: 12550, Time: 954},
        {BeatNumber: 4, Tempo: 12550, Time: 1431},
    },
}
```

### Reading Beat Grid

```go
tag := &anlz.BeatGridTag{}
if err := tag.UnmarshalBinary(data); err != nil {
    log.Fatal(err)
}

fmt.Printf("Track has %d beats\n", len(tag.Beats))
fmt.Printf("First beat at %d ms\n", tag.Beats[0].Time)

bpm := float64(tag.Beats[0].Tempo) / 100.0
fmt.Printf("Tempo: %.2f BPM\n", bpm)
```

### Analyzing Beat Grid

```go
func AnalyzeBeatGrid(tag *BeatGridTag) {
    if len(tag.Beats) == 0 {
        fmt.Println("No beats found")
        return
    }
    
    // Calculate average BPM
    totalTempo := uint32(0)
    for _, beat := range tag.Beats {
        totalTempo += uint32(beat.Tempo)
    }
    avgBPM := float64(totalTempo) / float64(len(tag.Beats)) / 100.0
    fmt.Printf("Average BPM: %.2f\n", avgBPM)
    
    // Check for tempo changes
    firstTempo := tag.Beats[0].Tempo
    hasTempoChange := false
    for _, beat := range tag.Beats {
        if beat.Tempo != firstTempo {
            hasTempoChange = true
            break
        }
    }
    fmt.Printf("Tempo changes: %v\n", hasTempoChange)
}
```

## Common Pitfalls

### ❌ Wrong Beat Number Range
```go
// WRONG - Beat numbers are 1-4, not 0-3
BeatNumber: uint16(i % 4)

// CORRECT
BeatNumber: uint16((i % 4) + 1)
```

### ❌ BPM Instead of BPM×100
```go
// WRONG - Tempo should be multiplied by 100
Tempo: uint16(bpm)

// CORRECT
Tempo: uint16(bpm * 100)
```

### ❌ Using Seconds Instead of Milliseconds
```go
// WRONG - Time is in milliseconds
Time: uint32(timeSeconds)

// CORRECT
Time: uint32(timeSeconds * 1000)
```

### ❌ Forgetting Unknown2 Value
```go
// WRONG - Unknown2 should be 0x00800000
ByteOrder.PutUint32(buf[16:20], 0)

// CORRECT
ByteOrder.PutUint32(buf[16:20], 0x00800000)
```

### ❌ Not Handling Variable Tempo
```go
// WRONG - Assuming constant tempo
tempo := uint16(trackBPM * 100)
for _, beat := range beats {
    beat.Tempo = tempo  // All same
}

// CORRECT - Calculate tempo from beat spacing
for i := range beats {
    if i > 0 {
        spacing := beats[i].Time - beats[i-1].Time
        bpm := 60000.0 / float64(spacing)
        beats[i].Tempo = uint16(bpm * 100)
    }
}
```

## Performance Notes

- Beat grid generation is CPU-intensive for audio analysis
- Constant BPM generation is fast (< 10ms for typical track)
- Consider caching beat grids
- Typical track: 400-800 beats (3-4 minutes at 120 BPM)
- Beat grid size: 24 + (beats × 8) bytes

## Beat Detection Quality

### Good Beat Detection
- Aligned with musical measures (4/4 time)
- First beat starts near beginning of track
- Consistent spacing (for constant BPM tracks)
- Accurate downbeats (beat 1 of each bar)

### Poor Beat Detection
- Off by half-beat
- Missing beats
- Extra beats on drum fills
- Wrong downbeat placement

**Recommendation:** Use Mixxx's analysis when available - it's well-tested.

## Future Enhancements

1. **Tempo Change Detection** - Identify sections with different tempos
2. **Beat Quality Scoring** - Validate beat grid quality
3. **Auto-correction** - Fix common beat grid errors
4. **Time Signature Support** - Handle 3/4, 6/8, etc. (Currently assumes 4/4)

## Related Files

- `anlz.go` - Uses BeatGridTag in File.Tags
- `tag.go` - BeatGridTag implements Tag interface
- `pkg/mediascanner/mediascanner.go` - Generates beat grids
- `pkg/mixxx/query.sql` - Queries for beat data

## Reference

Deep Symmetry Documentation:
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#beat-grid-tag

Beat grid is essential for:
- BPM sync between players
- Quantize feature
- Beat jump navigation
- Auto-loop lengths

