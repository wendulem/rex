# header.go - PMAI File Header

## Purpose

This file defines the `FileHeader` struct that represents the first 28 bytes of every Pioneer DJ analysis file. The header identifies the file format and specifies the total file size.

## Source Documentation

From: https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#analysis-file-header

> For some reason the analysis files store their numbers in big-endian byte order, the opposite of the export.pdb database file.

### Header Structure

```
Byte Offset
0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f
┌──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┬──┐
│P │M │A │I │ len_header  │  len_file   │     │ 00
├──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┤
│              Unknown (16 bytes)                │ 10
└────────────────────────────────────────────────┘
│                                                │ 1c (end)
```

### Field Descriptions

| Offset | Size | Name | Description |
|--------|------|------|-------------|
| 0x00 | 4 | Magic | Four-character code "PMAI" identifying format |
| 0x04 | 4 | HeaderLen | Length of header in bytes (always 0x1c = 28) |
| 0x08 | 4 | FileLen | Total file size in bytes (header + all tags) |
| 0x0c | 16 | Unknown | Purpose unclear, usually zeros |

**Important:** All multi-byte integers are stored in **big-endian** byte order.

## Implementation

### FileHeader Struct

```go
type FileHeader struct {
    Magic     [4]byte  // "PMAI"
    HeaderLen uint32   // Usually 0x1c (28 bytes)
    FileLen   uint32   // Total file size in bytes
    Unknown   [16]byte // Purpose unknown, typically zeros
}
```

### Constructor

```go
func NewFileHeader() *FileHeader {
    return &FileHeader{
        Magic:     [4]byte{'P', 'M', 'A', 'I'},
        HeaderLen: 0x1c,
        FileLen:   0,      // Set by File.WriteToFile()
        Unknown:   [16]byte{}, // Zero-initialized
    }
}
```

### Marshal (Encode to Bytes)

```go
func (h *FileHeader) MarshalBinary() ([]byte, error) {
    buf := make([]byte, 28)
    
    // Magic: PMAI
    copy(buf[0:4], h.Magic[:])
    
    // HeaderLen: 4 bytes, big-endian
    ByteOrder.PutUint32(buf[4:8], h.HeaderLen)
    
    // FileLen: 4 bytes, big-endian
    ByteOrder.PutUint32(buf[8:12], h.FileLen)
    
    // Unknown: 16 bytes
    copy(buf[12:28], h.Unknown[:])
    
    return buf, nil
}
```

### Unmarshal (Decode from Bytes)

```go
func (h *FileHeader) UnmarshalBinary(data []byte) error {
    if len(data) < 28 {
        return fmt.Errorf("header too short: %d bytes", len(data))
    }
    
    // Verify magic
    copy(h.Magic[:], data[0:4])
    if string(h.Magic[:]) != "PMAI" {
        return fmt.Errorf("invalid magic: %q (expected PMAI)", h.Magic)
    }
    
    // Read lengths (big-endian)
    h.HeaderLen = ByteOrder.Uint32(data[4:8])
    h.FileLen = ByteOrder.Uint32(data[8:12])
    
    // Read unknown bytes
    copy(h.Unknown[:], data[12:28])
    
    return nil
}
```

### Validation

```go
func (h *FileHeader) Validate() error {
    // Check magic
    if string(h.Magic[:]) != "PMAI" {
        return fmt.Errorf("invalid magic: %q", h.Magic)
    }
    
    // Check header length
    if h.HeaderLen != 0x1c {
        return fmt.Errorf("unexpected header length: %d (expected 28)", h.HeaderLen)
    }
    
    // Check file length is at least header size
    if h.FileLen < 28 {
        return fmt.Errorf("file length too small: %d", h.FileLen)
    }
    
    return nil
}
```

## Integration

### Used By

**`anlz.go`**
```go
func (f *File) WriteToFile(path string) error {
    // Calculate total size
    f.Header.FileLen = f.calculateFileLength()
    
    // Marshal header
    headerBytes, err := f.Header.MarshalBinary()
    // ...
}
```

**`anlz.go`**
```go
func LoadFromFile(path string) (*File, error) {
    data, _ := os.ReadFile(path)
    
    header := &FileHeader{}
    if err := header.UnmarshalBinary(data[0:28]); err != nil {
        return nil, err
    }
    // ...
}
```

## Testing Requirements

### Test Cases

1. **Marshal/Unmarshal Round-trip**
```go
func TestHeaderRoundTrip(t *testing.T) {
    original := &FileHeader{
        Magic:     [4]byte{'P', 'M', 'A', 'I'},
        HeaderLen: 0x1c,
        FileLen:   1234,
        Unknown:   [16]byte{},
    }
    
    data, err := original.MarshalBinary()
    assert.NoError(t, err)
    assert.Len(t, data, 28)
    
    loaded := &FileHeader{}
    err = loaded.UnmarshalBinary(data)
    assert.NoError(t, err)
    assert.Equal(t, original, loaded)
}
```

