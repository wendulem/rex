# tag.go - Base Tag Interface and Common Utilities

## Purpose

This file defines the `Tag` interface that all analysis file sections must implement, along with common utilities for tag manipulation, the `TagType` enum, and helper functions for tag header management.

## Source Documentation

From: https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#analysis-file-sections

> The structure of each tagged section has an "envelope" that can be understood even if the internal structure of the section is unknown, making it easy to navigate through the file looking for the section you need.

### Tag Structure

```
Byte Offset
0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
┌──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┐
│   fourcc    │ len_header  │   len_tag   │     │ 00
├──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┤
│             Tag-specific content              │ 10
│                                                │ 20
└────────────────────────────────────────────────┘
```

Every tag has:
- **fourcc** (4 bytes): Four-character code identifying tag type
- **len_header** (4 bytes): Header length in bytes
- **len_tag** (4 bytes): Total tag length including header
- **content**: Tag-specific data

## Key Responsibilities

### 1. Tag Interface
Defines contract for all tag types:
```go
type Tag interface {
    FourCC() [4]byte                    // e.g., "PQTZ", "PPTH"
    TagType() TagType                   // Enum identifier
    MarshalBinary() ([]byte, error)     // Encode to bytes
    UnmarshalBinary([]byte) error       // Decode from bytes
}
```

### 2. TagType Enum
Type-safe identifiers for tags:
```go
type TagType int

const (
    TagTypeBeatGrid TagType = iota
    TagTypeCueList
    TagTypeCueListExtended
    TagTypePath
    TagTypeVBR
    TagTypeWaveformPreview
    TagTypeWaveformTinyPreview
    TagTypeWaveformDetail
    TagTypeWaveformColorPreview
    TagTypeWaveformColorDetail
    TagTypeWaveform3BandPreview
    TagTypeWaveform3BandDetail
    TagTypeSongStructure
)
```

### 3. Common Tag Header
Shared structure for all tags:
```go
type TagHeader struct {
    FourCC    [4]byte // Tag identifier
    HeaderLen uint32  // Length of header in bytes
    TagLen    uint32  // Total length including header
}
```

### 4. Utility Functions
```go
func MarshalTagHeader(fourCC [4]byte, headerLen, tagLen uint32) []byte
func UnmarshalTagHeader(data []byte) (*TagHeader, error)
func TagTypeFromFourCC(fourCC [4]byte) TagType
func FourCCFromTagType(t TagType) [4]byte
```

## Implementation Details

### Tag Interface

```go
// Tag represents any tagged section in an analysis file
type Tag interface {
    // FourCC returns the four-character code identifying this tag type
    // Examples: "PQTZ" (beat grid), "PPTH" (path), "PCO2" (cues)
    FourCC() [4]byte
    
    // TagType returns the enum identifier for this tag
    TagType() TagType
    
    // MarshalBinary encodes the complete tag including header
    MarshalBinary() ([]byte, error)
    
    // UnmarshalBinary decodes the complete tag including header
    UnmarshalBinary([]byte) error
}
```

### TagType Enum

```go
type TagType int

const (
    TagTypeBeatGrid              TagType = iota  // PQTZ
    TagTypeCueList                              // PCOB (old format)
    TagTypeCueListExtended                      // PCO2 (nxs2 format)
    TagTypePath                                 // PPTH
    TagTypeVBR                                  // PVBR
    TagTypeWaveformPreview                      // PWAV
    TagTypeWaveformTinyPreview                  // PWV2
    TagTypeWaveformDetail                       // PWV3
    TagTypeWaveformColorPreview                 // PWV4
    TagTypeWaveformColorDetail                  // PWV5
    TagTypeWaveform3BandPreview                 // PWV6
    TagTypeWaveform3BandDetail                  // PWV7
    TagTypeSongStructure                        // PSSI
    TagTypeUnknown                              // Unrecognized
)

func (t TagType) String() string {
    names := [...]string{
        "BeatGrid",
        "CueList",
        "CueListExtended",
        "Path",
        "VBR",
        "WaveformPreview",
        "WaveformTinyPreview",
        "WaveformDetail",
        "WaveformColorPreview",
        "WaveformColorDetail",
        "Waveform3BandPreview",
        "Waveform3BandDetail",
        "SongStructure",
        "Unknown",
    }
    
    if int(t) < len(names) {
        return names[t]
    }
    return "Unknown"
}
```

