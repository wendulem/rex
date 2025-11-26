# rex: rekordbox exporter from audio files

Create Rekordbox-compatible export files from any folder of audio files (MP3, WAV, FLAC, M4A),
allowing them to be played on Pioneer CDJ equipment.

This project leans heavily on the reverse engineering work done by others.
A good starting point is: https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/

## Project state

**Working features:**
- ✅ Export.pdb database generation
- ✅ Track metadata (title, artist, album, BPM)
- ✅ Beat grid generation (sync works on CDJs!)
- ✅ Analysis file generation (.DAT files)
- ✅ Hot cues and loops (via JSON input)
- ✅ Playlist support

**Not yet implemented:**
- ❌ Waveforms (extraction done, tag implementation pending)
- ❌ Album artwork
- ❌ Song structure (lighting control)

This software generates Pioneer-compatible exports that work on CDJs for basic DJing.
Test thoroughly before relying on it for important gigs.

The reverse engineering work is documented in the [rekordbox package](pkg/rekordbox) subdirectories.
Especially [dbengine](pkg/rekordbox/dbengine), [page](pkg/rekordbox/page), and [anlz](pkg/rekordbox/anlz).

## Prerequisites

REX requires the following tools to be installed and available in your PATH:

### Required Tools

**FFmpeg** - Audio transcoding and analysis
```bash
# macOS
brew install ffmpeg

# Linux (Debian/Ubuntu)
sudo apt install ffmpeg

# Verify
ffmpeg -version
ffprobe -version
```

**aubio** - BPM and beat detection
```bash
# macOS
brew install aubio

# Linux (Debian/Ubuntu)
sudo apt install aubio-tools

# Verify
aubio --version
```

### Supported Audio Formats
- MP3 (copied directly)
- WAV (transcoded to MP3)
- FLAC (transcoded to MP3)
- M4A/AAC (transcoded to MP3)

## Formatting USB sticks

On Linux, the following command has been reported to create FAT32 file systems that work on Pioneer:
```
mkfs.fat -c -F 32 -n label -S 512 /dev/sdX
```

## Generate exports

This software has been tested successfully with Go 1.20+.

Use [REX](cmd/rex/main.go) to generate PDB files from a folder of audio files:

```bash
# Build
go build -o rex cmd/rex/main.go

# Generate export
./rex -root /path/to/USB -source /path/to/music/folder
```

### Options

- `-root <path>` - Root path of USB drive (required)
- `-source <path>` - Directory containing audio files to export (required)
- `-trackdir <name>` - Folder name on USB for audio files (default: "rex")
- `-cues <path>` - Optional JSON file with hot cues and loops
- `-f` - Force overwrite existing export.pdb

### Examples

```bash
# Basic export
./rex -root /Volumes/USB_DRIVE -source ~/Music/DJ

# Export with hot cues from JSON
./rex -root /Volumes/USB_DRIVE -source ~/Music/DJ -cues cues.json

# Use custom track directory
./rex -root /Volumes/USB_DRIVE -source ~/Music/DJ -trackdir my-tracks
```

### What Gets Generated

REX creates a complete Pioneer DJ export with:

1. **export.pdb** - Database with track metadata, artists, albums, playlists
2. **Analysis files (.DAT)** - Beat grids for sync, waveform data, cue points
3. **Audio files** - MP3s (original or transcoded)

Your audio files will be copied to the USB drive and organized in the specified track directory.
MP3 files are copied directly, other formats are transcoded to MP3.

**On the CDJ you can:**
- ✅ Load tracks
- ✅ See BPM (detected via aubio)
- ✅ Use sync/beatmatch (beat grids working!)
- ✅ Browse by artist/album
- ✅ Access playlists
- ✅ Use hot cues (from JSON file or add on-device)
- ✅ Jump to cue points
- ✅ Trigger loops

## Hot Cues and Loops

REX supports hot cues via a JSON file. This allows you to:
- Create hot cues in a web/mobile UI
- Export cue data as JSON
- REX embeds them in the ANLZ files

### JSON Format

See [cues.example.json](cues.example.json) for a complete example and [docs/HOT_CUE_JSON_FORMAT.md](docs/HOT_CUE_JSON_FORMAT.md) for full documentation.

**Quick example:**
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
          "label": "Drop",
          "color": 1
        },
        {
          "number": 2,
          "time_ms": 60000,
          "type": "loop",
          "loop_end_ms": 64000,
          "label": "4-Bar Loop",
          "color": 6
        }
      ]
    }
  ]
}
```

**Usage:**
```bash
./rex -root /Volumes/USB -source ~/Music -cues mycues.json
```

## Export file analysis

Use [Analyze](cmd/analyze/main.go) to introspect what's going on inside the files:

```bash
# Build analyzer
go build -o analyze cmd/analyze/main.go

# Inspect PDB database
./analyze -index -rows /path/to/USB/PIONEER/rekordbox/export.pdb

# Inspect analysis files
hexdump -C /path/to/USB/PIONEER/USBANLZ/*/*/ANLZ0000.DAT | head -50
```

## How it works

1. **Scan** - Reads audio files from source directory
2. **Analyze** - Uses FFmpeg to extract metadata and aubio to detect BPM/beats
3. **Copy** - Copies/transcodes audio files to USB
4. **Generate** - Creates export.pdb database with track info
5. **Analyze** - Generates .DAT files with beat grids for each track

The export structure matches Pioneer's format so CDJs can read it natively.

## Troubleshooting

**"aubio: command not found"**
- Install aubio (see Prerequisites above)

**"No tracks detected"**
- Check that source directory contains MP3/WAV/FLAC/M4A files
- Verify files have valid metadata

**"CDJ shows no BPM"**
- BPM detection failed, tracks will use 120 BPM default
- Check that aubio is installed and working: `aubio tempo yourfile.mp3`

## Contributing

Contributions welcome! The codebase is organized as:
- `cmd/rex/` - Main application
- `pkg/rekordbox/` - Pioneer format implementations
  - `anlz/` - Analysis file (.DAT) generation
  - `dbengine/` - PDB database engine
  - `page/` - Database page structures
- `pkg/audioproc/` - Audio analysis (aubio/FFmpeg integration)
- `pkg/mediascanner/` - Media scanning and processing
