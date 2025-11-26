package anlz

import (
	"fmt"
)

// CueEntry represents a single cue point or loop entry (PCP2 format)
type CueEntry struct {
	// Entry identification
	HotCueNumber uint32 // 0 for memory point, 1-8 for hot cues A-H

	// Cue data
	Type     uint8  // 1 = cue point, 2 = loop
	Time     uint32 // Position in milliseconds
	LoopTime uint32 // End position for loops (0 for cue points)

	// Loop quantization (for auto loops)
	LoopNumerator   uint16 // Loop size numerator (0 if not quantized)
	LoopDenominator uint16 // Loop size denominator (0 if not quantized)

	// Memory point/loop color (uses color table)
	ColorID uint8 // Color table ID (for memory points/loops)

	// Hot cue color (direct RGB)
	ColorCode  uint8 // Rekordbox color palette code (0 = default green)
	ColorRed   uint8 // RGB red component
	ColorGreen uint8 // RGB green component
	ColorBlue  uint8 // RGB blue component

	// Comment/label
	Comment string // Optional UTF-16BE comment
}

// IsHotCue returns true if this is a hot cue (not a memory point)
func (e *CueEntry) IsHotCue() bool {
	return e.HotCueNumber > 0
}

// IsLoop returns true if this is a loop (not a point)
func (e *CueEntry) IsLoop() bool {
	return e.Type == 2
}

// cueListExtendedTagMarshalBinary encodes the CueListExtendedTag to bytes
func cueListExtendedTagMarshalBinary(t *CueListExtendedTag) ([]byte, error) {
	const headerLen = uint32(0x14) // 20 bytes

	// Marshal all entries first to calculate sizes
	entryData := make([][]byte, len(t.Entries))
	totalEntryLen := uint32(0)

	for i, entry := range t.Entries {
		data, err := marshalCueEntry(&entry)
		if err != nil {
			return nil, fmt.Errorf("marshal entry %d: %w", i, err)
		}
		entryData[i] = data
		totalEntryLen += uint32(len(data))
	}

	// Calculate tag size
	tagLen := headerLen + totalEntryLen

	buf := make([]byte, tagLen)

	// Write tag header (12 bytes)
	copy(buf[0:12], MarshalTagHeader(t.FourCC(), headerLen, tagLen))

	// Type at bytes 12-15 (0 = memory points, 1 = hot cues)
	ByteOrder.PutUint32(buf[12:16], t.Type)

	// NumEntries at bytes 16-17
	ByteOrder.PutUint16(buf[16:18], uint16(len(t.Entries)))

	// Unknown at bytes 18-19 (usually 0000)
	ByteOrder.PutUint16(buf[18:20], 0)

	// Write entry data starting at byte 20
	offset := 20
	for _, data := range entryData {
		copy(buf[offset:], data)
		offset += len(data)
	}

	return buf, nil
}

