package library

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// CuePoint represents a hot cue, loop, or memory point
type CuePoint struct {
	Number    int           // 1-8 for hot cues, 0 for memory point
	Time      time.Duration // Position in track
	Type      CueType       // Point or Loop
	LoopEnd   time.Duration // End position if Type == Loop
	Label     string        // User-defined label
	ColorCode int           // 0-62 (rekordbox color palette)
}

// CueType specifies whether a cue is a point or loop
type CueType int

const (
	CueTypePoint CueType = iota // Simple cue point
	CueTypeLoop                 // Loop (has start and end)
)

// CueLibrary stores hot cues for multiple tracks
type CueLibrary struct {
	Cues map[string][]CuePoint // Key: track path
}

// GetCuesForTrack returns all cues for a given track
func (c *CueLibrary) GetCuesForTrack(trackPath string) []CuePoint {
	if c == nil || c.Cues == nil {
		return []CuePoint{}
	}
	return c.Cues[trackPath]
}

// LoadCuesFromJSON loads hot cues from a JSON file
func LoadCuesFromJSON(path string) (*CueLibrary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var cueData struct {
		Tracks []struct {
			Path string `json:"path"`
			Cues []struct {
				Number    int    `json:"number"`
				TimeMs    int64  `json:"time_ms"`
				Type      string `json:"type"`
				LoopEndMs int64  `json:"loop_end_ms,omitempty"`
				Label     string `json:"label,omitempty"`
				Color     int    `json:"color,omitempty"`
			} `json:"cues"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal(data, &cueData); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	// Convert to CueLibrary
	library := &CueLibrary{
		Cues: make(map[string][]CuePoint),
	}

	for _, track := range cueData.Tracks {
		cues := make([]CuePoint, 0, len(track.Cues))
		
		for _, c := range track.Cues {
			// Determine cue type
			cueType := CueTypePoint
			if c.Type == "loop" {
				cueType = CueTypeLoop
			}

			cue := CuePoint{
				Number:    c.Number,
				Time:      time.Duration(c.TimeMs) * time.Millisecond,
				Type:      cueType,
				LoopEnd:   time.Duration(c.LoopEndMs) * time.Millisecond,
				Label:     c.Label,
				ColorCode: c.Color,
			}
			
			cues = append(cues, cue)
		}
		
		library.Cues[track.Path] = cues
	}

	return library, nil
}

// SaveCuesToJSON saves the cue library to a JSON file
func (c *CueLibrary) SaveToJSON(path string) error {
	// Convert to JSON format
	var cueData struct {
		Tracks []struct {
			Path string `json:"path"`
			Cues []struct {
				Number    int    `json:"number"`
				TimeMs    int64  `json:"time_ms"`
				Type      string `json:"type"`
				LoopEndMs int64  `json:"loop_end_ms,omitempty"`
				Label     string `json:"label,omitempty"`
				Color     int    `json:"color,omitempty"`
			} `json:"cues"`
		} `json:"tracks"`
	}

	for trackPath, cues := range c.Cues {
		track := struct {
			Path string `json:"path"`
			Cues []struct {
				Number    int    `json:"number"`
				TimeMs    int64  `json:"time_ms"`
				Type      string `json:"type"`
				LoopEndMs int64  `json:"loop_end_ms,omitempty"`
				Label     string `json:"label,omitempty"`
				Color     int    `json:"color,omitempty"`
			} `json:"cues"`
		}{
			Path: trackPath,
		}

		for _, cue := range cues {
			cueType := "point"
			if cue.Type == CueTypeLoop {
				cueType = "loop"
			}

			jsonCue := struct {
				Number    int    `json:"number"`
				TimeMs    int64  `json:"time_ms"`
				Type      string `json:"type"`
				LoopEndMs int64  `json:"loop_end_ms,omitempty"`
				Label     string `json:"label,omitempty"`
				Color     int    `json:"color,omitempty"`
			}{
				Number:    cue.Number,
				TimeMs:    cue.Time.Milliseconds(),
				Type:      cueType,
				LoopEndMs: cue.LoopEnd.Milliseconds(),
				Label:     cue.Label,
				Color:     cue.ColorCode,
			}

			track.Cues = append(track.Cues, jsonCue)
		}

		cueData.Tracks = append(cueData.Tracks, track)
	}

	// Write to file
	jsonBytes, err := json.MarshalIndent(cueData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, jsonBytes, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

