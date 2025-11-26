package anlz

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestFileHeader_MarshalUnmarshal(t *testing.T) {
	original := &FileHeader{
		Magic:     [4]byte{'P', 'M', 'A', 'I'},
		HeaderLen: 0x1c,
		FileLen:   1234,
		Unknown:   [16]byte{},
	}

	// Marshal
	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	if len(data) != 28 {
		t.Errorf("Expected 28 bytes, got %d", len(data))
	}

	// Unmarshal
	loaded := &FileHeader{}
	err = loaded.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	// Verify
	if original.Magic != loaded.Magic {
		t.Errorf("Magic mismatch: %v != %v", original.Magic, loaded.Magic)
	}
	if original.HeaderLen != loaded.HeaderLen {
		t.Errorf("HeaderLen mismatch: %d != %d", original.HeaderLen, loaded.HeaderLen)
	}
	if original.FileLen != loaded.FileLen {
		t.Errorf("FileLen mismatch: %d != %d", original.FileLen, loaded.FileLen)
	}
}

func TestFileHeader_BigEndian(t *testing.T) {
	header := &FileHeader{
		Magic:     [4]byte{'P', 'M', 'A', 'I'},
		HeaderLen: 0x1c,
		FileLen:   0x12345678,
	}

	data, err := header.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Verify big-endian encoding of FileLen
	// 0x12345678 should be stored as 0x12 0x34 0x56 0x78
	if data[8] != 0x12 {
		t.Errorf("Byte 8: expected 0x12, got 0x%02x", data[8])
	}
	if data[9] != 0x34 {
		t.Errorf("Byte 9: expected 0x34, got 0x%02x", data[9])
	}
	if data[10] != 0x56 {
		t.Errorf("Byte 10: expected 0x56, got 0x%02x", data[10])
	}
	if data[11] != 0x78 {
		t.Errorf("Byte 11: expected 0x78, got 0x%02x", data[11])
	}
}

func TestFileHeader_InvalidMagic(t *testing.T) {
	data := make([]byte, 28)
	copy(data[0:4], []byte("XXXX")) // Invalid magic

	header := &FileHeader{}
	err := header.UnmarshalBinary(data)
	if err == nil {
		t.Error("Expected error for invalid magic, got nil")
	}
}

func TestFileHeader_ShortBuffer(t *testing.T) {
	data := make([]byte, 20) // Too short

	header := &FileHeader{}
	err := header.UnmarshalBinary(data)
	if err == nil {
		t.Error("Expected error for short buffer, got nil")
	}
}

func TestNewFileHeader(t *testing.T) {
	header := NewFileHeader()

	if string(header.Magic[:]) != "PMAI" {
		t.Errorf("Expected PMAI, got %q", header.Magic)
	}
	if header.HeaderLen != 0x1c {
		t.Errorf("Expected HeaderLen 0x1c, got 0x%x", header.HeaderLen)
	}
	if header.FileLen != 0 {
		t.Errorf("Expected FileLen 0, got %d", header.FileLen)
	}
}

func TestFileHeader_Validate(t *testing.T) {
	// Valid header
	valid := &FileHeader{
		Magic:     [4]byte{'P', 'M', 'A', 'I'},
		HeaderLen: 0x1c,
		FileLen:   100,
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("Valid header failed validation: %v", err)
	}

	// Invalid magic
	invalidMagic := &FileHeader{
		Magic:     [4]byte{'X', 'X', 'X', 'X'},
		HeaderLen: 0x1c,
		FileLen:   100,
	}
	if err := invalidMagic.Validate(); err == nil {
		t.Error("Expected error for invalid magic")
	}

	// Invalid header length
	invalidHeaderLen := &FileHeader{
		Magic:     [4]byte{'P', 'M', 'A', 'I'},
		HeaderLen: 32,
		FileLen:   100,
	}
	if err := invalidHeaderLen.Validate(); err == nil {
		t.Error("Expected error for invalid header length")
	}

	// Invalid file length
	invalidFileLen := &FileHeader{
		Magic:     [4]byte{'P', 'M', 'A', 'I'},
		HeaderLen: 0x1c,
		FileLen:   10,
	}
	if err := invalidFileLen.Validate(); err == nil {
		t.Error("Expected error for invalid file length")
	}
}

func TestEmptyFile(t *testing.T) {
	file := &File{
		Header: NewFileHeader(),
		Tags:   []Tag{},
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.dat")

	// Write
	err := file.WriteToFile(tmpFile)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	// Verify file exists
	info, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("File not created: %v", err)
	}

	// Should be exactly 28 bytes (header only)
	if info.Size() != 28 {
		t.Errorf("Expected file size 28, got %d", info.Size())
	}

	// Read back
	loaded, err := LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if len(loaded.Tags) != 0 {
		t.Errorf("Expected 0 tags, got %d", len(loaded.Tags))
	}

	if loaded.Header.FileLen != 28 {
		t.Errorf("Expected FileLen 28, got %d", loaded.Header.FileLen)
	}
}

