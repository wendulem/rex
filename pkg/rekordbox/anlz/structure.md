# structure.go - PSSI Song Structure Tag

## Purpose

This file implements the PSSI (Song Structure) tag, which provides phrase-level analysis of tracks for rekordbox Performance Mode and CDJ-3000 lighting control. It identifies musical sections like intro, verse, chorus, bridge, and outro.

## Source Documentation

From: https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#song-structure-tag

> This kind of section was originally used only in rekordbox Performance Mode, but starting with rekordbox version 6 it also gets exported to external media so CDJ-3000 players can use it to control lighting looks.

**Note:** Rekordbox 6 exports use XOR masking to obfuscate this data.

### Tag Structure

```
Byte Offset
0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
┌──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┐
│P │S │S │I │ len_header  │   len_tag   │lebts│ 00
├──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┤
│ lene│mood│  unknown    │ end │unk2 │bank│u3 │ 10
├──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┤
│           Phrase entries (24 bytes each)      │ 20
└────────────────────────────────────────────────┘
```

### Phrase Entry Structure

```
Byte Offset
0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
┌──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┐
│ index   │  beat   │  kind   │k1│b2│k2│b │beat2│ 00
├──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┤
│ beat3   │ beat4   │k3│u4│fill│beatfill       │ 10
└────────────────────────────────────────────────┘
```

### Field Descriptions

**Tag Header:**
| Offset | Size | Name | Description |
|--------|------|------|-------------|
| 0x00 | 4 | FourCC | "PSSI" identifier |
| 0x04 | 4 | HeaderLen | Always 0x20 (32 bytes) |
| 0x08 | 4 | TagLen | Total length (header + all entries) |
| 0x0c | 4 | EntryBytes | Always 0x18 (24 bytes per entry) |
| 0x10 | 2 | NumEntries | Number of phrase entries |
| 0x12 | 2 | Mood | Overall mood: 1=High, 2=Mid, 3=Low |
| 0x14 | 6 | Unknown | Purpose unclear |
| 0x1a | 2 | EndBeat | Beat where last phrase ends |
| 0x1c | 2 | Unknown2 | Purpose unclear |
| 0x1e | 1 | Bank | Style bank (0=default/Cool, 1=Cool, 2=Natural, etc.) |
| 0x1f | 1 | Unknown3 | Purpose unclear |

**Phrase Entry (24 bytes):**
| Offset | Size | Name | Description |
|--------|------|------|-------------|
| 0x00 | 2 | Index | Phrase number (1-based) |
| 0x02 | 2 | Beat | Beat where phrase starts |
| 0x04 | 2 | Kind | Phrase type (depends on mood) |
| 0x06 | 1 | K1 | Flag affecting phrase label |
| 0x07 | 1 | B2 | Unknown |
| 0x08 | 1 | K2 | Flag affecting phrase label |
| 0x09 | 1 | B | Beat count flag (0 or 1) |
| 0x0a | 2 | Beat2 | Extra beat position |
| 0x0c | 2 | Beat3 | Extra beat position |
| 0x0e | 2 | Beat4 | Extra beat position |
| 0x10 | 1 | K3 | Flag affecting phrase label |
| 0x11 | 1 | Unknown4 | Purpose unclear |
| 0x12 | 1 | Fill | Non-zero if fill-in present |
| 0x14 | 2 | FillBeat | Beat where fill-in starts |

## XOR Masking

Rekordbox 6 exports apply XOR masking to bytes after `NumEntries`:

```go
func UnmaskPSSI(data []byte, numEntries uint16) []byte {
    // XOR mask pattern
    baseMask := []byte{
        0xCB, 0xE1, 0xEE, 0xFA, 0xE5, 0xEE, 0xAD, 0xEE,
        0xE9, 0xD2, 0xE9, 0xEB, 0xE1, 0xE9, 0xF3, 0xE8,
        0xE9, 0xF4, 0xE1,
    }
    
    // Add numEntries to each byte
    mask := make([]byte, len(baseMask))
    for i := range mask {
        mask[i] = baseMask[i] + byte(numEntries)
    }
    
    // XOR all bytes after offset 0x10
    unmasked := make([]byte, len(data))
    copy(unmasked, data)
    
    offset := 0x12  // After NumEntries
    maskIdx := 0
    
    for offset < len(unmasked) {
        unmasked[offset] ^= mask[maskIdx%len(mask)]
        offset++
        maskIdx++
    }
    
    return unmasked
}
```

## Implementation

### SongStructureTag Struct