### Common Tag Header

```go
// TagHeader is the common header structure for all tags
type TagHeader struct {
    FourCC    [4]byte // Four-character code
    HeaderLen uint32  // Header length (varies by tag)
    TagLen    uint32  // Total tag length
}

// MarshalBinary encodes just the header (first 12 bytes)
func (h *TagHeader) MarshalBinary() ([]byte, error) {
    buf := make([]byte, 12)
    
    copy(buf[0:4], h.FourCC[:])
    ByteOrder.PutUint32(buf[4:8], h.HeaderLen)
    ByteOrder.PutUint32(buf[8:12], h.TagLen)
    
    return buf, nil
}

// UnmarshalBinary decodes the header
func (h *TagHeader) UnmarshalBinary(data []byte) error {
    if len(data) < 12 {
        return fmt.Errorf("header too short: %d bytes", len(data))
    }
    
    copy(h.FourCC[:], data[0:4])
    h.HeaderLen = ByteOrder.Uint32(data[4:8])
    h.TagLen = ByteOrder.Uint32(data[8:12])
    
    return nil
}
```

### Utility Functions

```go
// MarshalTagHeader creates header bytes for a tag
func MarshalTagHeader(fourCC [4]byte, headerLen, tagLen uint32) []byte {
    buf := make([]byte, 12)
    copy(buf[0:4], fourCC[:])
    ByteOrder.PutUint32(buf[4:8], headerLen)
    ByteOrder.PutUint32(buf[8:12], tagLen)
    return buf
}

// UnmarshalTagHeader reads header from bytes
func UnmarshalTagHeader(data []byte) (*TagHeader, error) {
    h := &TagHeader{}
    if err := h.UnmarshalBinary(data); err != nil {
        return nil, err
    }
    return h, nil
}

// TagTypeFromFourCC returns the TagType for a given FourCC
func TagTypeFromFourCC(fourCC [4]byte) TagType {
    switch string(fourCC[:]) {
    case "PQTZ":
        return TagTypeBeatGrid
    case "PCOB":
        return TagTypeCueList
    case "PCO2":
        return TagTypeCueListExtended
    case "PPTH":
        return TagTypePath
    case "PVBR":
        return TagTypeVBR
    case "PWAV":
        return TagTypeWaveformPreview
    case "PWV2":
        return TagTypeWaveformTinyPreview
    case "PWV3":
        return TagTypeWaveformDetail
    case "PWV4":
        return TagTypeWaveformColorPreview
    case "PWV5":
        return TagTypeWaveformColorDetail
    case "PWV6":
        return TagTypeWaveform3BandPreview
    case "PWV7":
        return TagTypeWaveform3BandDetail
    case "PSSI":
        return TagTypeSongStructure
    default:
        return TagTypeUnknown
    }
}

// FourCCFromTagType returns the FourCC for a given TagType
func FourCCFromTagType(t TagType) [4]byte {
    fourCCs := map[TagType][4]byte{
        TagTypeBeatGrid:              {'P', 'Q', 'T', 'Z'},
        TagTypeCueList:               {'P', 'C', 'O', 'B'},
        TagTypeCueListExtended:       {'P', 'C', 'O', '2'},
        TagTypePath:                  {'P', 'P', 'T', 'H'},
        TagTypeVBR:                   {'P', 'V', 'B', 'R'},
        TagTypeWaveformPreview:       {'P', 'W', 'A', 'V'},
        TagTypeWaveformTinyPreview:   {'P', 'W', 'V', '2'},
        TagTypeWaveformDetail:        {'P', 'W', 'V', '3'},
        TagTypeWaveformColorPreview:  {'P', 'W', 'V', '4'},
        TagTypeWaveformColorDetail:   {'P', 'W', 'V', '5'},
        TagTypeWaveform3BandPreview:  {'P', 'W', 'V', '6'},
        TagTypeWaveform3BandDetail:   {'P', 'W', 'V', '7'},
        TagTypeSongStructure:         {'P', 'S', 'S', 'I'},
    }
    
    if fourCC, ok := fourCCs[t]; ok {
        return fourCC
    }
    return [4]byte{0, 0, 0, 0}
}

// CreateTagForFourCC instantiates appropriate tag type
func CreateTagForFourCC(fourCC [4]byte) Tag {
    switch string(fourCC[:]) {
    case "PQTZ":
        return &BeatGridTag{}
    case "PCO2":
        return &CueListExtendedTag{}
    case "PCOB":
        return &CueListTag{}
    case "PPTH":
        return &PathTag{}
    case "PVBR":
        return &VBRTag{}
    case "PWAV":
        return &WaveformPreviewTag{}
    case "PWV2":
        return &WaveformTinyPreviewTag{}
    case "PWV3":
        return &WaveformDetailTag{}
    case "PWV4":
        return &WaveformColorPreviewTag{}
    case "PWV5":
        return &WaveformColorDetailTag{}
    case "PWV6":
        return &Waveform3BandPreviewTag{}
    case "PWV7":
        return &Waveform3BandDetailTag{}
    case "PSSI":
        return &SongStructureTag{}
    default:
        return nil
    }
}
```