// marshalCueEntry encodes a single CueEntry (PCP2 format)
func marshalCueEntry(e *CueEntry) ([]byte, error) {
	const baseHeaderLen = uint32(0x10) // 16 bytes for PCP2 header

	// Encode comment as UTF-16BE
	commentBytes := []byte{}
	if e.Comment != "" {
		commentBytes = encodeUTF16BE(e.Comment)
	}

	// Calculate entry length
	// Base: 16 (PCP2 header) + 28 (fixed data) + comment + 4 (colors) = 48 + comment
	entryLen := baseHeaderLen + 28 + uint32(len(commentBytes)) + 4

	buf := make([]byte, entryLen)

	// PCP2 entry header (12 bytes)
	copy(buf[0:4], []byte("PCP2"))
	ByteOrder.PutUint32(buf[4:8], baseHeaderLen)
	ByteOrder.PutUint32(buf[8:12], entryLen)

	// HotCueNumber at bytes 12-15
	ByteOrder.PutUint32(buf[12:16], e.HotCueNumber)

	// Type at byte 16 (1 = cue, 2 = loop)
	buf[16] = e.Type

	// Unknown1 at bytes 17-19 (usually 0x0003e8 = 1000 decimal)
	ByteOrder.PutUint16(buf[17:19], 0x03e8)
	buf[19] = 0x00

	// Time at bytes 20-23 (milliseconds)
	ByteOrder.PutUint32(buf[20:24], e.Time)

	// LoopTime at bytes 24-27
	ByteOrder.PutUint32(buf[24:28], e.LoopTime)

	// ColorID at byte 28
	buf[28] = e.ColorID

	// Unknown2 at bytes 29-35 (usually 0x01 followed by zeros)
	buf[29] = 0x01
	// Bytes 30-35 remain zero

	// LoopNumerator at bytes 36-37
	ByteOrder.PutUint16(buf[36:38], e.LoopNumerator)

	// LoopDenominator at bytes 38-39
	ByteOrder.PutUint16(buf[38:40], e.LoopDenominator)

	// CommentLen at bytes 40-43
	ByteOrder.PutUint32(buf[40:44], uint32(len(commentBytes)))

	// Comment at bytes 44+ (UTF-16BE)
	if len(commentBytes) > 0 {
		copy(buf[44:], commentBytes)
	}

	// Hot cue color at end (4 bytes)
	colorOffset := 44 + len(commentBytes)
	buf[colorOffset] = e.ColorCode
	buf[colorOffset+1] = e.ColorRed
	buf[colorOffset+2] = e.ColorGreen
	buf[colorOffset+3] = e.ColorBlue

	return buf, nil
}

// cueListExtendedTagUnmarshalBinary decodes the CueListExtendedTag from bytes
func cueListExtendedTagUnmarshalBinary(t *CueListExtendedTag, data []byte) error {
	// Parse tag header
	header, err := UnmarshalTagHeader(data)
	if err != nil {
		return err
	}

	// Verify FourCC
	if header.FourCC != t.FourCC() {
		return fmt.Errorf("wrong fourcc: %s (expected PCO2)", string(header.FourCC[:]))
	}

	// Verify header length
	if header.HeaderLen != 0x14 {
		return fmt.Errorf("unexpected header length: %d (expected 20)", header.HeaderLen)
	}

	// Read type
	if len(data) < 16 {
		return fmt.Errorf("data too short for type")
	}
	t.Type = ByteOrder.Uint32(data[12:16])

	// Read number of entries
	if len(data) < 18 {
		return fmt.Errorf("data too short for num entries")
	}
	numEntries := ByteOrder.Uint16(data[16:18])

	// Read entries
	t.Entries = make([]CueEntry, 0, numEntries)
	offset := 20

	for i := uint16(0); i < numEntries; i++ {
		if offset >= len(data) {
			break
		}

		entry, bytesRead, err := unmarshalCueEntry(data[offset:])
		if err != nil {
			return fmt.Errorf("unmarshal entry %d: %w", i, err)
		}

		t.Entries = append(t.Entries, *entry)
		offset += bytesRead
	}

	return nil
}

