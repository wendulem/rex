# waveform.go - Waveform Tags (PWAV, PWV2-PWV7)

## Purpose

This file implements all waveform tags for visual track representation. There are multiple waveform types with increasing complexity and color depth.

## Waveform Types

| Tag | Name | Size | Colors | File | Priority |
|-----|------|------|--------|------|----------|
| PWAV | Preview | 400 bytes | Mono (blue shades) | .DAT | Medium |
| PWV2 | Tiny Preview | 100 bytes | Mono (blue) | .DAT | Low |
| PWV3 | Detail | Variable | Mono (blue shades) | .EXT | Medium |
| PWV4 | Color Preview | 7,200 bytes | RGB | .EXT | High |
| PWV5 | Color Detail | Variable | RGB | .EXT | High |
| PWV6 | 3-Band Preview | 3,600 bytes | 3-band | .2EX | Low |
| PWV7 | 3-Band Detail | Variable | 3-band | .2EX | Low |

## Source Documentation

From: https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#waveform-preview-tag

> Waveform tags hold visual representations of the track, displayed on CDJs for navigation and needle drop.

### Common Structure

All waveform tags share this envelope:

```
0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
┌──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┐
│fourcc (PWxx)│ len_header  │   len_tag   │lebts│ 00
├──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┤
│ len_entries │   unknown   │   entries...      │ 10
└────────────────────────────────────────────────┘
```

## Implementation Overview

### Base Waveform Struct

```go
type WaveformTag struct {
    FourCCValue    [4]byte
    HeaderLen      uint32
    EntryBytes     uint32  // Bytes per entry
    Entries        []byte  // Raw waveform data
}
```

### PWAV - Waveform Preview (400 bytes)

**Monochrome preview for original nexus, 400 columns covering entire track.**

```go
type WaveformPreviewTag struct {
    Entries [400]byte
}

func (t *WaveformPreviewTag) FourCC() [4]byte {
    return [4]byte{'P', 'W', 'A', 'V'}
}

// Each byte encodes:
// - 5 low bits: height (0-31 pixels)
// - 3 high bits: whiteness (0-7, whiter = higher value)
func encodePreviewByte(height uint8, whiteness uint8) byte {
    return (height & 0x1F) | ((whiteness & 0x07) << 5)
}
```

### PWV2 - Tiny Waveform Preview (100 bytes)

**Even smaller preview for CDJ-900, 100 columns.**

```go
type WaveformTinyPreviewTag struct {
    Entries [100]byte
}

func (t *WaveformTinyPreviewTag) FourCC() [4]byte {
    return [4]byte{'P', 'W', 'V', '2'}
}

// Each byte encodes:
// - 4 low bits: height (0-15 pixels)
// - 4 high bits: unused
func encodeTinyByte(height uint8) byte {
    return height & 0x0F
}
```

### PWV3 - Waveform Detail (Variable)

**Detailed scrolling waveform, 150 entries per second (75 frames × 2).**

```go
type WaveformDetailTag struct {
    Entries []byte  // len = duration_seconds * 150
}

func (t *WaveformDetailTag) FourCC() [4]byte {
    return [4]byte{'P', 'W', 'V', '3'}
}

// Same encoding as PWAV: height + whiteness
```

### PWV4 - Waveform Color Preview (1,200 entries)

**Color waveform preview for nxs2, 1,200 columns.**

```go
type WaveformColorPreviewTag struct {
    Entries [][6]byte  // 1,200 entries, 6 bytes each
}

func (t *WaveformColorPreviewTag) FourCC() [4]byte {
    return [4]byte{'P', 'W', 'V', '4'}
}

// Each 6-byte entry is complex (see documentation)
// Contains frequency band information and colors
```

### PWV5 - Waveform Color Detail (Variable)

**Detailed color waveform, 150 entries per second.**

```go
type WaveformColorDetailTag struct {
    Entries [][2]byte  // len = duration_seconds * 150
}

func (t *WaveformColorDetailTag) FourCC() [4]byte {
    return [4]byte{'P', 'W', 'V', '5'}
}

// Each 2-byte entry (as big-endian uint16):
// Bits 15-13: Red (3 bits)
// Bits 12-10: Green (3 bits)
// Bits 9-7:   Blue (3 bits)
// Bits 6-2:   Height (5 bits)
// Bits 1-0:   Unused
func encodeColorDetail(r, g, b, height uint8) uint16 {
    return (uint16(r&0x07) << 13) |
           (uint16(g&0x07) << 10) |
           (uint16(b&0x07) << 7) |
           (uint16(height&0x1F) << 2)
}
```

### PWV6 - 3-Band Preview (1,200 entries)

**Three frequency bands (low/mid/high) for CDJ-3000.**

```go
type Waveform3BandPreviewTag struct {
    Entries [][3]byte  // 1,200 entries, 3 bytes each
}

func (t *Waveform3BandPreviewTag) FourCC() [4]byte {
    return [4]byte{'P', 'W', 'V', '6'}
}

// Each entry: [mid, high, low] byte heights
```

### PWV7 - 3-Band Detail (Variable)

**Detailed 3-band waveform, 150 entries per second.**

```go
type Waveform3BandDetailTag struct {
    Entries [][3]byte  // len = duration_seconds * 150
}

func (t *WaveformWaveform3BandDetailTag) FourCC() [4]byte {
    return [4]byte{'P', 'W', 'V', '7'}
}

// Same as PWV6: [mid, high, low] bytes
```

## Waveform Generation

