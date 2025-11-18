# anlz.go - Core Analysis File I/O

## Purpose

This file provides the core functionality for reading and writing Pioneer DJ analysis files. It defines the main `File` struct that orchestrates all tags and handles file-level operations.

## Source Documentation

From: https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#analysis-files

> Analysis files are "tagged type" files, where there is an overall file header section, and then each entry in the file has its own header which identifies the type and length of that section.

The file format consists of:
1. PMAI file header (28 bytes)
2. Series of tagged sections (variable length)

## Key Responsibilities

### 1. File struct
The main container for an analysis file:
```go
type File struct {
    Header *FileHeader  // PMAI header (28 bytes)
    Tags   []Tag        // All tagged sections
}
```

### 2. Tag Interface
Defines the contract all tag types must implement:
```go
type Tag interface {
    FourCC() [4]byte           // Tag identifier (e.g., "PQTZ", "PPTH")
    TagType() TagType          // Enum for type identification
    MarshalBinary() ([]byte, error)
    UnmarshalBinary([]byte) error
}
```

### 3. File Writing
Marshal all components and write to disk:
```go
func (f *File) WriteToFile(path string) error {
    // 1. Marshal all tags
    // 2. Calculate total file size
    // 3. Update header with size
    // 4. Write header + all tags
}
```

### 4. File Reading
Load and parse existing analysis files (useful for testing/validation):
```go
func LoadFromFile(path string) (*File, error) {
    // 1. Read and parse header
    // 2. Read tagged sections
    // 3. Identify and unmarshal each tag
}
```

### 5. Byte Order Constant
**Critical:** Analysis files use big-endian (opposite of PDB):
```go
var ByteOrder = binary.BigEndian
```

## Implementation Details

### File Structure

```
┌──────────────────────────────────────┐
│ File                                 │
│ ┌──────────────────────────────────┐ │
│ │ Header (FileHeader)              │ │
│ │  - Magic: [4]byte "PMAI"         │ │
│ │  - HeaderLen: uint32 (0x1c)      │ │
│ │  - FileLen: uint32 (total)       │ │
│ │  - Unknown: [16]byte             │ │
│ └──────────────────────────────────┘ │
│ ┌──────────────────────────────────┐ │
│ │ Tags: []Tag                      │ │
│ │  [0] PathTag                     │ │
│ │  [1] BeatGridTag                 │ │
│ │  [2] CueListExtendedTag          │ │
│ │  [3] WaveformPreviewTag          │ │
│ │  ...                             │ │
│ └──────────────────────────────────┘ │
└──────────────────────────────────────┘
```

### Tag Length Calculation

Each tag must calculate its total length including header:
```go
// In each tag's MarshalBinary()
headerLen := uint32(0x18)  // Tag-specific
dataLen := uint32(len(data))
totalLen := headerLen + dataLen

buf := make([]byte, totalLen)
copy(buf[0:4], fourCC[:])
ByteOrder.PutUint32(buf[4:8], headerLen)
ByteOrder.PutUint32(buf[8:12], totalLen)
copy(buf[headerLen:], data)
```

### File Length Calculation

The PMAI header stores the complete file size:
```go
func (f *File) calculateFileLength() uint32 {
    total := uint32(28) // Header size
    
    for _, tag := range f.Tags {
        data, _ := tag.MarshalBinary()
        total += uint32(len(data))
    }
    
    return total
}
```

### Writing Process

```go
func (f *File) WriteToFile(path string) error {
    // 1. Marshal all tags first (to calculate sizes)
    tagData := make([][]byte, len(f.Tags))
    totalSize := uint32(28)
    
    for i, tag := range f.Tags {
        data, err := tag.MarshalBinary()
        if err != nil {
            return fmt.Errorf("marshal tag %d: %w", i, err)
        }
        tagData[i] = data
        totalSize += uint32(len(data))
    }
    
    // 2. Update header with total size
    f.Header.FileLen = totalSize
    
    // 3. Marshal header
    headerBytes, err := f.Header.MarshalBinary()
    if err != nil {
        return fmt.Errorf("marshal header: %w", err)
    }
    
    // 4. Write to file
    file, err := os.Create(path)
    if err != nil {
        return err
    }
    defer file.Close()
    
    // Write header
    if _, err := file.Write(headerBytes); err != nil {
        return err
    }
    
    // Write all tags
    for i, data := range tagData {
        if _, err := file.Write(data); err != nil {
            return fmt.Errorf("write tag %d: %w", i, err)
        }
    }
    
    return nil
}
```

