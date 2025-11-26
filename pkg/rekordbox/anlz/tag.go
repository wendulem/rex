package anlz

import (
	"fmt"
)

// TagType represents the type of an ANLZ tag
type TagType int

const (
	TagTypeUnknown TagType = iota
	TagTypeBeatGrid
	TagTypePath
	TagTypeCueList
	TagTypeCueListExtended
	TagTypeWaveformPreview
	TagTypeWaveformTinyPreview
	TagTypeWaveformDetail
	TagTypeWaveformColorPreview
	TagTypeWaveformColorDetail
	TagTypeWaveform3BandPreview
	TagTypeWaveform3BandDetail
	TagTypeVBR
	TagTypeSongStructure
)

// String returns the human-readable name of the tag type
func (t TagType) String() string {
	switch t {
	case TagTypeBeatGrid:
		return "BeatGrid"
	case TagTypePath:
		return "Path"
	case TagTypeCueList:
		return "CueList"
	case TagTypeCueListExtended:
		return "CueListExtended"
	case TagTypeWaveformPreview:
		return "WaveformPreview"
	case TagTypeWaveformTinyPreview:
		return "WaveformTinyPreview"
	case TagTypeWaveformDetail:
		return "WaveformDetail"
	case TagTypeWaveformColorPreview:
		return "WaveformColorPreview"
	case TagTypeWaveformColorDetail:
		return "WaveformColorDetail"
	case TagTypeWaveform3BandPreview:
		return "Waveform3BandPreview"
	case TagTypeWaveform3BandDetail:
		return "Waveform3BandDetail"
	case TagTypeVBR:
		return "VBR"
	case TagTypeSongStructure:
		return "SongStructure"
	default:
		return "Unknown"
	}
}

// TagTypeFromFourCC returns the TagType for a given FourCC code
func TagTypeFromFourCC(fourCC [4]byte) TagType {
	switch string(fourCC[:]) {
	case "PQTZ":
		return TagTypeBeatGrid
	case "PPTH":
		return TagTypePath
	case "PCOB":
		return TagTypeCueList
	case "PCO2":
		return TagTypeCueListExtended
	case "PWAV":
		return TagTypeWaveformPreview
	case "PWV2":
		return TagTypeWaveformTinyPreview
	case "PWV3":
		return TagTypeWaveformDetail
	case "PWV4":
		return TagTypeWaveformColorPreview
	case "PWV5":
		return TagTypeWaveformColorDetail
	case "PWV6":
		return TagTypeWaveform3BandPreview
	case "PWV7":
		return TagTypeWaveform3BandDetail
	case "PVBR":
		return TagTypeVBR
	case "PSSI":
		return TagTypeSongStructure
	default:
		return TagTypeUnknown
	}
}

// TagHeader represents the common 12-byte header present in all tags
type TagHeader struct {
	FourCC    [4]byte
	HeaderLen uint32
	TagLen    uint32
}

// MarshalTagHeader creates the common 12-byte tag header
func MarshalTagHeader(fourCC [4]byte, headerLen, tagLen uint32) []byte {
	buf := make([]byte, 12)

	// FourCC (4 bytes)
	copy(buf[0:4], fourCC[:])

	// Header length (4 bytes, big-endian)
	ByteOrder.PutUint32(buf[4:8], headerLen)

	// Tag length (4 bytes, big-endian)
	ByteOrder.PutUint32(buf[8:12], tagLen)

	return buf
}

// UnmarshalTagHeader parses the common 12-byte tag header
func UnmarshalTagHeader(data []byte) (*TagHeader, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("data too short for tag header: %d bytes", len(data))
	}

	header := &TagHeader{}
	copy(header.FourCC[:], data[0:4])
	header.HeaderLen = ByteOrder.Uint32(data[4:8])
	header.TagLen = ByteOrder.Uint32(data[8:12])

	return header, nil
}

// Stub tag types (to be implemented in separate files)
// These are referenced in anlz.go's createTagForFourCC

type BeatGridTag struct{}

func (t *BeatGridTag) FourCC() [4]byte                     { return [4]byte{'P', 'Q', 'T', 'Z'} }
func (t *BeatGridTag) TagType() TagType                    { return TagTypeBeatGrid }
func (t *BeatGridTag) MarshalBinary() ([]byte, error)      { return nil, fmt.Errorf("not implemented") }
func (t *BeatGridTag) UnmarshalBinary(data []byte) error   { return fmt.Errorf("not implemented") }

type PathTag struct{}

func (t *PathTag) FourCC() [4]byte                         { return [4]byte{'P', 'P', 'T', 'H'} }
func (t *PathTag) TagType() TagType                        { return TagTypePath }
func (t *PathTag) MarshalBinary() ([]byte, error)          { return nil, fmt.Errorf("not implemented") }
func (t *PathTag) UnmarshalBinary(data []byte) error       { return fmt.Errorf("not implemented") }

type CueListTag struct{}

func (t *CueListTag) FourCC() [4]byte                      { return [4]byte{'P', 'C', 'O', 'B'} }
func (t *CueListTag) TagType() TagType                     { return TagTypeCueList }
func (t *CueListTag) MarshalBinary() ([]byte, error)       { return nil, fmt.Errorf("not implemented") }
func (t *CueListTag) UnmarshalBinary(data []byte) error    { return fmt.Errorf("not implemented") }

