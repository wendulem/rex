# cue.go - PCOB/PCO2 Cue Point Tags

## Purpose

This file implements cue point and loop tags for analysis files. There are two formats:
- **PCOB** - Original cue list format (up to 3 hot cues)
- **PCO2** - Extended format for nxs2 players (8+ hot cues, colors, comments)

We'll focus on **PCO2** as it's the modern standard and backwards compatible.

## Source Documentation

From: https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#extended-nxs2-cue-list-tag

> This is a variation of the Cue List Tag that was introduced with the Nexus 2 players to add support for more than three hot cues with custom color assignments, as well as DJ-assigned comment text for each hot cue and memory point.

### PCO2 Tag Structure

```
Byte Offset
0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
┌──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┐
│P │C │O │2 │ len_header  │   len_tag   │type │ 00
├──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┤
│   lencues   │ 0000│      Cue entries          │ 10
│ (variable length each)                        │ 20
└────────────────────────────────────────────────┘
```

### PCP2 Cue Entry Structure

```
Byte Offset
0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
┌──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┐
│P │C │P │2 │ len_header  │  len_entry  │hcue │ 00
├──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┤
│t│  unknown1   │    time     │ loop_time │cid│ 10
├──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┤
│  unknown2   │lnum │lden │len_comment│comment│ 20
│  (UTF-16BE with trailing NUL)                │ 30
│c│ r│ g│ b│
└──┴──┴──┴──┘
```

### Field Descriptions

**PCO2 Tag:**
| Offset | Size | Name | Description |
|--------|------|------|-------------|
| 0x00 | 4 | FourCC | "PCO2" identifier |
| 0x04 | 4 | HeaderLen | Always 0x14 (20 bytes) |
| 0x08 | 4 | TagLen | Total length (header + all entries) |
| 0x0c | 4 | Type | 0 = memory points, 1 = hot cues |
| 0x10 | 2 | NumEntries | Number of cue entries |
| 0x12 | 2 | Unknown | Usually 0000 |

**PCP2 Entry:**
| Offset | Size | Name | Description |
|--------|------|------|-------------|
| 0x00 | 4 | FourCC | "PCP2" identifier |
| 0x04 | 4 | HeaderLen | Always 0x10 (16 bytes) |
| 0x08 | 4 | EntryLen | Total entry length (variable) |
| 0x0c | 4 | HotCueNumber | 0 = memory point, 1-8 = hot cue A-H |
| 0x10 | 1 | Type | 1 = cue point, 2 = loop |
| 0x11 | 3 | Unknown1 | Usually 0x0003e8 (1000 decimal) |
| 0x14 | 4 | Time | Position in milliseconds |
| 0x18 | 4 | LoopTime | End position for loops (0 for cue points) |
| 0x1c | 1 | ColorID | Color table ID (for memory points/loops) |
| 0x1d | 7 | Unknown2 | Usually 0x01 followed by zeros |
| 0x24 | 2 | LoopNumerator | Quantized loop size numerator |
| 0x26 | 2 | LoopDenominator | Quantized loop size denominator |
| 0x28 | 4 | CommentLen | Length of comment in bytes |
| 0x2c | var | Comment | UTF-16BE string with trailing NUL |
| +0 | 1 | ColorCode | Hot cue color code (0-3E) |
| +1 | 1 | ColorRed | RGB red component |
| +2 | 1 | ColorGreen | RGB green component |
| +3 | 1 | ColorBlue | RGB blue component |

**Key Points:**
- Entry length is variable due to comment string
- Hot cues use ColorCode/RGB, memory points use ColorID
- Comment is UTF-16BE with trailing NUL
- Some entries may be incomplete (missing comment/color)

## Implementation

### CueListExtendedTag Struct

```go
type CueListExtendedTag struct {
    Type    uint32     // 0 = memory points, 1 = hot cues
    Entries []CueEntry
}

type CueEntry struct {
    // PCP2 header
    HotCueNumber uint32  // 0 for memory point, 1-8 for hot cues
    
    // Entry data
    Type             uint8   // 1 = cue, 2 = loop
    Time             uint32  // Position in ms
    LoopTime         uint32  // End position for loops
    ColorID          uint8   // For memory points/loops
    LoopNumerator    uint16  // Quantized loop fraction
    LoopDenominator  uint16
    Comment          string  // Optional comment
    
    // Hot cue colors
    ColorCode        uint8   // Rekordbox color code
    ColorRed         uint8   // RGB components
    ColorGreen       uint8
    ColorBlue        uint8
}
```

