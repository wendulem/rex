package main

import (
	`context`
	`flag`
	`fmt`
	`io`
	`os`
	`path/filepath`
	`strings`

	`github.com/ambientsound/rex/pkg/library`
	`github.com/ambientsound/rex/pkg/mediascanner`
	`github.com/ambientsound/rex/pkg/rekordbox/color`
	`github.com/ambientsound/rex/pkg/rekordbox/column`
	`github.com/ambientsound/rex/pkg/rekordbox/dbengine`
	`github.com/ambientsound/rex/pkg/rekordbox/page`
	`github.com/ambientsound/rex/pkg/rekordbox/pdb`
	`github.com/ambientsound/rex/pkg/rekordbox/playlist`
	`github.com/ambientsound/rex/pkg/rekordbox/unknown17`
	`github.com/ambientsound/rex/pkg/rekordbox/unknown18`
)

func main() {
	err := run()
	if err != nil {
		fmt.Printf("fatal error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	var err error

	fmt.Printf("REX: Pioneer DJ export generator from audio files\n")
	fmt.Printf("This software is neither supported nor endorsed by Pioneer.\n")
	fmt.Printf("Please do not rely on it for serious use.\n")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lib := library.New()

	// Initialize options
	basedir := flag.String("root", "./", "Root path of USB drive")
	trackDir := flag.String("trackdir", "rex", "Where on the USB drive to put exported files, relative to root path")
	sourceDir := flag.String("source", "", "Directory containing audio files to export")
	forceOverwrite := flag.Bool("f", false, "Overwrite export file if it exists")
	flag.Parse()

	if *sourceDir == "" {
		return fmt.Errorf("must specify -source directory with audio files")
	}

	*basedir, err = filepath.Abs(*basedir)
	if err != nil {
		return err
	}

	// Scan for audio files
	fmt.Printf("Scanning for audio files in: %s\n", *sourceDir)
	audioFiles, err := scanAudioFiles(*sourceDir)
	if err != nil {
		return fmt.Errorf("scan audio files: %w", err)
	}
	fmt.Printf("Found %d audio files\n", len(audioFiles))

	// Create output directories
	outputPath := filepath.Join(*basedir, "PIONEER", "rekordbox")
	err = os.MkdirAll(outputPath, 0755)
	if err != nil {
		return err
	}
	*trackDir = filepath.Join(*basedir, *trackDir)
	*trackDir, err = filepath.Abs(*trackDir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(*trackDir, 0755)
	if err != nil {
		return err
	}

	// Open output file for writing
	outputFile := filepath.Join(outputPath, "export.pdb")
	outputFile, err = filepath.Abs(outputFile)
	if err != nil {
		return err
	}
	var flags = os.O_CREATE | os.O_RDWR
	if *forceOverwrite {
		flags |= os.O_TRUNC
	}
	out, err := os.OpenFile(outputFile, flags, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	fmt.Printf("PIONEER database created: %s\n", outputFile)

	// Probe each audio file for metadata
	fmt.Printf("Analyzing audio files...\n")
	for i, audioPath := range audioFiles {
		fmt.Printf("\033[2K\r[%6d/%6d] %s", i+1, len(audioFiles), filepath.Base(audioPath))

		probe, err := mediascanner.ProbeMetadata(ctx, audioPath)
		if err != nil {
			fmt.Printf("\nWarning: skipping %s: %v\n", audioPath, err)
			continue
		}

		track := mediascanner.TrackFromFile(lib, audioPath, *probe)
		lib.InsertTrack(track)
	}
	fmt.Printf("\033[2K\r")
	fmt.Printf("Analyzed %d tracks\n", len(lib.Tracks().All()))

	// Create a single "All Tracks" playlist
	allTracksPlaylist := &library.Playlist{
		ID:     1,
		Name:   "All Tracks",
		Tracks: lib.Tracks().All(),
	}
	lib.Playlists().Insert(allTracksPlaylist)

	fmt.Printf("Tracks marked for export: %d\n", len(lib.Tracks().All()))
	fmt.Printf("Copying or encoding tracks to %s\n", *trackDir)

	for i, t := range lib.Tracks().All() {
		fmt.Printf("\r[%6d/%6d] ", i+1, len(lib.Tracks().All()))
		result, err := mediascanner.RenderTo(ctx, t, *trackDir)
		if err != nil {
			fmt.Printf("\n")
			return fmt.Errorf("render %q: %w\n", t.OutputPath, err)
		}
		fmt.Printf("\033[2K\r[%6d/%6d] %s %s", i+1, len(lib.Tracks().All()), result.Action, t.OutputPath)
	}

	fmt.Printf("\033[2K\r")
	fmt.Printf("All tracks copied to destination\n")
	fmt.Printf("Writing PDB file...\n")

	// Intermediary type for storing "INSERT statements"
	type Insert struct {
		Type page.Type
		Row  page.Row
	}
	inserts := make([]Insert, 0)

	// Create PDB data types for tracks, artists, albums and playlists.
	tracks := lib.Tracks().All()
	for i := range tracks {
		pdbtrack := mediascanner.PdbTrack(lib, tracks[i], *basedir)
		inserts = append(inserts, Insert{
			Type: page.Type_Tracks,
			Row:  &pdbtrack,
		})
	}

	artists := lib.Artists().All()
	for i := range artists {
		pdbartist := mediascanner.PdbArtist(lib, artists[i])
		inserts = append(inserts, Insert{
			Type: page.Type_Artists,
			Row:  &pdbartist,
		})
	}

	albums := lib.Albums().All()
	for i := range albums {
		pdbalbum := mediascanner.PdbAlbum(lib, albums[i])
		inserts = append(inserts, Insert{
			Type: page.Type_Albums,
			Row:  &pdbalbum,
		})
	}

	// Generate playlists
	playlists := lib.Playlists().All()
	for playlistID := range playlists {
		pl := &playlist.Playlist{
			PlaylistHeader: playlist.PlaylistHeader{
				Id: uint32(playlistID),
			},
			Name: playlists[playlistID].GetName(),
		}
		inserts = append(inserts, Insert{
			Type: page.Type_PlaylistTree,
			Row:  pl,
		})
		for trackIndex, t := range playlists[playlistID].Tracks {
			ent := &playlist.Entry{
				EntryIndex: uint32(trackIndex + 1),
				TrackID:    uint32(lib.Tracks().ID(t)),
				PlaylistID: uint32(playlistID),
			}
			inserts = append(inserts, Insert{
				Type: page.Type_PlaylistEntries,
				Row:  ent,
			})
		}
	}

	for _, uk := range unknown17.InitialDataset {
		inserts = append(inserts, Insert{
			Type: page.Type_Unknown17,
			Row:  uk,
		})
	}

	for _, uk := range unknown18.InitialDataset {
		inserts = append(inserts, Insert{
			Type: page.Type_Unknown18,
			Row:  uk,
		})
	}

	for _, uk := range color.InitialDataset {
		inserts = append(inserts, Insert{
			Type: page.Type_Colors,
			Row:  uk,
		})
	}

	for _, uk := range column.InitialDataset {
		inserts = append(inserts, Insert{
			Type: page.Type_Columns,
			Row:  uk,
		})
	}

	// Initialize the database.
	db := dbengine.New(out)

	// Create all tables found in a typical rekordbox export.
	for _, pageType := range pdb.TableOrder {
		err = db.CreateTable(pageType)
		if err != nil {
			panic(err)
		}
	}

	// Generate data pages with the inserts generated earlier.
	// When a data page is full, it is inserted into the db.
	// This is a quick and dirty way for export ONLY,
	// it will not work to modify existing databases.
	dataPages := make(map[page.Type]*page.Data)
	for _, insert := range inserts {
		if dataPages[insert.Type] == nil {
			dataPages[insert.Type] = page.NewPage(insert.Type)
		}
		err = dataPages[insert.Type].Insert(insert.Row)
		if err == nil {
			continue
		}
		if err == io.ErrShortWrite {
			err = db.InsertPage(dataPages[insert.Type])
			if err != nil {
				panic(err)
			}
			dataPages[insert.Type] = nil
			continue
		}
		panic(err)
	}

	// Insert the remainding pages.
	for _, pg := range dataPages {
		if pg == nil {
			continue
		}
		err = db.InsertPage(pg)
		if err != nil {
			panic(err)
		}
	}

	// Flush buffers and exit program.
	err = out.Close()
	if err == nil {
		fmt.Printf("Finished successfully.\n")
	}

	return nil
}

// scanAudioFiles recursively scans a directory for audio files
func scanAudioFiles(rootDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".mp3" || ext == ".wav" || ext == ".flac" || ext == ".m4a" {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}