func TestTagTypeFromFourCC(t *testing.T) {
	tests := []struct {
		fourCC   [4]byte
		expected TagType
	}{
		{[4]byte{'P', 'Q', 'T', 'Z'}, TagTypeBeatGrid},
		{[4]byte{'P', 'P', 'T', 'H'}, TagTypePath},
		{[4]byte{'P', 'C', 'O', 'B'}, TagTypeCueList},
		{[4]byte{'P', 'C', 'O', '2'}, TagTypeCueListExtended},
		{[4]byte{'P', 'W', 'A', 'V'}, TagTypeWaveformPreview},
		{[4]byte{'P', 'W', 'V', '2'}, TagTypeWaveformTinyPreview},
		{[4]byte{'P', 'W', 'V', '3'}, TagTypeWaveformDetail},
		{[4]byte{'P', 'W', 'V', '4'}, TagTypeWaveformColorPreview},
		{[4]byte{'P', 'W', 'V', '5'}, TagTypeWaveformColorDetail},
		{[4]byte{'P', 'W', 'V', '6'}, TagTypeWaveform3BandPreview},
		{[4]byte{'P', 'W', 'V', '7'}, TagTypeWaveform3BandDetail},
		{[4]byte{'P', 'V', 'B', 'R'}, TagTypeVBR},
		{[4]byte{'P', 'S', 'S', 'I'}, TagTypeSongStructure},
		{[4]byte{'X', 'X', 'X', 'X'}, TagTypeUnknown},
	}

	for _, tt := range tests {
		result := TagTypeFromFourCC(tt.fourCC)
		if result != tt.expected {
			t.Errorf("FourCC %s: expected %s, got %s", 
				string(tt.fourCC[:]), tt.expected, result)
		}
	}
}

func TestTagType_String(t *testing.T) {
	tests := []struct {
		tagType  TagType
		expected string
	}{
		{TagTypeBeatGrid, "BeatGrid"},
		{TagTypePath, "Path"},
		{TagTypeCueList, "CueList"},
		{TagTypeUnknown, "Unknown"},
	}

	for _, tt := range tests {
		result := tt.tagType.String()
		if result != tt.expected {
			t.Errorf("TagType %d: expected %q, got %q", tt.tagType, tt.expected, result)
		}
	}
}

func TestMarshalTagHeader(t *testing.T) {
	fourCC := [4]byte{'P', 'Q', 'T', 'Z'}
	headerLen := uint32(0x18)
	tagLen := uint32(100)

	data := MarshalTagHeader(fourCC, headerLen, tagLen)

	if len(data) != 12 {
		t.Errorf("Expected 12 bytes, got %d", len(data))
	}

	// Verify FourCC
	if string(data[0:4]) != "PQTZ" {
		t.Errorf("FourCC mismatch: %q", data[0:4])
	}

	// Verify big-endian encoding
	if binary.BigEndian.Uint32(data[4:8]) != headerLen {
		t.Errorf("HeaderLen mismatch")
	}
	if binary.BigEndian.Uint32(data[8:12]) != tagLen {
		t.Errorf("TagLen mismatch")
	}
}

func TestUnmarshalTagHeader(t *testing.T) {
	data := make([]byte, 12)
	copy(data[0:4], []byte("PQTZ"))
	binary.BigEndian.PutUint32(data[4:8], 0x18)
	binary.BigEndian.PutUint32(data[8:12], 100)

	header, err := UnmarshalTagHeader(data)
	if err != nil {
		t.Fatalf("UnmarshalTagHeader failed: %v", err)
	}

	if string(header.FourCC[:]) != "PQTZ" {
		t.Errorf("FourCC mismatch: %q", header.FourCC)
	}
	if header.HeaderLen != 0x18 {
		t.Errorf("HeaderLen mismatch: %d", header.HeaderLen)
	}
	if header.TagLen != 100 {
		t.Errorf("TagLen mismatch: %d", header.TagLen)
	}
}

func TestByteOrder(t *testing.T) {
	// Verify ByteOrder is big-endian
	buf := make([]byte, 4)
	ByteOrder.PutUint32(buf, 0x12345678)

	if buf[0] != 0x12 || buf[1] != 0x34 || buf[2] != 0x56 || buf[3] != 0x78 {
		t.Errorf("ByteOrder is not big-endian: %v", buf)
	}
}

