package anlz

import (
	"fmt"
	"unicode/utf16"
)

// pathTagMarshalBinary encodes the PathTag to bytes
func pathTagMarshalBinary(t *PathTag) ([]byte, error) {
	const headerLen = uint32(0x10) // 16 bytes
	
	// Encode path as UTF-16BE with trailing NUL
	pathBytes := encodeUTF16BE(t.Path)
	
	// Calculate total tag length
	tagLen := headerLen + uint32(len(pathBytes))
	
	// Create buffer
	buf := make([]byte, tagLen)
	
	// Write tag header (12 bytes)
	copy(buf[0:12], MarshalTagHeader(t.FourCC(), headerLen, tagLen))
	
	// Write path length at bytes 12-15 (4 bytes)
	ByteOrder.PutUint32(buf[12:16], uint32(len(pathBytes)))
	
	// Write path data starting at byte 16
	copy(buf[16:], pathBytes)
	
	return buf, nil
}

// pathTagUnmarshalBinary decodes the PathTag from bytes
func pathTagUnmarshalBinary(t *PathTag, data []byte) error {
	// Parse tag header
	header, err := UnmarshalTagHeader(data)
	if err != nil {
		return err
	}
	
	// Verify FourCC
	if header.FourCC != t.FourCC() {
		return fmt.Errorf("wrong fourcc: %s (expected PPTH)", string(header.FourCC[:]))
	}
	
	// Verify header length
	if header.HeaderLen != 0x10 {
		return fmt.Errorf("unexpected header length: %d (expected 16)", header.HeaderLen)
	}
	
	// Read path length
	if len(data) < 16 {
		return fmt.Errorf("data too short for path length")
	}
	pathLen := ByteOrder.Uint32(data[12:16])
	
	// Verify we have enough data
	if len(data) < int(16+pathLen) {
		return fmt.Errorf("data too short for path: need %d, have %d", 16+pathLen, len(data))
	}
	
	// Decode path from UTF-16BE
	pathBytes := data[16 : 16+pathLen]
	path, err := decodeUTF16BE(pathBytes)
	if err != nil {
		return fmt.Errorf("decode path: %w", err)
	}
	
	t.Path = path
	return nil
}

// encodeUTF16BE encodes a string to UTF-16 Big-Endian with trailing NUL
func encodeUTF16BE(s string) []byte {
	// Convert string to runes, then to UTF-16
	runes := []rune(s)
	encoded := utf16.Encode(runes)
	
	// Allocate buffer: 2 bytes per uint16, plus 2 bytes for trailing NUL
	buf := make([]byte, (len(encoded)+1)*2)
	
	// Write each uint16 as big-endian
	for i, r := range encoded {
		ByteOrder.PutUint16(buf[i*2:], r)
	}
	
	// Trailing NUL is already zero-initialized
	// buf[len(buf)-2:] = 0x00 0x00
	
	return buf
}

// decodeUTF16BE decodes a UTF-16 Big-Endian byte slice to a string
func decodeUTF16BE(data []byte) (string, error) {
	if len(data)%2 != 0 {
		return "", fmt.Errorf("invalid UTF-16BE data: odd number of bytes")
	}
	
	// Read uint16 values
	numChars := len(data) / 2
	encoded := make([]uint16, numChars)
	
	for i := 0; i < numChars; i++ {
		encoded[i] = ByteOrder.Uint16(data[i*2:])
	}
	
	// Remove trailing NULs
	for len(encoded) > 0 && encoded[len(encoded)-1] == 0 {
		encoded = encoded[:len(encoded)-1]
	}
	
	// Decode from UTF-16 to runes
	runes := utf16.Decode(encoded)
	
	return string(runes), nil
}