// unmarshalCueEntry decodes a single CueEntry, returns entry and bytes consumed
func unmarshalCueEntry(data []byte) (*CueEntry, int, error) {
	// Parse PCP2 header
	if len(data) < 12 {
		return nil, 0, fmt.Errorf("data too short for entry header")
	}

	// Verify FourCC
	fourCC := string(data[0:4])
	if fourCC != "PCP2" {
		return nil, 0, fmt.Errorf("wrong entry fourcc: %s (expected PCP2)", fourCC)
	}

	// Read entry length
	entryLen := ByteOrder.Uint32(data[8:12])

	if len(data) < int(entryLen) {
		return nil, 0, fmt.Errorf("data too short for entry: need %d, have %d", entryLen, len(data))
	}

	entry := &CueEntry{}

	// HotCueNumber at bytes 12-15
	entry.HotCueNumber = ByteOrder.Uint32(data[12:16])

	// Type at byte 16
	if len(data) < 17 {
		return entry, int(entryLen), nil // Incomplete entry
	}
	entry.Type = data[16]

	// Time at bytes 20-23
	if len(data) < 24 {
		return entry, int(entryLen), nil // Incomplete entry
	}
	entry.Time = ByteOrder.Uint32(data[20:24])

	// LoopTime at bytes 24-27
	if len(data) < 28 {
		return entry, int(entryLen), nil // Incomplete entry
	}
	entry.LoopTime = ByteOrder.Uint32(data[24:28])

	// ColorID at byte 28
	if len(data) < 29 {
		return entry, int(entryLen), nil // Incomplete entry
	}
	entry.ColorID = data[28]

	// LoopNumerator at bytes 36-37
	if len(data) < 38 {
		return entry, int(entryLen), nil // Incomplete entry
	}
	entry.LoopNumerator = ByteOrder.Uint16(data[36:38])

	// LoopDenominator at bytes 38-39
	if len(data) < 40 {
		return entry, int(entryLen), nil // Incomplete entry
	}
	entry.LoopDenominator = ByteOrder.Uint16(data[38:40])

	// CommentLen at bytes 40-43
	if len(data) < 44 {
		return entry, int(entryLen), nil // Incomplete entry
	}
	commentLen := ByteOrder.Uint32(data[40:44])

	// Comment (UTF-16BE)
	if commentLen > 0 && len(data) >= int(44+commentLen) {
		comment, err := decodeUTF16BE(data[44 : 44+commentLen])
		if err == nil {
			entry.Comment = comment
		}
	}

	// Hot cue colors at end
	colorOffset := 44 + int(commentLen)
	if len(data) >= colorOffset+4 {
		entry.ColorCode = data[colorOffset]
		entry.ColorRed = data[colorOffset+1]
		entry.ColorGreen = data[colorOffset+2]
		entry.ColorBlue = data[colorOffset+3]
	}

	return entry, int(entryLen), nil
}

// NewHotCue creates a new hot cue entry
func NewHotCue(number int, timeMs uint32, label string, colorCode uint8) CueEntry {
	// Get RGB values for color code
	red, green, blue := colorCodeToRGB(colorCode)

	return CueEntry{
		HotCueNumber:    uint32(number),
		Type:            1, // Cue point
		Time:            timeMs,
		LoopTime:        0,
		ColorID:         0,
		LoopNumerator:   0,
		LoopDenominator: 0,
		Comment:         label,
		ColorCode:       colorCode,
		ColorRed:        red,
		ColorGreen:      green,
		ColorBlue:       blue,
	}
}

// NewHotLoop creates a new hot loop entry
func NewHotLoop(number int, startMs, endMs uint32, label string, colorCode uint8) CueEntry {
	red, green, blue := colorCodeToRGB(colorCode)

	return CueEntry{
		HotCueNumber:    uint32(number),
		Type:            2, // Loop
		Time:            startMs,
		LoopTime:        endMs,
		ColorID:         0,
		LoopNumerator:   0,
		LoopDenominator: 0,
		Comment:         label,
		ColorCode:       colorCode,
		ColorRed:        red,
		ColorGreen:      green,
		ColorBlue:       blue,
	}
}

// colorCodeToRGB converts a rekordbox color code to RGB values
// This is a simplified mapping - rekordbox has a specific palette
func colorCodeToRGB(code uint8) (uint8, uint8, uint8) {
	// Default green (code 0)
	if code == 0 {
		return 0, 255, 0
	}

	// Basic palette (codes 1-8)
	colors := [][3]uint8{
		{255, 0, 0},     // 1: Red
		{255, 128, 0},   // 2: Orange
		{255, 255, 0},   // 3: Yellow
		{0, 255, 0},     // 4: Green
		{0, 255, 255},   // 5: Cyan
		{0, 0, 255},     // 6: Blue
		{255, 0, 255},   // 7: Magenta
		{255, 255, 255}, // 8: White
	}

	if code > 0 && int(code) <= len(colors) {
		return colors[code-1][0], colors[code-1][1], colors[code-1][2]
	}

	// Default to green for unknown codes
	return 0, 255, 0
}

