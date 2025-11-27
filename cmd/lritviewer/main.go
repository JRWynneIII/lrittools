package main

import (
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/log"
	"github.com/jrwynneiii/lrittools/tui"
)

var cli struct {
	Dir string `arg:"" help:"Path to directory containing LRIT files"`
}

func main() {
	_ = kong.Parse(&cli)

	if files, err := filepath.Glob(filepath.Join(cli.Dir, "*.lrit")); err == nil {
		if len(files) == 0 {
			log.Fatalf("No LRIT files found")
		}
		for idx, file := range files {
			files[idx] = filepath.Base(file)
		}
		tui.StartLRITViewerUI(files, cli.Dir)
	} else {
		log.Fatalf("Error reading directory: %s", err.Error())
	}

}