2. **Big-Endian Verification**
```go
func TestHeaderBigEndian(t *testing.T) {
    header := &FileHeader{
        Magic:     [4]byte{'P', 'M', 'A', 'I'},
        HeaderLen: 0x1c,
        FileLen:   0x12345678,
    }
    
    data, _ := header.MarshalBinary()
    
    // Verify big-endian encoding
    // 0x12345678 should be stored as 0x12 0x34 0x56 0x78
    assert.Equal(t, byte(0x12), data[8])
    assert.Equal(t, byte(0x34), data[9])
    assert.Equal(t, byte(0x56), data[10])
    assert.Equal(t, byte(0x78), data[11])
}
```

3. **Magic Validation**
```go
func TestInvalidMagic(t *testing.T) {
    data := make([]byte, 28)
    copy(data[0:4], []byte("XXXX")) // Invalid magic
    
    header := &FileHeader{}
    err := header.UnmarshalBinary(data)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid magic")
}
```

4. **Short Buffer**
```go
func TestShortBuffer(t *testing.T) {
    data := make([]byte, 20) // Too short
    
    header := &FileHeader{}
    err := header.UnmarshalBinary(data)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "too short")
}
```

5. **Default Constructor**
```go
func TestNewFileHeader(t *testing.T) {
    header := NewFileHeader()
    
    assert.Equal(t, [4]byte{'P', 'M', 'A', 'I'}, header.Magic)
    assert.Equal(t, uint32(0x1c), header.HeaderLen)
    assert.Equal(t, uint32(0), header.FileLen) // Set later
}
```

## Usage Examples

### Creating New Header
```go
// Create header with defaults
header := anlz.NewFileHeader()

// FileLen will be set by File.WriteToFile()
```

### Reading Existing Header
```go
data, _ := os.ReadFile("ANLZ0000.DAT")

header := &anlz.FileHeader{}
if err := header.UnmarshalBinary(data[0:28]); err != nil {
    log.Fatal(err)
}

fmt.Printf("File size: %d bytes\n", header.FileLen)
```

### Validation
```go
header := &anlz.FileHeader{}
if err := header.UnmarshalBinary(data); err != nil {
    return err
}

if err := header.Validate(); err != nil {
    return fmt.Errorf("invalid header: %w", err)
}
```

## Byte Order Example

### Writing (Big-Endian)
```go
// File length: 12345 (decimal) = 0x00003039 (hex)
header.FileLen = 12345

data, _ := header.MarshalBinary()

// Bytes 8-11 in file:
// data[8]  = 0x00
// data[9]  = 0x00
// data[10] = 0x30
// data[11] = 0x39
```

### Reading (Big-Endian)
```go
// File contains: 00 00 30 39 at bytes 8-11
data := []byte{
    'P', 'M', 'A', 'I',           // Magic
    0x00, 0x00, 0x00, 0x1c,       // HeaderLen = 28
    0x00, 0x00, 0x30, 0x39,       // FileLen = 12345
    0, 0, 0, 0, 0, 0, 0, 0,       // Unknown
    0, 0, 0, 0, 0, 0, 0, 0,       // Unknown
}

header := &FileHeader{}
header.UnmarshalBinary(data)

// header.FileLen == 12345
```

## Common Pitfalls

### ❌ Using Little-Endian
```go
// WRONG - This is for PDB files
binary.LittleEndian.PutUint32(buf[8:12], h.FileLen)

// CORRECT - Analysis files are big-endian
ByteOrder.PutUint32(buf[8:12], h.FileLen)
```

### ❌ Wrong Magic Bytes
```go
// WRONG - Magic should be "PMAI"
Magic: [4]byte{'P', 'A', 'M', 'I'}

// CORRECT
Magic: [4]byte{'P', 'M', 'A', 'I'}
```

### ❌ Hardcoded Header Length
```go
// WRONG - Header length might change in future
const headerSize = 32

// CORRECT - Use struct's HeaderLen field
headerSize := int(header.HeaderLen)
```

## Unknown Field Notes

The 16 bytes at offset 0x0c are labeled "Unknown" because their purpose hasn't been reverse-engineered. Observations:

- Usually all zeros
- Sometimes contains non-zero values
- CDJs don't seem to care what's there
- Rekordbox exports always use zeros

**Recommendation:** Leave as zeros for maximum compatibility.

## Performance Notes

- Header is only 28 bytes - no optimization needed
- Marshal/Unmarshal is called once per file
- Use stack allocation for temporary buffers
- No goroutines needed

## Future Considerations

### Version Support
If Pioneer introduces new header versions:
```go
type FileHeader struct {
    Magic     [4]byte
    Version   uint16   // NEW: Header version
    HeaderLen uint16   // Changed from uint32
    FileLen   uint32
    Unknown   [16]byte
}
```

### Extended Headers
For larger files (> 4 GB):
```go
type FileHeader struct {
    Magic     [4]byte
    HeaderLen uint32
    FileLen   uint64   // Changed from uint32
    Unknown   [12]byte // Reduced to fit
}
```

## Related Files

- `anlz.go` - Uses FileHeader in File struct
- `tag.go` - Tag headers follow similar pattern
- All tag implementations - Come after file header

## Reference

Deep Symmetry Documentation:
- https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html#analysis-file-header

Byte field diagram reproduced from above URL showing big-endian storage.

