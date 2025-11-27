package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/log"
	"github.com/jrwynneiii/ccsds_tools/lrit"
)

const (
	RC_SUCCESS              = 0
	RC_IO_ERROR             = 1
	RC_INVALID_PH           = 2
	RC_INVALID_SH           = 3
	RC_FAILED_DECOMPRESSION = 4
)

var cli struct {
	Paths        []string `arg:"" help:"Path to LRIT file" sep:" "`
	ListHeaders  bool     `help:"Print the available secondary headers" default:"false"`
	NoDecompress bool     `help:"Disable zip decompression" default:"false"`
	NonText      bool     `help:"Displays non-text data; This might corrupt your terminal environment if not piped!" default:"false"`
}

func main() {
	_ = kong.Parse(&cli)
	if len(cli.Paths) == 0 {
		os.Exit(0)
	}

	var rc int = 0
	for _, path := range cli.Paths {
		ret := catLritFile(path)
		if ret > rc {
			rc = ret
		}
	}
	os.Exit(rc)
}

func catLritFile(path string) int {
	if fileData, err := os.ReadFile(path); err == nil {
		ph := lrit.MakePrimaryHeader(fileData)
		fileData = fileData[16:]
		lf := lrit.File{
			PrimaryHeader: ph,
			Data:          fileData,
			CRCGood:       true, //Probably can remove this from the lrit.File type
		}

		if err = lf.PopulateSecondaryHeaders(); err != nil {
			log.Errorf("Could not parse LRIT file headers (%s): %s", path, err.Error())
			return RC_INVALID_SH
		}

		if valid, err := lf.IsValid(); !valid {
			log.Errorf("Invalid LRIT file (%s): %s", path, err.Error())
			log.Errorf("Have: %d, Want: %d", len(lf.Data), lf.PrimaryHeader.DataLength/8)
			return RC_INVALID_PH
		}

		if cli.ListHeaders {
			fmt.Printf("Headers:\nPrimary Header:   \t%##v\n", lf.PrimaryHeader)
			for _, sh := range lf.SecondaryHeaders {
				fmt.Printf("Secondary Header:\t%##v\n", sh)
			}
			fmt.Printf("\n")
		}

		//Check if is an image
		if lf.IsImageFile() && cli.NonText {
			fmt.Println(string(lf.Data))
		} else if lf.IsImageFile() && !cli.NonText {
			log.Infof("File (%s) contains binary data", path)
		} else {
			if lf.ContainsZipArchive() {
				log.Infof("File (%s) contains zip compressed data", path)
				if files, err := lf.UnzipToBuffer(); err == nil {
					for name, data := range files {
						fmt.Printf("%s:\n", name)
						fmt.Println(string(data))
						if len(files) > 1 {
							fmt.Printf("\n")
						}
					}
				} else {
					log.Errorf("Failed to decompress: %s", path, err.Error())
					return RC_FAILED_DECOMPRESSION
				}
			} else {
				fmt.Println(string(lf.Data))
			}
		}
	} else {
		log.Errorf("Could not read file %s: %s", path, err.Error())
		return RC_IO_ERROR
	}
	return RC_SUCCESS
}