```go
type SongStructureTag struct {
    Mood         uint16         // 1=High, 2=Mid, 3=Low
    EndBeat      uint16         // Beat where last phrase ends
    Bank         uint8          // Style bank
    Entries      []PhraseEntry
    IsMasked     bool           // Whether to apply XOR masking on write
}

type PhraseEntry struct {
    Index        uint16  // Phrase number (1-based)
    Beat         uint16  // Start beat
    Kind         uint16  // Phrase type
    K1           uint8   // Flag
    K2           uint8   // Flag
    K3           uint8   // Flag
    B            uint8   // Beat count flag
    Beat2        uint16  // Extra beat
    Beat3        uint16  // Extra beat
    Beat4        uint16  // Extra beat
    Fill         uint8   // Fill-in flag
    FillBeat     uint16  // Fill-in start beat
}
```

### Phrase Labels by Mood

```go
// GetPhraseLabel returns the human-readable label for a phrase
func GetPhraseLabel(mood, kind uint16, k1, k2, k3 uint8) string {
    switch mood {
    case 1: // High mood
        return getHighMoodLabel(kind, k1, k2, k3)
    case 2: // Mid mood
        return getMidMoodLabel(kind)
    case 3: // Low mood
        return getLowMoodLabel(kind)
    default:
        return "Unknown"
    }
}

func getHighMoodLabel(kind uint16, k1, k2, k3 uint8) string {
    switch kind {
    case 1:
        if k1 == 1 {
            return "Intro 1"
        }
        return "Intro 2"
    case 2:
        if k2 == 0 && k3 == 0 {
            return "Up 1"
        }
        if k2 == 0 && k3 == 1 {
            return "Up 2"
        }
        if k2 == 1 && k3 == 0 {
            return "Up 3"
        }
        return "Up"
    case 3:
        return "Down"
    case 5:
        if k1 == 1 {
            return "Chorus 1"
        }
        return "Chorus 2"
    case 6:
        if k1 == 1 {
            return "Outro 1"
        }
        return "Outro 2"
    default:
        return fmt.Sprintf("Unknown %d", kind)
    }
}

func getMidMoodLabel(kind uint16) string {
    labels := map[uint16]string{
        1:  "Intro",
        2:  "Verse 1",
        3:  "Verse 2",
        4:  "Verse 3",
        5:  "Verse 4",
        6:  "Verse 5",
        7:  "Verse 6",
        8:  "Bridge",
        9:  "Chorus",
        10: "Outro",
    }
    if label, ok := labels[kind]; ok {
        return label
    }
    return fmt.Sprintf("Unknown %d", kind)
}

func getLowMoodLabel(kind uint16) string {
    labels := map[uint16]string{
        1:  "Intro",
        2:  "Verse 1",
        3:  "Verse 1",
        4:  "Verse 1",
        5:  "Verse 2",
        6:  "Verse 2",
        7:  "Verse 2",
        8:  "Bridge",
        9:  "Chorus",
        10: "Outro",
    }
    if label, ok := labels[kind]; ok {
        return label
    }
    return fmt.Sprintf("Unknown %d", kind)
}
```

### Marshal (Encode to Bytes)

```go
func (t *SongStructureTag) MarshalBinary() ([]byte, error) {
    headerLen := uint32(0x20)
    numEntries := uint16(len(t.Entries))
    entryLen := numEntries * 24
    tagLen := headerLen + uint32(entryLen)
    
    buf := make([]byte, tagLen)
    
    // Write header
    copy(buf[0:4], t.FourCC()[:])
    ByteOrder.PutUint32(buf[4:8], headerLen)
    ByteOrder.PutUint32(buf[8:12], tagLen)
    ByteOrder.PutUint32(buf[12:16], 0x18)  // Entry size
    ByteOrder.PutUint16(buf[16:18], numEntries)
    ByteOrder.PutUint16(buf[18:20], t.Mood)
    // Unknown bytes 20-26
    ByteOrder.PutUint16(buf[26:28], t.EndBeat)
    // Unknown bytes 28-30
    buf[30] = t.Bank
    // Unknown byte 31
    
    // Write entries
    offset := 32
    for _, entry := range t.Entries {
        ByteOrder.PutUint16(buf[offset:offset+2], entry.Index)
        ByteOrder.PutUint16(buf[offset+2:offset+4], entry.Beat)
        ByteOrder.PutUint16(buf[offset+4:offset+6], entry.Kind)
        buf[offset+6] = entry.K1
        // buf[offset+7] = unknown
        buf[offset+8] = entry.K2
        buf[offset+9] = entry.B
        ByteOrder.PutUint16(buf[offset+10:offset+12], entry.Beat2)
        ByteOrder.PutUint16(buf[offset+12:offset+14], entry.Beat3)
        ByteOrder.PutUint16(buf[offset+14:offset+16], entry.Beat4)
        buf[offset+16] = entry.K3
        // buf[offset+17] = unknown
        buf[offset+18] = entry.Fill
        ByteOrder.PutUint16(buf[offset+20:offset+22], entry.FillBeat)
        offset += 24
    }
    
    // Apply XOR masking if needed
    if t.IsMasked {
        buf = maskPSSI(buf, numEntries)
    }
    
    return buf, nil
}
```

