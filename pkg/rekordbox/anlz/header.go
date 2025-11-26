package anlz

import (
	"fmt"
)

// FileHeader represents the PMAI file header (first 28 bytes)
type FileHeader struct {
	Magic     [4]byte  // "PMAI" identifier
	HeaderLen uint32   // Length of header in bytes (always 0x1c = 28)
	FileLen   uint32   // Total file size in bytes (header + all tags)
	Unknown   [16]byte // Purpose unknown, usually zeros
}

// NewFileHeader creates a new file header with default values
func NewFileHeader() *FileHeader {
	return &FileHeader{
		Magic:     [4]byte{'P', 'M', 'A', 'I'},
		HeaderLen: 0x1c, // 28 bytes
		FileLen:   0,    // Will be set by File.WriteToFile()
		Unknown:   [16]byte{},
	}
}

// MarshalBinary encodes the file header to 28 bytes
func (h *FileHeader) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 28)

	// Magic: PMAI (4 bytes)
	copy(buf[0:4], h.Magic[:])

	// HeaderLen: 4 bytes, big-endian
	ByteOrder.PutUint32(buf[4:8], h.HeaderLen)

	// FileLen: 4 bytes, big-endian
	ByteOrder.PutUint32(buf[8:12], h.FileLen)

	// Unknown: 16 bytes
	copy(buf[12:28], h.Unknown[:])

	return buf, nil
}

// UnmarshalBinary decodes the file header from 28 bytes
func (h *FileHeader) UnmarshalBinary(data []byte) error {
	if len(data) < 28 {
		return fmt.Errorf("header too short: %d bytes (expected 28)", len(data))
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

// Validate checks if the header has valid values
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

