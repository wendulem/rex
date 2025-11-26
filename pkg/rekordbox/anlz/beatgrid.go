package anlz

import (
	"fmt"
	"time"
)

// Beat represents a single beat in the beat grid
type Beat struct {
	BeatNumber uint16 // Position in bar: 1, 2, 3, or 4
	Tempo      uint16 // BPM × 100 (e.g., 12500 = 125.00 BPM)
	Time       uint32 // Milliseconds from track start
}

// beatGridTagMarshalBinary encodes the BeatGridTag to bytes
func beatGridTagMarshalBinary(t *BeatGridTag) ([]byte, error) {
	const headerLen = uint32(0x18) // 24 bytes
	
	numBeats := uint32(len(t.Beats))
	beatDataLen := numBeats * 8 // 8 bytes per beat
	tagLen := headerLen + beatDataLen
	
	buf := make([]byte, tagLen)
	
	// Write tag header (12 bytes)
	copy(buf[0:12], MarshalTagHeader(t.FourCC(), headerLen, tagLen))
	
	// Unknown1 at bytes 12-15 (4 bytes)
	ByteOrder.PutUint32(buf[12:16], 0)
	
	// Unknown2 at bytes 16-19 (4 bytes) - always 0x00800000
	ByteOrder.PutUint32(buf[16:20], 0x00800000)
	
	// NumBeats at bytes 20-23 (4 bytes)
	ByteOrder.PutUint32(buf[20:24], numBeats)
	
	// Write beat entries starting at byte 24
	offset := 24
	for _, beat := range t.Beats {
		ByteOrder.PutUint16(buf[offset:offset+2], beat.BeatNumber)
		ByteOrder.PutUint16(buf[offset+2:offset+4], beat.Tempo)
		ByteOrder.PutUint32(buf[offset+4:offset+8], beat.Time)
		offset += 8
	}
	
	return buf, nil
}

// beatGridTagUnmarshalBinary decodes the BeatGridTag from bytes
func beatGridTagUnmarshalBinary(t *BeatGridTag, data []byte) error {
	// Parse tag header
	header, err := UnmarshalTagHeader(data)
	if err != nil {
		return err
	}
	
	// Verify FourCC
	if header.FourCC != t.FourCC() {
		return fmt.Errorf("wrong fourcc: %s (expected PQTZ)", string(header.FourCC[:]))
	}
	
	// Verify header length
	if header.HeaderLen != 0x18 {
		return fmt.Errorf("unexpected header length: %d (expected 24)", header.HeaderLen)
	}
	
	// Read number of beats
	if len(data) < 24 {
		return fmt.Errorf("data too short for beat count")
	}
	numBeats := ByteOrder.Uint32(data[20:24])
	
	// Verify we have enough data for all beats
	expectedLen := 24 + (numBeats * 8)
	if len(data) < int(expectedLen) {
		return fmt.Errorf("data too short for %d beats: need %d, have %d", numBeats, expectedLen, len(data))
	}
	
	// Read beat entries
	t.Beats = make([]Beat, numBeats)
	offset := 24
	
	for i := uint32(0); i < numBeats; i++ {
		t.Beats[i] = Beat{
			BeatNumber: ByteOrder.Uint16(data[offset : offset+2]),
			Tempo:      ByteOrder.Uint16(data[offset+2 : offset+4]),
			Time:       ByteOrder.Uint32(data[offset+4 : offset+8]),
		}
		offset += 8
	}
	
	return nil
}

// GenerateConstantBPMGrid creates a beat grid from a constant BPM value
func GenerateConstantBPMGrid(bpm float64, duration time.Duration) *BeatGridTag {
	if bpm <= 0 {
		bpm = 120.0 // Default BPM
	}
	
	// Calculate milliseconds per beat
	msPerBeat := 60000.0 / bpm
	
	// Calculate number of beats that fit in the duration
	durationMs := duration.Milliseconds()
	numBeats := int(float64(durationMs) / msPerBeat)
	
	if numBeats <= 0 {
		numBeats = 1 // At least one beat
	}
	
	beats := make([]Beat, numBeats)
	tempoBPM100 := uint16(bpm * 100) // BPM × 100
	
	for i := 0; i < numBeats; i++ {
		beats[i] = Beat{
			BeatNumber: uint16((i % 4) + 1), // Cycles 1, 2, 3, 4
			Tempo:      tempoBPM100,
			Time:       uint32(float64(i) * msPerBeat),
		}
	}
	
	return &BeatGridTag{Beats: beats}
}

// GenerateFromBeats creates a beat grid from a list of beat timestamps
func GenerateFromBeats(beatTimes []time.Duration, bpm float64) *BeatGridTag {
	if len(beatTimes) == 0 {
		return &BeatGridTag{Beats: []Beat{}}
	}
	
	if bpm <= 0 {
		bpm = 120.0 // Default BPM
	}
	
	beats := make([]Beat, len(beatTimes))
	tempoBPM100 := uint16(bpm * 100)
	
	for i, beatTime := range beatTimes {
		beats[i] = Beat{
			BeatNumber: uint16((i % 4) + 1), // Cycles 1, 2, 3, 4
			Tempo:      tempoBPM100,
			Time:       uint32(beatTime.Milliseconds()),
		}
	}
	
	return &BeatGridTag{Beats: beats}
}

