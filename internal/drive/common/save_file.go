package common

import (
	"fmt"
	"io"
	"os"
	"time"
)

type SaveFileArgs struct {
	Out           io.Writer
	Body          io.Reader
	ContentLength int64
	Path          string
	Force         bool
	Skip          bool
	Stdout        bool
	Progress      io.Writer
}

func SaveFile(args SaveFileArgs) (int64, int64, error) {
	// Wrap response body in progress reader
	srcReader := GetProgressReader(args.Body, args.Progress, args.ContentLength)

	if args.Stdout {
		// Write file content to stdout
		_, err := io.Copy(args.Out, srcReader)
		return 0, 0, err
	}

	// Check if file exists to force
	if !args.Skip && !args.Force && FileExists(args.Path) {
		return 0, 0, fmt.Errorf("File '%s' already exists, use --force to overwrite or --skip to skip", args.Path)
	}

	//Check if file exists to skip
	if args.Skip && FileExists(args.Path) {
		fmt.Printf("File '%s' already exists, skipping\n", args.Path)
		return 0, 0, nil
	}

	// Ensure any parent directories exists
	if err := Mkdir(args.Path); err != nil {
		return 0, 0, err
	}

	// Download to tmp file
	tmpPath := args.Path + ".incomplete"

	// Create new file
	outFile, err := os.Create(tmpPath)
	if err != nil {
		return 0, 0, fmt.Errorf("Unable to create new file: %s", err)
	}

	started := time.Now()

	// Save file to disk
	bytes, err := io.Copy(outFile, srcReader)
	if err != nil {
		outFile.Close()
		os.Remove(tmpPath)
		return 0, 0, fmt.Errorf("Failed saving file: %s", err)
	}

	// Calculate average download rate
	rate := CalcRate(bytes, started, time.Now())

	// Close File
	outFile.Close()

	// Rename tmp file to proper filename
	return bytes, rate, os.Rename(tmpPath, args.Path)
}
