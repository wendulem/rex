# path.go - PPTH Path Tag

## Purpose

This file implements the PPTH (Path) tag, which stores the file path to the audio file on the media. This is the simplest but one of the most critical tags - without it, CDJs cannot locate the audio file to play.

## Source Documentation

From: https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#path-tag

> This kind of section holds the file path of the audio file for which the track analysis was performed. It is identified by the four-character code PPTH

### Tag Structure

```
Byte Offset
0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
┌──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┐
│P │P │T │H │ len_header  │   len_tag   │     │ 00
├──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┤
│  len_path   │          path (UTF-16BE)        │ 10
│              with trailing NUL 0000            │ 20
└────────────────────────────────────────────────┘
```

### Field Descriptions

| Offset | Size | Name | Description |
|--------|------|------|-------------|
| 0x00 | 4 | FourCC | "PPTH" identifier |
| 0x04 | 4 | HeaderLen | Always 0x10 (16 bytes) |
| 0x08 | 4 | TagLen | Total length including path string |
| 0x0c | 4 | PathLen | Length of path string in bytes (including trailing NUL) |
| 0x10 | variable | Path | UTF-16 Big-Endian string with trailing 0x0000 |

**Key Points:**
- Path is encoded as **UTF-16 Big-Endian**
- Must have trailing NUL (0x0000)
- PathLen includes the trailing NUL
- Path is relative to the USB media root (e.g., `/B/Contents/...`)

## Implementation

### PathTag Struct

```go
type PathTag struct {
    Path string  // File path on media (e.g., "/B/Contents/track.mp3")
}
```

### Interface Implementation

```go
func (t *PathTag) FourCC() [4]byte {
    return [4]byte{'P', 'P', 'T', 'H'}
}

func (t *PathTag) TagType() TagType {
    return TagTypePath
}
```

### Marshal (Encode to Bytes)

```go
func (t *PathTag) MarshalBinary() ([]byte, error) {
    // Encode path as UTF-16 Big-Endian with trailing NUL
    pathBytes, err := encodeUTF16BE(t.Path)
    if err != nil {
        return nil, fmt.Errorf("encode path: %w", err)
    }
    
    headerLen := uint32(0x10)
    pathLen := uint32(len(pathBytes))
    tagLen := headerLen + pathLen
    
    buf := make([]byte, tagLen)
    
    // Write tag header
    copy(buf[0:4], t.FourCC()[:])
    ByteOrder.PutUint32(buf[4:8], headerLen)
    ByteOrder.PutUint32(buf[8:12], tagLen)
    ByteOrder.PutUint32(buf[12:16], pathLen)
    
    // Write path string
    copy(buf[16:], pathBytes)
    
    return buf, nil
}
```

### Unmarshal (Decode from Bytes)

```go
func (t *PathTag) UnmarshalBinary(data []byte) error {
    // Parse header
    header := &TagHeader{}
    if err := header.UnmarshalBinary(data); err != nil {
        return err
    }
    
    // Verify FourCC
    if header.FourCC != t.FourCC() {
        return fmt.Errorf("wrong fourcc: %s (expected PPTH)", header.FourCC)
    }
    
    // Read path length
    if len(data) < 16 {
        return fmt.Errorf("data too short for path length")
    }
    pathLen := ByteOrder.Uint32(data[12:16])
    
    // Read and decode path
    if len(data) < int(header.TagLen) {
        return fmt.Errorf("data too short for path string")
    }
    
    pathBytes := data[16:header.TagLen]
    path, err := decodeUTF16BE(pathBytes)
    if err != nil {
        return fmt.Errorf("decode path: %w", err)
    }
    
    t.Path = path
    return nil
}
```

### UTF-16 Big-Endian Encoding

