# Hot Cue JSON Format

This document describes the JSON format for providing hot cues, loops, and memory points to REX.

## Overview

You can create hot cues in your own UI (web app, mobile app, etc.) and export them as JSON.
REX will read this JSON file and embed the cues in the generated ANLZ files.

## Usage

```bash
# Export with hot cues
./rex -root /Volumes/USB -source ~/Music -cues cues.json
```

## JSON Schema

```json
{
  "tracks": [
    {
      "path": "/absolute/path/to/track.mp3",
      "cues": [
        {
          "number": 1,
          "time_ms": 30000,
          "type": "point",
          "label": "Drop",
          "color": 1
        }
      ]
    }
  ]
}
```

## Field Descriptions

### Track Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | ✅ | Absolute path to audio file (must match source files) |
| `cues` | array | ✅ | Array of cue objects |

### Cue Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `number` | int | ✅ | Hot cue number: 1-8 for hot cues A-H, 0 for memory point |
| `time_ms` | int | ✅ | Position in track (milliseconds) |
| `type` | string | ✅ | `"point"` for cue point, `"loop"` for loop |
| `loop_end_ms` | int | ⚠️ | Loop end position (required if type is "loop") |
| `label` | string | ❌ | Optional label/comment for the cue |
| `color` | int | ❌ | Color code 0-8 (see color table below) |

## Color Codes

| Code | Color | RGB | Description |
|------|-------|-----|-------------|
| 0 | Green | (0, 255, 0) | Default color |
| 1 | Red | (255, 0, 0) | |
| 2 | Orange | (255, 128, 0) | |
| 3 | Yellow | (255, 255, 0) | |
| 4 | Green | (0, 255, 0) | |
| 5 | Cyan | (0, 255, 255) | |
| 6 | Blue | (0, 0, 255) | |
| 7 | Magenta | (255, 0, 255) | |
| 8 | White | (255, 255, 255) | |

## Examples

### Basic Hot Cues

```json
{
  "tracks": [
    {
      "path": "/Users/dj/Music/my_track.mp3",
      "cues": [
        {
          "number": 1,
          "time_ms": 0,
          "type": "point",
          "label": "Start",
          "color": 4
        },
        {
          "number": 2,
          "time_ms": 60000,
          "type": "point",
          "label": "Drop",
          "color": 1
        }
      ]
    }
  ]
}
```

### Hot Cue with Loop

```json
{
  "tracks": [
    {
      "path": "/Users/dj/Music/track.mp3",
      "cues": [
        {
          "number": 1,
          "time_ms": 30000,
          "type": "point",
          "label": "Intro",
          "color": 0
        },
        {
          "number": 2,
          "time_ms": 90000,
          "type": "loop",
          "loop_end_ms": 98000,
          "label": "8-Bar Loop",
          "color": 6
        }
      ]
    }
  ]
}
```

### Minimal Example (No Labels or Colors)

```json
{
  "tracks": [
    {
      "path": "/Users/dj/Music/track.mp3",
      "cues": [
        {
          "number": 1,
          "time_ms": 15000,
          "type": "point"
        },
        {
          "number": 2,
          "time_ms": 45000,
          "type": "point"
        }
      ]
    }
  ]
}
```

## Generating the JSON from Your UI

### Web App Example

```javascript
// In your web app
const cues = {
  tracks: [
    {
      path: selectedTrack.path,
      cues: hotCues.map(cue => ({
        number: cue.number,
        time_ms: cue.positionMs,
        type: cue.isLoop ? "loop" : "point",
        loop_end_ms: cue.loopEndMs,
        label: cue.label,
        color: cue.colorCode
      }))
    }
  ]
};

// Save to file
const jsonString = JSON.stringify(cues, null, 2);
// Provide download or save to local file
```

### Mobile App Example

```swift
// Swift example
struct CueJSON: Codable {
    let number: Int
    let time_ms: Int64
    let type: String
    let loop_end_ms: Int64?
    let label: String?
    let color: Int?
}

struct TrackJSON: Codable {
    let path: String
    let cues: [CueJSON]
}

struct LibraryJSON: Codable {
    let tracks: [TrackJSON]
}

// Export
let library = LibraryJSON(tracks: yourTracks)
let encoder = JSONEncoder()
encoder.outputFormatting = .prettyPrinted
let data = try encoder.encode(library)
```

## Validation

REX will:
- Skip tracks not found in source directory
- Ignore cues with invalid times (> track duration)
- Warn about malformed JSON
- Continue exporting even if cues are invalid

## Tips

1. **Time Accuracy:** Use milliseconds for precise positioning
2. **Track Paths:** Must match exactly (use absolute paths)
3. **Hot Cue Numbers:** 1-8 for hot cues, 0 for memory points
4. **Loop Validation:** Ensure `loop_end_ms` > `time_ms`
5. **Labels:** Optional but helpful for organization
6. **Colors:** Optional, defaults to green (0)

## Testing Your JSON

```bash
# Validate JSON syntax
cat cues.json | jq .

# Generate export with cues
./rex -root /tmp/test -source ~/Music -cues cues.json

# Check if cues were embedded
hexdump -C /tmp/test/PIONEER/USBANLZ/*/*/ANLZ0000.DAT | grep -A 5 "PCO2"
```