### Reading Process

```go
func LoadFromFile(path string) (*File, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    // Parse header
    if len(data) < 28 {
        return nil, errors.New("file too short for header")
    }
    
    header := &FileHeader{}
    if err := header.UnmarshalBinary(data[0:28]); err != nil {
        return nil, err
    }
    
    // Parse tags
    offset := 28
    tags := []Tag{}
    
    for offset < len(data) {
        // Read FourCC
        fourCC := [4]byte{data[offset], data[offset+1], data[offset+2], data[offset+3]}
        
        // Read tag length
        tagLen := ByteOrder.Uint32(data[offset+8 : offset+12])
        
        // Create appropriate tag type
        tag := createTagForFourCC(fourCC)
        if tag == nil {
            // Unknown tag, skip it
            offset += int(tagLen)
            continue
        }
        
        // Unmarshal tag
        if err := tag.UnmarshalBinary(data[offset : offset+int(tagLen)]); err != nil {
            return nil, fmt.Errorf("unmarshal tag %s: %w", fourCC, err)
        }
        
        tags = append(tags, tag)
        offset += int(tagLen)
    }
    
    return &File{Header: header, Tags: tags}, nil
}
```

### Tag Type Registry

Helper to create tags from FourCC codes:
```go
func createTagForFourCC(fourCC [4]byte) Tag {
    switch string(fourCC[:]) {
    case "PQTZ":
        return &BeatGridTag{}
    case "PPTH":
        return &PathTag{}
    case "PCO2":
        return &CueListExtendedTag{}
    case "PCOB":
        return &CueListTag{}
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
    case "PVBR":
        return &VBRTag{}
    case "PSSI":
        return &SongStructureTag{}
    default:
        return nil // Unknown tag
    }
}
```

## Integration Points

### Called By
- `pkg/mediascanner/mediascanner.go::GenerateAnalysisFiles()`
- `cmd/rex/main.go` (indirectly through mediascanner)

### Calls
- All tag implementations (`beatgrid.go`, `cue.go`, `path.go`, etc.)
- `header.go::FileHeader.MarshalBinary()`

### Dependencies
```go
import (
    "encoding/binary"
    "errors"
    "fmt"
    "io"
    "os"
)
```

## Error Handling

### File Creation Errors
```go
if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
    return fmt.Errorf("create directory: %w", err)
}
```

### Tag Marshal Errors
```go
for i, tag := range f.Tags {
    if data, err := tag.MarshalBinary(); err != nil {
        return fmt.Errorf("marshal tag %d (%s): %w", i, tag.FourCC(), err)
    }
}
```

### Graceful Unknown Tags
When reading, skip unknown tags instead of failing:
```go
if tag := createTagForFourCC(fourCC); tag == nil {
    log.Printf("Skipping unknown tag: %s", fourCC)
    offset += int(tagLen)
    continue
}
```

## Testing Requirements

### Test Cases

1. **Empty File**
```go
func TestEmptyFile(t *testing.T) {
    file := &File{
        Header: NewFileHeader(),
        Tags:   []Tag{},
    }
    // Should write header only, length=28
}
```

2. **Single Tag**
```go
func TestSingleTag(t *testing.T) {
    file := &File{
        Header: NewFileHeader(),
        Tags: []Tag{
            &PathTag{Path: "/path/to/track.mp3"},
        },
    }
    // Verify correct file length calculation
}
```

3. **Multiple Tags**
```go
func TestMultipleTags(t *testing.T) {
    file := &File{
        Header: NewFileHeader(),
        Tags: []Tag{
            &PathTag{Path: "/path"},
            &BeatGridTag{Beats: []Beat{{1, 12500, 0}}},
        },
    }
    // Verify tags written in order
}
```

