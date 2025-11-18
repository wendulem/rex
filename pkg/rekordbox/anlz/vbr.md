# vbr.go - PVBR VBR Index Tag

## Purpose

This file implements the PVBR (VBR) tag, which provides an index for fast seeking within variable-bit-rate audio files. Without this index, CDJs would need to scan the entire file from the beginning to find a specific time position, which would be too slow for features like cue points and hot cues.

## Source Documentation

From: https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#vbr-tag

> This kind of section has not yet been explained, but it is believed to hold an index allowing rapid seeking to particular times within variable-bit-rate tracks.

**Note:** This tag is not fully reverse-engineered. The structure below is based on observations.

### Tag Structure

```
Byte Offset
0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
┌──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┐
│P │V │B │R │ len_header  │   len_tag   │unk1 │ 00
├──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┤
│     unknown2    │      Index entries...       │ 10
└────────────────────────────────────────────────┘
```

### Field Descriptions

| Offset | Size | Name | Description |
|--------|------|------|-------------|
| 0x00 | 4 | FourCC | "PVBR" identifier |
| 0x04 | 4 | HeaderLen | Always 0x10 (16 bytes) |
| 0x08 | 4 | TagLen | Total length (header + index data) |
| 0x0c | 4 | Unknown1 | Purpose unknown |
| 0x10 | variable | Unknown2 | Index data (structure unclear) |

**Important:** The internal structure of the VBR index is not fully understood. This implementation provides a placeholder that can be expanded once more information is available.

## Implementation

### VBRTag Struct

```go
type VBRTag struct {
    IndexData []byte  // Opaque index data
}
```

### Interface Implementation

```go
func (t *VBRTag) FourCC() [4]byte {
    return [4]byte{'P', 'V', 'B', 'R'}
}

func (t *VBRTag) TagType() TagType {
    return TagTypeVBR
}
```

### Marshal (Encode to Bytes)

```go
func (t *VBRTag) MarshalBinary() ([]byte, error) {
    headerLen := uint32(0x10)
    dataLen := uint32(len(t.IndexData))
    tagLen := headerLen + dataLen
    
    buf := make([]byte, tagLen)
    
    // Write tag header
    copy(buf[0:4], t.FourCC()[:])
    ByteOrder.PutUint32(buf[4:8], headerLen)
    ByteOrder.PutUint32(buf[8:12], tagLen)
    
    // Unknown field
    ByteOrder.PutUint32(buf[12:16], 0)
    
    // Write index data
    if len(t.IndexData) > 0 {
        copy(buf[16:], t.IndexData)
    }
    
    return buf, nil
}
```

### Unmarshal (Decode from Bytes)

```go
func (t *VBRTag) UnmarshalBinary(data []byte) error {
    // Parse header
    header := &TagHeader{}
    if err := header.UnmarshalBinary(data); err != nil {
        return err
    }
    
    // Verify FourCC
    if header.FourCC != t.FourCC() {
        return fmt.Errorf("wrong fourcc: %s (expected PVBR)", header.FourCC)
    }
    
    // Read index data
    if len(data) > 16 {
        t.IndexData = make([]byte, len(data)-16)
        copy(t.IndexData, data[16:])
    }
    
    return nil
}
```

## VBR Index Generation

### Placeholder Implementation

Since the format is not fully understood, start with an empty or minimal index:

```go
func GenerateVBRTag() *VBRTag {
    // Empty VBR tag - CDJs may generate their own index on load
    return &VBRTag{
        IndexData: []byte{},
    }
}
```

### From MP3 File (Speculative)

MP3 files may have existing VBR headers (XING/VBRI):

```go
func GenerateVBRFromMP3(audioPath string) (*VBRTag, error) {
    // Read MP3 file
    file, err := os.Open(audioPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    // Look for XING or VBRI header
    header := make([]byte, 200)
    file.Read(header)
    
    // Check for VBR markers
    if bytes.Contains(header, []byte("Xing")) {
        // Parse XING header
        return parseXingHeader(header)
    }
    
    if bytes.Contains(header, []byte("VBRI")) {
        // Parse VBRI header
        return parseVBRIHeader(header)
    }
    
    // No VBR header found - file might be CBR
    return GenerateVBRTag(), nil
}
```

### Building Custom Index

If generating from scratch:

```go
// This is speculative - structure not confirmed
type VBRIndexEntry struct {
    TimeMs      uint32  // Time position
    FileOffset  uint32  // Byte offset in audio file
}

func BuildVBRIndex(audioPath string) (*VBRTag, error) {
    entries := []VBRIndexEntry{}
    
    // Scan through file at regular intervals
    // and record byte offsets for time positions
    
    // ... scanning logic ...
    
    // Encode entries into IndexData
    buf := &bytes.Buffer{}
    for _, entry := range entries {
        binary.Write(buf, ByteOrder, entry.TimeMs)
        binary.Write(buf, ByteOrder, entry.FileOffset)
    }
    
    return &VBRTag{
        IndexData: buf.Bytes(),
    }, nil
}
```

## When to Include VBR Tag

### CBR Files (Constant Bit Rate)
- MP3 files with `-b` constant bitrate
- Most modern encoders use CBR by default
- **VBR tag not needed**

