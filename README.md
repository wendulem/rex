# rex: rekordbox exporter from audio files

Create Rekordbox-compatible export files from any folder of audio files (MP3, WAV, FLAC, M4A),
allowing them to be played on Pioneer CDJ equipment.

This project leans heavily on the reverse engineering work done by others.
A good starting point is: https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/

## Project state

Do not use files generated from this project on a live gig, it probably won't work and you'll be miserable.
That said, it is possible to create PDB files that can be opened in Rekordbox.
These files have also been tested on a few Pioneer devices and are usable to varying degrees.
Trying to import them on a Denon Prime 4 results in something happening, but no library.

I figured out some more fields from various tables, and also a bit how the table structure should be built up.
The important stuff is in the [rekordbox package](pkg/rekordbox) subdirectories.
Especially the stuff in [dbengine](pkg/rekordbox/dbengine) and [page](pkg/rekordbox/page)
might be of particular interest. Many tests are broken, they might not be relevant.

## Prerequisites

REX requires **FFMPEG** to analyze and transcode audio files. Make sure `ffmpeg` and `ffprobe` are in your PATH.

Supported audio formats:
- MP3
- WAV
- FLAC
- M4A/AAC

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
- `-f` - Force overwrite existing export.pdb

### Example

```bash
# Export all music from ~/Music/DJ folder to USB drive
./rex -root /Volumes/USB_DRIVE -source ~/Music/DJ

# Use custom track directory
./rex -root /Volumes/USB_DRIVE -source ~/Music/DJ -trackdir my-tracks
```

Your audio files will be copied to the USB drive and organized in the specified track directory.
MP3 files are copied directly, other formats are transcoded to MP3.

These features are NOT supported yet:

* Waveforms
* Beat grid
* Hot Cue

## Export file analysis

Use [Analyze](cmd/analyze/main.go) to introspect what's going on inside the files:

```
go build -o analyze cmd/analyze/main.go
./analyze -index -rows /path/to/USB/PIONEER/rekordbox/export.pdb
```