## Song Structure Generation

### Placeholder Implementation

For initial implementation without audio analysis:

```go
func GeneratePlaceholderStructure(durationBeats uint16) *SongStructureTag {
    // Simple structure: Intro → Verse → Chorus → Outro
    entries := []PhraseEntry{
        {Index: 1, Beat: 1, Kind: 1},     // Intro
        {Index: 2, Beat: 17, Kind: 2},    // Verse
        {Index: 3, Beat: 65, Kind: 9},    // Chorus
        {Index: 4, Beat: 97, Kind: 10},   // Outro
    }
    
    return &SongStructureTag{
        Mood:     2,  // Mid mood
        EndBeat:  durationBeats,
        Bank:     0,  // Default
        Entries:  entries,
        IsMasked: true,
    }
}
```

### From Audio Analysis

This would require sophisticated audio analysis (beyond initial scope):

```go
func AnalyzeSongStructure(audioPath string) (*SongStructureTag, error) {
    // This would use machine learning or signal processing
    // to detect:
    // - Intro/outro sections (low energy)
    // - Verse sections (vocals, moderate energy)
    // - Chorus sections (high energy, repetition)
    // - Bridge sections (transition points)
    
    // Not implemented - placeholder only
    return GeneratePlaceholderStructure(256), nil
}
```

## Integration Points

### Optional Tag (Low Priority)

Song structure is only used by CDJ-3000 for lighting:

```go
func GenerateAnalysisFiles(track *library.Track) error {
    // .DAT file - basic tags
    datFile := &anlz.File{ /* ... */ }
    datFile.WriteToFile(track.AnalyzePath)
    
    // .2EX file - advanced features including structure
    if generateAdvanced {
        exFile := &anlz.File{
            Header: anlz.NewFileHeader(),
            Tags: []anlz.Tag{
                GeneratePlaceholderStructure(track.DurationBeats),
            },
        }
        
        exPath := strings.Replace(track.AnalyzePath, ".DAT", ".2EX", 1)
        exFile.WriteToFile(exPath)
    }
    
    return nil
}
```

## Testing Requirements

### Test Cases

1. **XOR Masking**
```go
func TestPSSI_XORMasking(t *testing.T) {
    original := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
    numEntries := uint16(5)
    
    masked := maskPSSI(original, numEntries)
    unmasked := UnmaskPSSI(masked, numEntries)
    
    assert.Equal(t, original, unmasked)
}
```

2. **Phrase Labels**
```go
func TestGetPhraseLabel(t *testing.T) {
    // High mood, kind 1, k1=1 → "Intro 1"
    label := GetPhraseLabel(1, 1, 1, 0, 0)
    assert.Equal(t, "Intro 1", label)
    
    // Mid mood, kind 9 → "Chorus"
    label = GetPhraseLabel(2, 9, 0, 0, 0)
    assert.Equal(t, "Chorus", label)
}
```

## Common Pitfalls

### ❌ Wrong Entry Size
```go
// WRONG - Entries might be different sizes
ByteOrder.PutUint32(buf[12:16], 0x20)

// CORRECT - Always 0x18 (24 bytes)
ByteOrder.PutUint32(buf[12:16], 0x18)
```

### ❌ Not Masking Exports
```go
// WRONG - Modern rekordbox expects masking
IsMasked: false

// CORRECT
IsMasked: true
```

## Performance Notes

- Structure analysis is complex (if implemented)
- Placeholder is fast (<1ms)
- XOR masking is negligible overhead
- Can skip entirely for basic functionality

## Future Enhancements

1. **Audio Analysis** - Detect actual song structure
2. **Machine Learning** - Train model on labeled data
3. **Manual Editing** - UI for adjusting phrases
4. **Auto-detection** - Recognize common patterns

## Related Files

- `anlz.go` - Uses SongStructureTag in File.Tags
- `tag.go` - SongStructureTag implements Tag interface
- `pkg/mediascanner/mediascanner.go` - Generates structure tags

## Reference

Deep Symmetry Documentation:
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#song-structure-tag

Song structure enables:
- CDJ-3000 lighting control
- Rekordbox Performance Mode
- Visual phrase markers
- Auto-remix features

**Recommendation:** Skip this tag initially. It's only needed for CDJ-3000 lighting, which is a niche feature. Implement placeholder if time permits.