### VBR Files (Variable Bit Rate)
- MP3 files with `-V` quality setting
- Files encoded with LAME VBR
- **VBR tag recommended** for fast seeking

### Detection

```go
func IsVBR(audioPath string) (bool, error) {
    // Check file header for VBR markers
    file, err := os.Open(audioPath)
    if err != nil {
        return false, err
    }
    defer file.Close()
    
    header := make([]byte, 200)
    file.Read(header)
    
    // XING or VBRI header indicates VBR
    hasXing := bytes.Contains(header, []byte("Xing"))
    hasVBRI := bytes.Contains(header, []byte("VBRI"))
    hasInfo := bytes.Contains(header, []byte("Info"))  // CBR with metadata
    
    return (hasXing || hasVBRI) && !hasInfo, nil
}
```

## Integration Points

### Optional Tag

VBR tag is optional - include only if needed:

```go
func GenerateAnalysisFiles(track *library.Track) error {
    file := &anlz.File{
        Header: anlz.NewFileHeader(),
        Tags:   []anlz.Tag{},
    }
    
    // Add path, beat grid, cues...
    
    // Optionally add VBR tag
    if isVBR, _ := IsVBR(track.OutputPath); isVBR {
        vbrTag := GenerateVBRTag()
        file.Tags = append(file.Tags, vbrTag)
    }
    
    return file.WriteToFile(track.AnalyzePath)
}
```

## Testing Requirements

### Test Cases

1. **Empty VBR Tag**
```go
func TestVBRTag_Empty(t *testing.T) {
    tag := &VBRTag{
        IndexData: []byte{},
    }
    
    data, err := tag.MarshalBinary()
    assert.NoError(t, err)
    
    // Should be header only (16 bytes)
    assert.Equal(t, 16, len(data))
    
    // Verify FourCC
    assert.Equal(t, []byte("PVBR"), data[0:4])
}
```

2. **Round-trip**
```go
func TestVBRTag_RoundTrip(t *testing.T) {
    original := &VBRTag{
        IndexData: []byte{1, 2, 3, 4, 5},
    }
    
    data, _ := original.MarshalBinary()
    
    loaded := &VBRTag{}
    err := loaded.UnmarshalBinary(data)
    assert.NoError(t, err)
    assert.Equal(t, original.IndexData, loaded.IndexData)
}
```

3. **VBR Detection**
```go
func TestIsVBR(t *testing.T) {
    // Test with sample MP3 files
    isVBR, err := IsVBR("testdata/vbr_track.mp3")
    assert.NoError(t, err)
    assert.True(t, isVBR)
    
    isVBR, err = IsVBR("testdata/cbr_track.mp3")
    assert.NoError(t, err)
    assert.False(t, isVBR)
}
```

## Common Pitfalls

### ❌ Including VBR Tag for CBR Files
```go
// WRONG - Always adding VBR tag
file.Tags = append(file.Tags, GenerateVBRTag())

// CORRECT - Only for VBR files
if isVBR {
    file.Tags = append(file.Tags, GenerateVBRTag())
}
```

### ❌ Wrong Header Length
```go
// WRONG - Header length might vary
headerLen := uint32(0x18)

// CORRECT - PVBR header is always 0x10
headerLen := uint32(0x10)
```

## Performance Notes

- VBR index generation can be slow (requires file scanning)
- Consider generating asynchronously
- Cache results for repeated exports
- Empty tag is fast to generate

## Limitations

**Current Limitations:**
1. Internal VBR structure not fully understood
2. May not work for all VBR files
3. CDJs might regenerate their own index

**Workaround:**
- Provide empty VBR tag
- CDJ will generate index on first load
- Subsequent loads will be faster

## Future Work

### Reverse Engineering Tasks

To fully implement VBR support:

1. **Capture Real VBR Tags**
   - Export VBR tracks from rekordbox
   - Analyze resulting PVBR tag structure
   - Document byte layout

2. **Test on Hardware**
   - Load tracks with/without VBR tags
   - Measure seeking performance
   - Verify correctness

3. **Implement Parser**
   - Once structure is known
   - Parse existing VBR tags
   - Generate compatible tags

### Proposed Structure (Unconfirmed)

```go
// Speculative - not confirmed
type VBRIndex struct {
    NumEntries uint32
    Entries    []VBRIndexEntry
}

type VBRIndexEntry struct {
    TimeMs     uint32  // Milliseconds from start
    FileOffset uint32  // Byte offset in file
    FrameSize  uint16  // Frame size at this position
}
```

## Related Files

- `anlz.go` - Uses VBRTag in File.Tags
- `tag.go` - VBRTag implements Tag interface
- `pkg/mediascanner/mediascanner.go` - Decides when to include VBR tag

## Reference

Deep Symmetry Documentation:
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#vbr-tag

External References:
- MPEG Audio Frame Header: http://www.mp3-tech.org/programmer/frame_header.html
- XING VBR Header: http://gabriel.mp3-tech.org/mp3infotag.html
- VBRI Header: http://www.codeproject.com/Articles/8295/MPEG-Audio-Frame-Header

VBR tags enable:
- Fast seeking in VBR files
- Accurate time display
- Responsive cue point jumping
- Smooth waveform scrolling

**Recommendation:** Start with empty VBR tags. CDJs will function correctly, just slightly slower on first load of VBR tracks. Implement full VBR support once format is fully understood.