### Interface Implementation

```go
func (t *CueListExtendedTag) FourCC() [4]byte {
    return [4]byte{'P', 'C', 'O', '2'}
}

func (t *CueListExtendedTag) TagType() TagType {
    return TagTypeCueListExtended
}
```

### Marshal (Encode to Bytes)

```go
func (t *CueListExtendedTag) MarshalBinary() ([]byte, error) {
    // Marshal all entries first to calculate sizes
    entryData := make([][]byte, len(t.Entries))
    totalEntryLen := uint32(0)
    
    for i, entry := range t.Entries {
        data, err := entry.MarshalBinary()
        if err != nil {
            return nil, fmt.Errorf("marshal entry %d: %w", i, err)
        }
        entryData[i] = data
        totalEntryLen += uint32(len(data))
    }
    
    // Calculate tag size
    headerLen := uint32(0x14)
    tagLen := headerLen + totalEntryLen
    
    buf := make([]byte, tagLen)
    
    // Write PCO2 header
    copy(buf[0:4], t.FourCC()[:])
    ByteOrder.PutUint32(buf[4:8], headerLen)
    ByteOrder.PutUint32(buf[8:12], tagLen)
    ByteOrder.PutUint32(buf[12:16], t.Type)
    ByteOrder.PutUint16(buf[16:18], uint16(len(t.Entries)))
    ByteOrder.PutUint16(buf[18:20], 0)  // Unknown
    
    // Write entries
    offset := 20
    for _, data := range entryData {
        copy(buf[offset:], data)
        offset += len(data)
    }
    
    return buf, nil
}

func (e *CueEntry) MarshalBinary() ([]byte, error) {
    // Encode comment
    commentBytes, err := encodeUTF16BE(e.Comment)
    if err != nil {
        return nil, err
    }
    commentLen := uint32(len(commentBytes))
    
    // Calculate entry size
    headerLen := uint32(0x10)
    dataLen := uint32(0x1c) // Up to comment length field
    entryLen := headerLen + dataLen + commentLen + 4 // +4 for color bytes
    
    buf := make([]byte, entryLen)
    
    // Write PCP2 mini-header
    copy(buf[0:4], []byte{'P', 'C', 'P', '2'})
    ByteOrder.PutUint32(buf[4:8], headerLen)
    ByteOrder.PutUint32(buf[8:12], entryLen)
    ByteOrder.PutUint32(buf[12:16], e.HotCueNumber)
    
    // Write entry data
    buf[16] = e.Type
    buf[17] = 0x00
    buf[18] = 0x03
    buf[19] = 0xe8  // Unknown1 = 0x0003e8 = 1000
    
    ByteOrder.PutUint32(buf[20:24], e.Time)
    ByteOrder.PutUint32(buf[24:28], e.LoopTime)
    buf[28] = e.ColorID
    
    // Unknown2: 0x01 followed by zeros
    buf[29] = 0x01
    // buf[30:36] already zero
    
    ByteOrder.PutUint16(buf[36:38], e.LoopNumerator)
    ByteOrder.PutUint16(buf[38:40], e.LoopDenominator)
    ByteOrder.PutUint32(buf[40:44], commentLen)
    
    // Write comment
    copy(buf[44:44+commentLen], commentBytes)
    
    // Write colors
    colorOffset := 44 + int(commentLen)
    buf[colorOffset] = e.ColorCode
    buf[colorOffset+1] = e.ColorRed
    buf[colorOffset+2] = e.ColorGreen
    buf[colorOffset+3] = e.ColorBlue
    
    return buf, nil
}
```

### Unmarshal (Decode from Bytes)