4. **Round-trip**
```go
func TestRoundTrip(t *testing.T) {
    original := createTestFile()
    tmpPath := t.TempDir() + "/test.dat"
    
    original.WriteToFile(tmpPath)
    loaded, err := LoadFromFile(tmpPath)
    
    assert.NoError(t, err)
    assert.Equal(t, len(original.Tags), len(loaded.Tags))
}
```

5. **Byte Order**
```go
func TestBigEndian(t *testing.T) {
    file := &File{Header: NewFileHeader(), Tags: []Tag{}}
    file.WriteToFile("/tmp/test.dat")
    
    data, _ := os.ReadFile("/tmp/test.dat")
    
    // Verify file length at bytes 8-11 is big-endian
    fileLen := binary.BigEndian.Uint32(data[8:12])
    assert.Equal(t, uint32(28), fileLen)
}
```

## Usage Examples

### Basic Creation
```go
file := &anlz.File{
    Header: anlz.NewFileHeader(),
    Tags:   []anlz.Tag{},
}

file.Tags = append(file.Tags, &anlz.PathTag{
    Path: "/B/Contents/0123/track.mp3",
})

file.Tags = append(file.Tags, &anlz.BeatGridTag{
    Beats: generateBeats(track),
})

err := file.WriteToFile("/path/to/ANLZ0000.DAT")
```

### Reading Existing File
```go
file, err := anlz.LoadFromFile("/path/to/ANLZ0000.DAT")
if err != nil {
    log.Fatal(err)
}

// Find specific tag
for _, tag := range file.Tags {
    if tag.TagType() == anlz.TagTypeBeatGrid {
        beatGrid := tag.(*anlz.BeatGridTag)
        fmt.Printf("Track has %d beats\n", len(beatGrid.Beats))
    }
}
```

### Validation
```go
func ValidateAnalysisFile(path string) error {
    file, err := anlz.LoadFromFile(path)
    if err != nil {
        return err
    }
    
    // Check required tags
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
        return errors.New("missing PPTH tag")
    }
    if !hasBeatGrid {
        return errors.New("missing PQTZ tag")
    }
    
    return nil
}
```

## Performance Considerations

### Memory Usage
- Keep entire file in memory (typically < 1 MB)
- Pre-allocate tag data slices
- Reuse buffers where possible

### File I/O
- Write atomically (create temp file, rename)
- Buffer writes for efficiency
- Validate before writing

### Tag Ordering
While order doesn't matter functionally, match rekordbox for compatibility:
1. PPTH (path)
2. PQTZ (beat grid)
3. PCO2 (cues)
4. PWAV (waveform preview)
5. PWV2 (tiny preview)
6. PVBR (VBR index)

## Common Pitfalls

### ❌ Wrong Byte Order
```go
// WRONG - This is for PDB files
binary.LittleEndian.PutUint32(buf, length)

// CORRECT - Analysis files are big-endian
anlz.ByteOrder.PutUint32(buf, length)
```

### ❌ Incorrect Length Calculation
```go
// WRONG - Missing header in length
tagLen := uint32(len(data))

// CORRECT - Include header in total
tagLen := headerLen + uint32(len(data))
```

### ❌ Forgetting Trailing NUL in Strings
```go
// WRONG - Missing terminator
bytes := utf16Encode(str)

// CORRECT - Add NUL terminator
bytes := utf16EncodeWithNUL(str)
```

## Future Enhancements

1. **Streaming Write** - For very large waveforms
2. **Incremental Updates** - Modify existing files
3. **Validation** - Verify tag integrity
4. **Compression** - Reduce file sizes
5. **Parallel Processing** - Generate multiple files concurrently

## Related Files

- `header.go` - PMAI file header structure
- `tag.go` - Tag interface and common utilities
- `beatgrid.go` - Beat grid tag implementation
- `path.go` - Path tag implementation
- `cue.go` - Cue tag implementations
- `waveform.go` - Waveform tag implementations

## Reference

Deep Symmetry Documentation:
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#analysis-files
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#analysis-file-header
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#analysis-file-sections

