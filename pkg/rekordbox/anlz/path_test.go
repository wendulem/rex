package anlz

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPathTag_MarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"simple path", "/Contents/track.mp3"},
		{"path with spaces", "/My Music/Track Name.mp3"},
		{"unix path", "/B/rex/audio/song.mp3"},
		{"empty path", ""},
		{"unicode path", "/Music/日本語/track.mp3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := &PathTag{
				Path: tt.path,
			}

			// Marshal
			data, err := original.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary failed: %v", err)
			}

			// Verify minimum size (header is 16 bytes)
			if len(data) < 16 {
				t.Errorf("Data too short: %d bytes", len(data))
			}

			// Verify FourCC
			if string(data[0:4]) != "PPTH" {
				t.Errorf("Wrong FourCC: %q", data[0:4])
			}

			// Unmarshal
			loaded := &PathTag{}
			err = loaded.UnmarshalBinary(data)
			if err != nil {
				t.Fatalf("UnmarshalBinary failed: %v", err)
			}

			// Verify path matches
			if loaded.Path != original.Path {
				t.Errorf("Path mismatch: %q != %q", loaded.Path, original.Path)
			}
		})
	}
}

func TestPathTag_BigEndian(t *testing.T) {
	tag := &PathTag{
		Path: "/test/path.mp3",
	}

	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Verify header length is big-endian (bytes 4-7)
	// Should be 0x00000010 (16 decimal)
	if data[4] != 0x00 || data[5] != 0x00 || data[6] != 0x00 || data[7] != 0x10 {
		t.Errorf("HeaderLen not big-endian: %v", data[4:8])
	}

	// Verify tag length is big-endian (bytes 8-11)
	tagLen := ByteOrder.Uint32(data[8:12])
	if tagLen != uint32(len(data)) {
		t.Errorf("TagLen mismatch: %d != %d", tagLen, len(data))
	}

	// Verify path length is big-endian (bytes 12-15)
	pathLen := ByteOrder.Uint32(data[12:16])
	expectedPathLen := uint32(len(data) - 16)
	if pathLen != expectedPathLen {
		t.Errorf("PathLen mismatch: %d != %d", pathLen, expectedPathLen)
	}
}

func TestPathTag_UTF16Encoding(t *testing.T) {
	tag := &PathTag{
		Path: "AB", // Simple ASCII for verification
	}

	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Path starts at byte 16
	// "AB" in UTF-16BE with NUL should be:
	// 0x00 0x41 (A) 0x00 0x42 (B) 0x00 0x00 (NUL)
	pathData := data[16:]
	
	if len(pathData) != 6 {
		t.Errorf("Expected 6 bytes for 'AB' + NUL, got %d", len(pathData))
	}

	// Check 'A' (0x0041 in UTF-16BE)
	if pathData[0] != 0x00 || pathData[1] != 0x41 {
		t.Errorf("'A' not encoded correctly: %02x %02x", pathData[0], pathData[1])
	}

	// Check 'B' (0x0042 in UTF-16BE)
	if pathData[2] != 0x00 || pathData[3] != 0x42 {
		t.Errorf("'B' not encoded correctly: %02x %02x", pathData[2], pathData[3])
	}

	// Check trailing NUL (0x0000)
	if pathData[4] != 0x00 || pathData[5] != 0x00 {
		t.Errorf("Trailing NUL not present: %02x %02x", pathData[4], pathData[5])
	}
}

func TestPathTag_EmptyPath(t *testing.T) {
	tag := &PathTag{
		Path: "",
	}

	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Header (16 bytes) + just NUL terminator (2 bytes) = 18 bytes
	if len(data) != 18 {
		t.Errorf("Expected 18 bytes for empty path, got %d", len(data))
	}

	// Unmarshal and verify
	loaded := &PathTag{}
	err = loaded.UnmarshalBinary(data)
	if err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	if loaded.Path != "" {
		t.Errorf("Expected empty path, got %q", loaded.Path)
	}
}

func TestPathTag_UnicodeRoundtrip(t *testing.T) {
	tests := []string{
		"Hello World",
		"日本語",
		"Émilie",
		"Привет",
		"🎵 Music 🎶",
	}

	for _, testPath := range tests {
		t.Run(testPath, func(t *testing.T) {
			original := &PathTag{Path: testPath}

			data, err := original.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary failed: %v", err)
			}

			loaded := &PathTag{}
			err = loaded.UnmarshalBinary(data)
			if err != nil {
				t.Fatalf("UnmarshalBinary failed: %v", err)
			}

			if loaded.Path != original.Path {
				t.Errorf("Unicode path mismatch: %q != %q", loaded.Path, original.Path)
			}
		})
	}
}