### Tag Implementation Helper

Common pattern for implementing tags:

```go
// Example: Every tag implementation should follow this pattern
type ExampleTag struct {
    // Tag-specific fields
    Data string
}

func (t *ExampleTag) FourCC() [4]byte {
    return [4]byte{'E', 'X', 'A', 'M'}
}

func (t *ExampleTag) TagType() TagType {
    return TagTypeExample
}

func (t *ExampleTag) MarshalBinary() ([]byte, error) {
    // 1. Encode tag-specific data
    dataBytes := []byte(t.Data)
    
    // 2. Calculate lengths
    headerLen := uint32(0x10)  // Tag-specific
    dataLen := uint32(len(dataBytes))
    tagLen := headerLen + dataLen
    
    // 3. Allocate buffer
    buf := make([]byte, tagLen)
    
    // 4. Write header
    copy(buf[0:4], t.FourCC()[:])
    ByteOrder.PutUint32(buf[4:8], headerLen)
    ByteOrder.PutUint32(buf[8:12], tagLen)
    
    // 5. Write tag-specific header fields (12 to headerLen)
    // ...
    
    // 6. Write data
    copy(buf[headerLen:], dataBytes)
    
    return buf, nil
}

func (t *ExampleTag) UnmarshalBinary(data []byte) error {
    // 1. Parse header
    header := &TagHeader{}
    if err := header.UnmarshalBinary(data); err != nil {
        return err
    }
    
    // 2. Verify FourCC
    if header.FourCC != t.FourCC() {
        return fmt.Errorf("wrong fourcc: %s", header.FourCC)
    }
    
    // 3. Parse tag-specific header fields
    // ...
    
    // 4. Parse data
    dataOffset := int(header.HeaderLen)
    t.Data = string(data[dataOffset:header.TagLen])
    
    return nil
}
```

## Integration Points

### Used By
- All tag implementations (`beatgrid.go`, `cue.go`, `path.go`, etc.)
- `anlz.go` for tag type dispatch

### Calls
- None (base interface and utilities)

### Dependencies
```go
import (
    "encoding/binary"
    "fmt"
)
```

## Testing Requirements

### Test Cases

1. **TagType String Conversion**
```go
func TestTagTypeString(t *testing.T) {
    assert.Equal(t, "BeatGrid", TagTypeBeatGrid.String())
    assert.Equal(t, "Path", TagTypePath.String())
}
```

2. **FourCC Conversion**
```go
func TestTagTypeFromFourCC(t *testing.T) {
    fourCC := [4]byte{'P', 'Q', 'T', 'Z'}
    tagType := TagTypeFromFourCC(fourCC)
    assert.Equal(t, TagTypeBeatGrid, tagType)
}

func TestFourCCFromTagType(t *testing.T) {
    fourCC := FourCCFromTagType(TagTypeBeatGrid)
    assert.Equal(t, [4]byte{'P', 'Q', 'T', 'Z'}, fourCC)
}
```

3. **Tag Header Marshal/Unmarshal**
```go
func TestTagHeaderRoundTrip(t *testing.T) {
    original := &TagHeader{
        FourCC:    [4]byte{'T', 'E', 'S', 'T'},
        HeaderLen: 0x18,
        TagLen:    0x100,
    }
    
    data, err := original.MarshalBinary()
    assert.NoError(t, err)
    
    loaded := &TagHeader{}
    err = loaded.UnmarshalBinary(data)
    assert.NoError(t, err)
    assert.Equal(t, original, loaded)
}
```