```go
func (t *CueListExtendedTag) UnmarshalBinary(data []byte) error {
    // Parse header
    header := &TagHeader{}
    if err := header.UnmarshalBinary(data); err != nil {
        return err
    }
    
    if header.FourCC != t.FourCC() {
        return fmt.Errorf("wrong fourcc: %s", header.FourCC)
    }
    
    // Read type and count
    t.Type = ByteOrder.Uint32(data[12:16])
    numEntries := ByteOrder.Uint16(data[16:18])
    
    // Parse entries
    t.Entries = make([]CueEntry, 0, numEntries)
    offset := 20
    
    for i := 0; i < int(numEntries) && offset < len(data); i++ {
        entry := CueEntry{}
        
        // Read entry length
        if offset+12 > len(data) {
            break
        }
        entryLen := ByteOrder.Uint32(data[offset+8 : offset+12])
        
        // Unmarshal entry
        if err := entry.UnmarshalBinary(data[offset : offset+int(entryLen)]); err != nil {
            return fmt.Errorf("unmarshal entry %d: %w", i, err)
        }
        
        t.Entries = append(t.Entries, entry)
        offset += int(entryLen)
    }
    
    return nil
}

func (e *CueEntry) UnmarshalBinary(data []byte) error {
    if len(data) < 44 {
        return fmt.Errorf("entry too short: %d bytes", len(data))
    }
    
    // Read header
    e.HotCueNumber = ByteOrder.Uint32(data[12:16])
    
    // Read data
    e.Type = data[16]
    e.Time = ByteOrder.Uint32(data[20:24])
    e.LoopTime = ByteOrder.Uint32(data[24:28])
    e.ColorID = data[28]
    e.LoopNumerator = ByteOrder.Uint16(data[36:38])
    e.LoopDenominator = ByteOrder.Uint16(data[38:40])
    
    // Read comment if present
    commentLen := ByteOrder.Uint32(data[40:44])
    if commentLen > 0 && len(data) >= 44+int(commentLen) {
        commentBytes := data[44 : 44+commentLen]
        comment, err := decodeUTF16BE(commentBytes)
        if err != nil {
            return fmt.Errorf("decode comment: %w", err)
        }
        e.Comment = comment
    }
    
    // Read colors if present
    colorOffset := 44 + int(commentLen)
    if len(data) >= colorOffset+4 {
        e.ColorCode = data[colorOffset]
        e.ColorRed = data[colorOffset+1]
        e.ColorGreen = data[colorOffset+2]
        e.ColorBlue = data[colorOffset+3]
    }
    
    return nil
}
```

## Cue Generation from Mixxx

### Query for Cues

Add to `pkg/mixxx/query.sql`:

```sql
-- name: GetTrackCues :many
SELECT 
    hotcue,
    type,
    position,
    length,
    label,
    color
FROM cues
WHERE track_id = ?
ORDER BY hotcue;
```

### Convert Mixxx Cues

```go
func GenerateCuesFromMixxx(mixxxCues []mixxx.Cue) *anlz.CueListExtendedTag {
    entries := make([]anlz.CueEntry, 0, len(mixxxCues))
    
    for _, mc := range mixxxCues {
        entry := anlz.CueEntry{
            HotCueNumber: uint32(mc.Hotcue),
            Time:         uint32(mc.Position * 1000), // Convert to ms
            Comment:      mc.Label.String,
        }
        
        // Determine type
        if mc.Length.Valid && mc.Length.Float64 > 0 {
            entry.Type = 2  // Loop
            entry.LoopTime = uint32((mc.Position + mc.Length.Float64) * 1000)
        } else {
            entry.Type = 1  // Cue point
        }
        
        // Parse color
        if mc.Color.Valid {
            r, g, b := parseColor(mc.Color.Int64)
            entry.ColorRed = uint8(r)
            entry.ColorGreen = uint8(g)
            entry.ColorBlue = uint8(b)
            entry.ColorCode = colorToRekordboxCode(r, g, b)
        }
        
        entries = append(entries, entry)
    }
    
    return &anlz.CueListExtendedTag{
        Type:    1,  // Hot cues
        Entries: entries,
    }
}
```

## Color Handling

### Rekordbox Color Codes

From documentation, color codes 0x00-0x3E map to rekordbox's hot cue palette.