func TestPathTag_InvalidFourCC(t *testing.T) {
	// Create data with wrong FourCC
	data := make([]byte, 20)
	copy(data[0:4], []byte("XXXX"))
	ByteOrder.PutUint32(data[4:8], 0x10)
	ByteOrder.PutUint32(data[8:12], 20)

	tag := &PathTag{}
	err := tag.UnmarshalBinary(data)
	if err == nil {
		t.Error("Expected error for wrong FourCC, got nil")
	}
}

func TestPathTag_ShortBuffer(t *testing.T) {
	data := make([]byte, 10) // Too short

	tag := &PathTag{}
	err := tag.UnmarshalBinary(data)
	if err == nil {
		t.Error("Expected error for short buffer, got nil")
	}
}

func TestPathTag_InFile(t *testing.T) {
	// Create a file with just a path tag
	file := &File{
		Header: NewFileHeader(),
		Tags: []Tag{
			&PathTag{Path: "/B/rex/test.mp3"},
		},
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.dat")

	// Write
	err := file.WriteToFile(tmpFile)
	if err != nil {
		t.Fatalf("WriteToFile failed: %v", err)
	}

	// Read back
	loaded, err := LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify we have one tag
	if len(loaded.Tags) != 1 {
		t.Fatalf("Expected 1 tag, got %d", len(loaded.Tags))
	}

	// Verify it's a path tag
	pathTag, ok := loaded.Tags[0].(*PathTag)
	if !ok {
		t.Fatalf("Tag is not a PathTag")
	}

	// Verify path
	if pathTag.Path != "/B/rex/test.mp3" {
		t.Errorf("Path mismatch: %q", pathTag.Path)
	}
}

func TestEncodeDecodeUTF16BE(t *testing.T) {
	tests := []string{
		"",
		"A",
		"Hello",
		"Hello World",
		"日本語",
		"🎵",
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			// Encode
			encoded := encodeUTF16BE(test)

			// Verify it's even length (UTF-16 uses 2-byte units)
			if len(encoded)%2 != 0 {
				t.Errorf("Encoded length not even: %d", len(encoded))
			}

			// Verify trailing NUL
			if len(encoded) >= 2 {
				if encoded[len(encoded)-2] != 0 || encoded[len(encoded)-1] != 0 {
					t.Errorf("Missing trailing NUL: %v", encoded[len(encoded)-2:])
				}
			}

			// Decode
			decoded, err := decodeUTF16BE(encoded)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			// Verify match
			if decoded != test {
				t.Errorf("Mismatch: %q != %q", decoded, test)
			}
		})
	}
}

func TestDecodeUTF16BE_OddLength(t *testing.T) {
	// Odd number of bytes is invalid for UTF-16
	data := []byte{0x00, 0x41, 0x00}

	_, err := decodeUTF16BE(data)
	if err == nil {
		t.Error("Expected error for odd-length data, got nil")
	}
}

func TestPathTag_FileStructure(t *testing.T) {
	// Create a detailed test of the file structure
	tag := &PathTag{
		Path: "/test.mp3",
	}

	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Verify structure byte by byte
	// Bytes 0-3: FourCC "PPTH"
	if !bytes.Equal(data[0:4], []byte("PPTH")) {
		t.Errorf("FourCC wrong: %v", data[0:4])
	}

	// Bytes 4-7: HeaderLen (0x10 = 16 in big-endian)
	headerLen := ByteOrder.Uint32(data[4:8])
	if headerLen != 0x10 {
		t.Errorf("HeaderLen wrong: %d", headerLen)
	}

	// Bytes 8-11: TagLen (total length)
	tagLen := ByteOrder.Uint32(data[8:12])
	if tagLen != uint32(len(data)) {
		t.Errorf("TagLen wrong: %d != %d", tagLen, len(data))
	}

	// Bytes 12-15: PathLen
	pathLen := ByteOrder.Uint32(data[12:16])
	expectedPathLen := uint32(len(data) - 16)
	if pathLen != expectedPathLen {
		t.Errorf("PathLen wrong: %d != %d", pathLen, expectedPathLen)
	}

	// Bytes 16+: Path data (UTF-16BE)
	// Already tested in other tests
}

func TestPathTag_RealWorldPath(t *testing.T) {
	// Test with a realistic rekordbox path
	tag := &PathTag{
		Path: "/Contents/0016/00010203/01.mp3",
	}

	data, err := tag.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Write to file and read back
	tmpFile := filepath.Join(t.TempDir(), "test.dat")
	err = os.WriteFile(tmpFile, data, 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Read back and verify
	readData, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	loaded := &PathTag{}
	err = loaded.UnmarshalBinary(readData)
	if err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	if loaded.Path != tag.Path {
		t.Errorf("Path mismatch: %q != %q", loaded.Path, tag.Path)
	}
}