### From Audio File (FFmpeg)

```go
func GenerateWaveformFromAudio(audioPath string, duration float64) (*WaveformPreviewTag, error) {
    // Use FFmpeg to extract amplitude data
    cmd := exec.Command("ffmpeg",
        "-i", audioPath,
        "-af", "aresample=8000,astats=metadata=1:reset=1",
        "-f", "null", "-")
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, err
    }
    
    // Parse FFmpeg output for amplitude data
    amplitudes := parseAmplitudes(output)
    
    // Downsample to 400 points
    tag := &WaveformPreviewTag{}
    step := len(amplitudes) / 400
    
    for i := 0; i < 400; i++ {
        idx := i * step
        if idx >= len(amplitudes) {
            break
        }
        
        amp := amplitudes[idx]
        height := uint8(amp * 31)  // Scale to 0-31
        whiteness := uint8(amp * 7)  // Scale to 0-7
        
        tag.Entries[i] = encodePreviewByte(height, whiteness)
    }
    
    return tag, nil
}
```

### Simple Placeholder Waveform

For initial implementation without audio analysis:

```go
func GeneratePlaceholderWaveform() *WaveformPreviewTag {
    tag := &WaveformPreviewTag{}
    
    // Create simple sine wave pattern
    for i := 0; i < 400; i++ {
        t := float64(i) / 400.0 * 2 * math.Pi
        amp := (math.Sin(t) + 1) / 2  // 0-1
        
        height := uint8(amp * 15 + 8)  // 8-23 range
        whiteness := 3  // Medium whiteness
        
        tag.Entries[i] = encodePreviewByte(height, whiteness)
    }
    
    return tag
}
```

## Integration Points

### Minimal Implementation (Phase 1)

Start with just PWAV (preview):

```go
func GenerateAnalysisFiles(track *library.Track) error {
    file := &anlz.File{
        Header: anlz.NewFileHeader(),
        Tags: []anlz.Tag{
            // ... path, beatgrid, cues
            anlz.GeneratePlaceholderWaveform(),
        },
    }
    
    return file.WriteToFile(track.AnalyzePath)
}
```

### Full Implementation (Phase 2)

Add color waveforms in .EXT file:

```go
func GenerateExtendedAnalysisFile(track *library.Track) error {
    file := &anlz.File{
        Header: anlz.NewFileHeader(),
        Tags: []anlz.Tag{
            GenerateColorPreview(track),
            GenerateColorDetail(track),
        },
    }
    
    extPath := strings.Replace(track.AnalyzePath, ".DAT", ".EXT", 1)
    return file.WriteToFile(extPath)
}
```

## Testing Requirements

### Test Cases

1. **Preview Encoding**
```go
func TestWaveformPreview_Encoding(t *testing.T) {
    tag := &WaveformPreviewTag{}
    
    // Set first entry: height=15, whiteness=5
    tag.Entries[0] = encodePreviewByte(15, 5)
    
    // Verify encoding
    height := tag.Entries[0] & 0x1F
    whiteness := (tag.Entries[0] >> 5) & 0x07
    
    assert.Equal(t, uint8(15), height)
    assert.Equal(t, uint8(5), whiteness)
}
```

2. **Color Detail Encoding**
```go
func TestWaveformColorDetail_Encoding(t *testing.T) {
    // RGB(7,6,5), height=20
    encoded := encodeColorDetail(7, 6, 5, 20)
    
    r := (encoded >> 13) & 0x07
    g := (encoded >> 10) & 0x07
    b := (encoded >> 7) & 0x07
    h := (encoded >> 2) & 0x1F
    
    assert.Equal(t, uint16(7), r)
    assert.Equal(t, uint16(6), g)
    assert.Equal(t, uint16(5), b)
    assert.Equal(t, uint16(20), h)
}
```

## Common Pitfalls

### ❌ Wrong Entry Count
```go
// WRONG - Must be exact size
Entries: make([]byte, 395)

// CORRECT - PWAV is always 400 bytes
Entries: [400]byte{}
```

### ❌ Height Out of Range
```go
// WRONG - Height for PWAV is 0-31 (5 bits)
height := uint8(35)

// CORRECT - Clamp to valid range
height := uint8(math.Min(float64(amp*31), 31))
```

### ❌ Wrong Byte Order for Color Detail
```go
// WRONG - Must be big-endian
binary.LittleEndian.PutUint16(buf, colorValue)

// CORRECT
ByteOrder.PutUint16(buf, colorValue)
```

## Performance Notes

- Waveform generation is I/O and CPU intensive
- Consider caching generated waveforms
- Can parallelize across multiple tracks
- Audio decoding is the bottleneck

## Future Enhancements

1. **GPU Acceleration** - Use GPU for waveform rendering
2. **Caching** - Store generated waveforms
3. **Quality Levels** - Low/medium/high quality options
4. **Real-time Generation** - Stream while copying tracks

## Related Files

- `anlz.go` - Uses waveform tags in File.Tags
- `tag.go` - Waveform tags implement Tag interface
- `pkg/mediascanner/mediascanner.go` - Generates waveforms

## Reference

Deep Symmetry Documentation:
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#waveform-preview-tag
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#waveform-detail-tag
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#waveform-color-preview-tag
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#waveform-color-detail-tag
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#waveform-3-band-preview-tag
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#waveform-3-band-detail-tag

Waveforms enable:
- Visual navigation and scrubbing
- Needle drop to specific positions
- Visual beatmatching
- Track structure visualization

