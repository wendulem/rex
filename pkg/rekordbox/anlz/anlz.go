package anlz

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

// ByteOrder defines the byte order for ANLZ files (BIG-ENDIAN)
// This is the opposite of PDB files which use little-endian
var ByteOrder = binary.BigEndian

// File represents a complete Pioneer DJ analysis file (.DAT, .EXT, .2EX)
type File struct {
	Header *FileHeader
	Tags   []Tag
}

// Tag is the interface that all tag types must implement
type Tag interface {
	// FourCC returns the 4-character code identifying the tag type
	FourCC() [4]byte

	// TagType returns the enum identifying the tag type
	TagType() TagType

	// MarshalBinary encodes the tag to bytes (including tag header)
	MarshalBinary() ([]byte, error)

	// UnmarshalBinary decodes the tag from bytes (including tag header)
	UnmarshalBinary(data []byte) error
}

// WriteToFile marshals the complete file and writes it to disk
func (f *File) WriteToFile(path string) error {
	// 1. Marshal all tags first to calculate sizes
	tagData := make([][]byte, len(f.Tags))
	totalSize := uint32(28) // File header size

	for i, tag := range f.Tags {
		data, err := tag.MarshalBinary()
		if err != nil {
			fourCC := tag.FourCC()
			return fmt.Errorf("marshal tag %d (%s): %w", i, string(fourCC[:]), err)
		}
		tagData[i] = data
		totalSize += uint32(len(data))
	}

	// 2. Update header with total file size
	f.Header.FileLen = totalSize

	// 3. Marshal header
	headerBytes, err := f.Header.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal header: %w", err)
	}

	// 4. Create directory if needed
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// 5. Write to file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header
	if _, err := file.Write(headerBytes); err != nil {
		return err
	}

	// Write all tags
	for i, data := range tagData {
		if _, err := file.Write(data); err != nil {
			return fmt.Errorf("write tag %d: %w", i, err)
		}
	}

	return nil
}

// LoadFromFile reads and parses an existing ANLZ file
func LoadFromFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse header
	if len(data) < 28 {
		return nil, fmt.Errorf("file too short for header: %d bytes", len(data))
	}

	header := &FileHeader{}
	if err := header.UnmarshalBinary(data[0:28]); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}

	// Parse tags
	offset := 28
	tags := []Tag{}

	for offset < len(data) {
		if len(data) < offset+12 {
			break // Not enough data for tag header
		}

		// Read FourCC
		fourCC := [4]byte{data[offset], data[offset+1], data[offset+2], data[offset+3]}

		// Read tag length
		tagLen := ByteOrder.Uint32(data[offset+8 : offset+12])

		if len(data) < offset+int(tagLen) {
			return nil, fmt.Errorf("incomplete tag %s at offset %d", string(fourCC[:]), offset)
		}

		// Create appropriate tag type
		tag := createTagForFourCC(fourCC)
		if tag == nil {
			// Unknown tag, skip it
			offset += int(tagLen)
			continue
		}

		// Unmarshal tag
		if err := tag.UnmarshalBinary(data[offset : offset+int(tagLen)]); err != nil {
			return nil, fmt.Errorf("unmarshal tag %s: %w", string(fourCC[:]), err)
		}

		tags = append(tags, tag)
		offset += int(tagLen)
	}

	return &File{Header: header, Tags: tags}, nil
}

// createTagForFourCC creates a tag instance based on the FourCC code
func createTagForFourCC(fourCC [4]byte) Tag {
	tagType := TagTypeFromFourCC(fourCC)
	
	switch tagType {
	case TagTypeBeatGrid:
		return &BeatGridTag{}
	case TagTypePath:
		return &PathTag{}
	case TagTypeCueListExtended:
		return &CueListExtendedTag{}
	case TagTypeCueList:
		return &CueListTag{}
	case TagTypeWaveformPreview:
		return &WaveformPreviewTag{}
	case TagTypeWaveformTinyPreview:
		return &WaveformTinyPreviewTag{}
	case TagTypeWaveformDetail:
		return &WaveformDetailTag{}
	case TagTypeWaveformColorPreview:
		return &WaveformColorPreviewTag{}
	case TagTypeWaveformColorDetail:
		return &WaveformColorDetailTag{}
	case TagTypeWaveform3BandPreview:
		return &Waveform3BandPreviewTag{}
	case TagTypeWaveform3BandDetail:
		return &Waveform3BandDetailTag{}
	case TagTypeVBR:
		return &VBRTag{}
	case TagTypeSongStructure:
		return &SongStructureTag{}
	default:
		return nil // Unknown tag
	}
}