type CueListExtendedTag struct{}

func (t *CueListExtendedTag) FourCC() [4]byte              { return [4]byte{'P', 'C', 'O', '2'} }
func (t *CueListExtendedTag) TagType() TagType             { return TagTypeCueListExtended }
func (t *CueListExtendedTag) MarshalBinary() ([]byte, error) { return nil, fmt.Errorf("not implemented") }
func (t *CueListExtendedTag) UnmarshalBinary(data []byte) error { return fmt.Errorf("not implemented") }

type WaveformPreviewTag struct{}

func (t *WaveformPreviewTag) FourCC() [4]byte              { return [4]byte{'P', 'W', 'A', 'V'} }
func (t *WaveformPreviewTag) TagType() TagType             { return TagTypeWaveformPreview }
func (t *WaveformPreviewTag) MarshalBinary() ([]byte, error) { return nil, fmt.Errorf("not implemented") }
func (t *WaveformPreviewTag) UnmarshalBinary(data []byte) error { return fmt.Errorf("not implemented") }

type WaveformTinyPreviewTag struct{}

func (t *WaveformTinyPreviewTag) FourCC() [4]byte          { return [4]byte{'P', 'W', 'V', '2'} }
func (t *WaveformTinyPreviewTag) TagType() TagType         { return TagTypeWaveformTinyPreview }
func (t *WaveformTinyPreviewTag) MarshalBinary() ([]byte, error) { return nil, fmt.Errorf("not implemented") }
func (t *WaveformTinyPreviewTag) UnmarshalBinary(data []byte) error { return fmt.Errorf("not implemented") }

type WaveformDetailTag struct{}

func (t *WaveformDetailTag) FourCC() [4]byte               { return [4]byte{'P', 'W', 'V', '3'} }
func (t *WaveformDetailTag) TagType() TagType              { return TagTypeWaveformDetail }
func (t *WaveformDetailTag) MarshalBinary() ([]byte, error) { return nil, fmt.Errorf("not implemented") }
func (t *WaveformDetailTag) UnmarshalBinary(data []byte) error { return fmt.Errorf("not implemented") }

type WaveformColorPreviewTag struct{}

func (t *WaveformColorPreviewTag) FourCC() [4]byte         { return [4]byte{'P', 'W', 'V', '4'} }
func (t *WaveformColorPreviewTag) TagType() TagType        { return TagTypeWaveformColorPreview }
func (t *WaveformColorPreviewTag) MarshalBinary() ([]byte, error) { return nil, fmt.Errorf("not implemented") }
func (t *WaveformColorPreviewTag) UnmarshalBinary(data []byte) error { return fmt.Errorf("not implemented") }

type WaveformColorDetailTag struct{}

func (t *WaveformColorDetailTag) FourCC() [4]byte          { return [4]byte{'P', 'W', 'V', '5'} }
func (t *WaveformColorDetailTag) TagType() TagType         { return TagTypeWaveformColorDetail }
func (t *WaveformColorDetailTag) MarshalBinary() ([]byte, error) { return nil, fmt.Errorf("not implemented") }
func (t *WaveformColorDetailTag) UnmarshalBinary(data []byte) error { return fmt.Errorf("not implemented") }

type Waveform3BandPreviewTag struct{}

func (t *Waveform3BandPreviewTag) FourCC() [4]byte         { return [4]byte{'P', 'W', 'V', '6'} }
func (t *Waveform3BandPreviewTag) TagType() TagType        { return TagTypeWaveform3BandPreview }
func (t *Waveform3BandPreviewTag) MarshalBinary() ([]byte, error) { return nil, fmt.Errorf("not implemented") }
func (t *Waveform3BandPreviewTag) UnmarshalBinary(data []byte) error { return fmt.Errorf("not implemented") }

type Waveform3BandDetailTag struct{}

func (t *Waveform3BandDetailTag) FourCC() [4]byte          { return [4]byte{'P', 'W', 'V', '7'} }
func (t *Waveform3BandDetailTag) TagType() TagType         { return TagTypeWaveform3BandDetail }
func (t *Waveform3BandDetailTag) MarshalBinary() ([]byte, error) { return nil, fmt.Errorf("not implemented") }
func (t *Waveform3BandDetailTag) UnmarshalBinary(data []byte) error { return fmt.Errorf("not implemented") }

type VBRTag struct{}

func (t *VBRTag) FourCC() [4]byte                          { return [4]byte{'P', 'V', 'B', 'R'} }
func (t *VBRTag) TagType() TagType                         { return TagTypeVBR }
func (t *VBRTag) MarshalBinary() ([]byte, error)           { return nil, fmt.Errorf("not implemented") }
func (t *VBRTag) UnmarshalBinary(data []byte) error        { return fmt.Errorf("not implemented") }

type SongStructureTag struct{}

func (t *SongStructureTag) FourCC() [4]byte                { return [4]byte{'P', 'S', 'S', 'I'} }
func (t *SongStructureTag) TagType() TagType               { return TagTypeSongStructure }
func (t *SongStructureTag) MarshalBinary() ([]byte, error) { return nil, fmt.Errorf("not implemented") }
func (t *SongStructureTag) UnmarshalBinary(data []byte) error { return fmt.Errorf("not implemented") }