```go
// encodeUTF16BE converts a UTF-8 string to UTF-16 Big-Endian with trailing NUL
func encodeUTF16BE(s string) ([]byte, error) {
    // Convert to runes
    runes := []rune(s)
    
    // Encode as UTF-16
    encoded := utf16.Encode(runes)
    
    // Allocate buffer with space for trailing NUL
    buf := make([]byte, (len(encoded)+1)*2)
    
    // Write each UTF-16 code unit in big-endian
    for i, r := range encoded {
        ByteOrder.PutUint16(buf[i*2:], r)
    }
    
    // Trailing NUL (0x0000) is already zero from make()
    
    return buf, nil
}

// decodeUTF16BE converts UTF-16 Big-Endian bytes to UTF-8 string
func decodeUTF16BE(data []byte) (string, error) {
    if len(data)%2 != 0 {
        return "", fmt.Errorf("odd number of bytes for UTF-16")
    }
    
    // Decode UTF-16 code units
    units := make([]uint16, len(data)/2)
    for i := 0; i < len(units); i++ {
        units[i] = ByteOrder.Uint16(data[i*2:])
    }
    
    // Remove trailing NUL if present
    if len(units) > 0 && units[len(units)-1] == 0 {
        units = units[:len(units)-1]
    }
    
    // Decode to runes
    runes := utf16.Decode(units)
    
    return string(runes), nil
}
```

## Path Format

### USB Media Root

Paths are relative to the USB media device:
- `/B/` - USB slot
- `/C/` - SD card slot

### Typical Path Structure

```
/B/Contents/{hash_prefix}/{hash}/track.mp3

Example:
/B/Contents/P016/0000875E/track.mp3
```

Where:
- `P016` - First 3 characters of track hash
- `0000875E` - Full 8-character track hash

**Note:** The hash is generated from the original track path, not the exported path.

### Path from Track Struct

```go
func pathFromTrack(track *library.Track, basedir string) string {
    // Track.OutputPath is already set by mediascanner
    // Need to convert absolute path to media-relative path
    
    // Example: /Volumes/USB/rex/track.mp3
    //       -> /B/rex/track.mp3
    
    relPath := strings.TrimPrefix(track.OutputPath, basedir)
    mediaPath := "/B" + relPath
    
    return mediaPath
}
```

## Integration Points

### Called By
- `pkg/mediascanner/mediascanner.go::GenerateAnalysisFiles()`

### Data Source
- Track's `OutputPath` field (after copying/transcoding)
- USB mount point (basedir)

### Example Usage in Mediascanner

```go
func GenerateAnalysisFiles(track *library.Track, basedir string) error {
    file := &anlz.File{
        Header: anlz.NewFileHeader(),
        Tags:   []anlz.Tag{},
    }
    
    // Create path tag
    mediaPath := strings.TrimPrefix(track.OutputPath, basedir)
    pathTag := &anlz.PathTag{
        Path: "/B" + mediaPath,
    }
    
    file.Tags = append(file.Tags, pathTag)
    // ... add other tags
    
    return file.WriteToFile(track.AnalyzePath)
}
```

## Testing Requirements

### Test Cases

1. **Simple ASCII Path**
```go
func TestPathTag_ASCII(t *testing.T) {
    tag := &PathTag{
        Path: "/B/Contents/track.mp3",
    }
    
    data, err := tag.MarshalBinary()
    assert.NoError(t, err)
    
    // Verify FourCC
    assert.Equal(t, []byte("PPTH"), data[0:4])
    
    // Round-trip
    loaded := &PathTag{}
    err = loaded.UnmarshalBinary(data)
    assert.NoError(t, err)
    assert.Equal(t, tag.Path, loaded.Path)
}
```

2. **Unicode Path**
```go
func TestPathTag_Unicode(t *testing.T) {
    tag := &PathTag{
        Path: "/B/Music/日本語/track.mp3",
    }
    
    data, err := tag.MarshalBinary()
    assert.NoError(t, err)
    
    loaded := &PathTag{}
    err = loaded.UnmarshalBinary(data)
    assert.NoError(t, err)
    assert.Equal(t, tag.Path, loaded.Path)
}
```

3. **Empty Path**
```go
func TestPathTag_Empty(t *testing.T) {
    tag := &PathTag{
        Path: "",
    }
    
    data, err := tag.MarshalBinary()
    assert.NoError(t, err)
    
    // Should still have trailing NUL (2 bytes)
    assert.Equal(t, uint32(0x10+2), binary.BigEndian.Uint32(data[8:12]))
}
```

4. **UTF-16 Encoding**
```go
func TestUTF16BE_Encoding(t *testing.T) {
    // Test basic ASCII
    bytes, err := encodeUTF16BE("AB")
    assert.NoError(t, err)
    // "A" = 0x0041, "B" = 0x0042, NUL = 0x0000
    assert.Equal(t, []byte{0x00, 0x41, 0x00, 0x42, 0x00, 0x00}, bytes)
    
    // Test decoding
    str, err := decodeUTF16BE(bytes)
    assert.NoError(t, err)
    assert.Equal(t, "AB", str)
}
```