```go
// Default colors for hot cues if not specified
var DefaultHotCueColors = []RGB{
    {255, 0, 0},     // A - Red
    {255, 165, 0},   // B - Orange
    {255, 255, 0},   // C - Yellow
    {0, 255, 0},     // D - Green
    {0, 255, 255},   // E - Cyan
    {0, 0, 255},     // F - Blue
    {255, 0, 255},   // G - Magenta
    {255, 255, 255}, // H - White
}

func colorToRekordboxCode(r, g, b uint8) uint8 {
    // Find closest match in rekordbox palette
    // For now, use simple mapping
    if r > 200 && g < 100 && b < 100 {
        return 0x01  // Red
    }
    // ... more mappings
    return 0x00  // Default green
}
```

## Testing Requirements

### Test Cases

1. **Simple Hot Cue**
```go
func TestCueListExtendedTag_SimpleHotCue(t *testing.T) {
    tag := &CueListExtendedTag{
        Type: 1,  // Hot cues
        Entries: []CueEntry{
            {
                HotCueNumber: 1,  // Hot Cue A
                Type:         1,  // Cue point
                Time:         5000,  // 5 seconds
                Comment:      "Intro",
                ColorCode:    0x01,
                ColorRed:     255,
                ColorGreen:   0,
                ColorBlue:    0,
            },
        },
    }
    
    data, err := tag.MarshalBinary()
    assert.NoError(t, err)
    
    loaded := &CueListExtendedTag{}
    err = loaded.UnmarshalBinary(data)
    assert.NoError(t, err)
    assert.Equal(t, tag.Entries[0].Comment, loaded.Entries[0].Comment)
}
```

2. **Loop Entry**
```go
func TestCueEntry_Loop(t *testing.T) {
    entry := CueEntry{
        HotCueNumber: 2,
        Type:         2,  // Loop
        Time:         10000,
        LoopTime:     14000,  // 4-second loop
        LoopNumerator: 4,
        LoopDenominator: 1,
    }
    
    data, _ := entry.MarshalBinary()
    loaded := CueEntry{}
    loaded.UnmarshalBinary(data)
    
    assert.Equal(t, uint8(2), loaded.Type)
    assert.Equal(t, uint32(14000), loaded.LoopTime)
}
```

3. **Unicode Comment**
```go
func TestCueEntry_UnicodeComment(t *testing.T) {
    entry := CueEntry{
        HotCueNumber: 1,
        Type:         1,
        Time:         0,
        Comment:      "ドロップ",  // Japanese
    }
    
    data, _ := entry.MarshalBinary()
    loaded := CueEntry{}
    loaded.UnmarshalBinary(data)
    
    assert.Equal(t, entry.Comment, loaded.Comment)
}
```

## Common Pitfalls

### ❌ Wrong Type Value
```go
// WRONG - Type is 0 or 1
Type: 2

// CORRECT - 0 = memory points, 1 = hot cues
Type: 1
```

### ❌ Missing Unknown1 Value
```go
// WRONG - Should be 0x0003e8
buf[17] = 0x00
buf[18] = 0x00
buf[19] = 0x00

// CORRECT
buf[17] = 0x00
buf[18] = 0x03
buf[19] = 0xe8
```

### ❌ Wrong Hot Cue Numbers
```go
// WRONG - Hot cues are 1-8, not 0-7
HotCueNumber: 0  // Should be 1 for Hot Cue A

// CORRECT
HotCueNumber: 1  // Hot Cue A
```

## Future Enhancements

1. **Auto-cue Generation** - Detect intro/outro points
2. **Loop Detection** - Find natural loop points
3. **Color Palette** - Full rekordbox color mapping
4. **Phrase-based Cues** - Generate from song structure

## Related Files

- `anlz.go` - Uses CueListExtendedTag in File.Tags
- `tag.go` - CueListExtendedTag implements Tag interface
- `pkg/mixxx/query.sql` - Queries cue data
- `pkg/mediascanner/mediascanner.go` - Generates cue tags

## Reference

Deep Symmetry Documentation:
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#extended-nxs2-cue-list-tag
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#cue-list-tag (original PCOB format)