4. **Create Tag Factory**
```go
func TestCreateTagForFourCC(t *testing.T) {
    tag := CreateTagForFourCC([4]byte{'P', 'Q', 'T', 'Z'})
    assert.NotNil(t, tag)
    assert.Equal(t, TagTypeBeatGrid, tag.TagType())
    
    // Unknown tag
    tag = CreateTagForFourCC([4]byte{'X', 'X', 'X', 'X'})
    assert.Nil(t, tag)
}
```

## Usage Examples

### Implementing a New Tag

```go
type MyCustomTag struct {
    Value uint32
}

func (t *MyCustomTag) FourCC() [4]byte {
    return [4]byte{'M', 'Y', 'T', 'G'}
}

func (t *MyCustomTag) TagType() TagType {
    return TagTypeCustom
}

func (t *MyCustomTag) MarshalBinary() ([]byte, error) {
    headerLen := uint32(0x10)
    tagLen := headerLen + 4 // 4 bytes for uint32
    
    buf := make([]byte, tagLen)
    copy(buf[0:4], t.FourCC()[:])
    ByteOrder.PutUint32(buf[4:8], headerLen)
    ByteOrder.PutUint32(buf[8:12], tagLen)
    // Header padding (12-16)
    ByteOrder.PutUint32(buf[16:20], t.Value)
    
    return buf, nil
}

func (t *MyCustomTag) UnmarshalBinary(data []byte) error {
    header := &TagHeader{}
    if err := header.UnmarshalBinary(data); err != nil {
        return err
    }
    
    t.Value = ByteOrder.Uint32(data[16:20])
    return nil
}
```

### Using Tag Interface

```go
func ProcessTag(tag Tag) {
    fmt.Printf("Processing %s tag\n", tag.TagType())
    
    data, err := tag.MarshalBinary()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Tag size: %d bytes\n", len(data))
}
```

### Tag Dispatch

```go
func HandleTag(tag Tag) {
    switch tag.TagType() {
    case TagTypeBeatGrid:
        beatGrid := tag.(*BeatGridTag)
        fmt.Printf("Beat grid with %d beats\n", len(beatGrid.Beats))
        
    case TagTypePath:
        path := tag.(*PathTag)
        fmt.Printf("Path: %s\n", path.Path)
        
    default:
        fmt.Printf("Unknown tag type: %s\n", tag.TagType())
    }
}
```

## Common Pitfalls

### ❌ Wrong Header Length
```go
// WRONG - Not including header in total length
tagLen := uint32(len(data))

// CORRECT - Total includes header
tagLen := headerLen + uint32(len(data))
```

### ❌ Forgetting Big-Endian
```go
// WRONG
binary.LittleEndian.PutUint32(buf[4:8], headerLen)

// CORRECT
ByteOrder.PutUint32(buf[4:8], headerLen)
```

### ❌ Not Verifying FourCC on Unmarshal
```go
// WRONG - Blindly parsing
func (t *MyTag) UnmarshalBinary(data []byte) error {
    // Parse data...
}

// CORRECT - Verify tag type
func (t *MyTag) UnmarshalBinary(data []byte) error {
    header := &TagHeader{}
    header.UnmarshalBinary(data)
    
    if header.FourCC != t.FourCC() {
        return fmt.Errorf("wrong tag type")
    }
    // Parse data...
}
```

## Performance Notes

- Tag interface is called frequently during file operations
- Header parsing is cheap (12 bytes)
- Tag factory uses string comparison (fast enough)
- Consider caching FourCC → TagType mappings if needed

## Future Enhancements

1. **Tag Validation**
```go
type Tag interface {
    // ... existing methods
    Validate() error
}
```

2. **Tag Priority**
```go
func (t TagType) Priority() int {
    // Path and BeatGrid are critical
    // Waveforms are optional
}
```

3. **Tag Dependencies**
```go
func (t TagType) RequiredTags() []TagType {
    // Some tags might depend on others
}
```

## Related Files

- `anlz.go` - Uses Tag interface for file operations
- `beatgrid.go` - Implements Tag for beat grids
- `cue.go` - Implements Tag for cue points
- `path.go` - Implements Tag for paths
- `waveform.go` - Implements Tag for waveforms

## Reference

Deep Symmetry Documentation:
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#analysis-file-sections

Common envelope structure described in section "Tagged section structure".