5. **Big-Endian Verification**
```go
func TestPathTag_BigEndian(t *testing.T) {
    tag := &PathTag{Path: "/B/test.mp3"}
    data, _ := tag.MarshalBinary()
    
    // Verify lengths are big-endian
    headerLen := binary.BigEndian.Uint32(data[4:8])
    assert.Equal(t, uint32(0x10), headerLen)
    
    // Not little-endian
    notHeaderLen := binary.LittleEndian.Uint32(data[4:8])
    assert.NotEqual(t, uint32(0x10), notHeaderLen)
}
```

## Usage Examples

### Creating Path Tag

```go
tag := &anlz.PathTag{
    Path: "/B/rex/track.mp3",
}

data, err := tag.MarshalBinary()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Path tag: %d bytes\n", len(data))
```

### Reading Path Tag

```go
tag := &anlz.PathTag{}
if err := tag.UnmarshalBinary(data); err != nil {
    log.Fatal(err)
}

fmt.Printf("Track path: %s\n", tag.Path)
```

### Complete Example

```go
func createPathTag(track *library.Track, usbRoot string) (*anlz.PathTag, error) {
    // Get absolute output path
    absPath := track.OutputPath
    
    // Make relative to USB root
    relPath := strings.TrimPrefix(absPath, usbRoot)
    
    // Add media prefix
    mediaPath := "/B" + relPath
    
    return &anlz.PathTag{Path: mediaPath}, nil
}
```

## Common Pitfalls

### ❌ Using UTF-8 Instead of UTF-16
```go
// WRONG - Path must be UTF-16BE
buf.Write([]byte(t.Path))

// CORRECT
pathBytes, _ := encodeUTF16BE(t.Path)
buf.Write(pathBytes)
```

### ❌ Forgetting Trailing NUL
```go
// WRONG - Missing NUL terminator
encoded := utf16.Encode([]rune(s))
buf := make([]byte, len(encoded)*2)

// CORRECT - Space for NUL
buf := make([]byte, (len(encoded)+1)*2)
```

### ❌ Little-Endian UTF-16
```go
// WRONG - UTF-16LE
binary.LittleEndian.PutUint16(buf[i*2:], r)

// CORRECT - UTF-16BE
ByteOrder.PutUint16(buf[i*2:], r)
```

### ❌ Absolute Paths
```go
// WRONG - Absolute filesystem path
Path: "/Users/dj/Music/track.mp3"

// CORRECT - Media-relative path
Path: "/B/rex/track.mp3"
```

### ❌ Wrong PathLen
```go
// WRONG - PathLen is length of string
pathLen := uint32(len(t.Path))

// CORRECT - PathLen is bytes (UTF-16 + NUL)
pathBytes, _ := encodeUTF16BE(t.Path)
pathLen := uint32(len(pathBytes))
```

## Performance Notes

- Path encoding is simple and fast
- Typical path: 50-100 bytes
- UTF-16 doubles byte count vs ASCII
- No optimization needed

## Error Handling

### Invalid Characters
```go
// Handle paths with unsupported characters
func sanitizePath(path string) string {
    // Replace backslashes (Windows) with forward slashes
    path = strings.ReplaceAll(path, "\\", "/")
    
    // Ensure starts with /
    if !strings.HasPrefix(path, "/") {
        path = "/" + path
    }
    
    return path
}
```

### Validation
```go
func (t *PathTag) Validate() error {
    if t.Path == "" {
        return errors.New("path is empty")
    }
    
    if !strings.HasPrefix(t.Path, "/B/") && !strings.HasPrefix(t.Path, "/C/") {
        return errors.New("path must start with /B/ or /C/")
    }
    
    return nil
}
```

## Future Enhancements

1. **Path Normalization** - Automatically fix common issues
2. **Media Slot Detection** - Auto-determine /B/ vs /C/
3. **Path Validation** - Check file actually exists
4. **Compression** - Deduplicate common path prefixes

## Related Files

- `anlz.go` - Uses PathTag in File.Tags
- `tag.go` - PathTag implements Tag interface
- `pkg/mediascanner/mediascanner.go` - Generates paths from tracks

## Reference

Deep Symmetry Documentation:
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#path-tag

See byte field diagram for PPTH structure and UTF-16BE string encoding requirements.

