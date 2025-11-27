package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"os"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/log"
	"github.com/jrwynneiii/lrittools/ziq"
)

var cli struct {
	Verbose    bool   `help:"Prints debug output by default"`
	File       string `arg:"" help:"Path to a ziq IQ file"`
	OutputFile string `arg:"" help:"File path to output file"`
	Normalize  bool   `help:"Writes normalized complex64 values to output" default:"true"`
}

func main() {
	_ = kong.Parse(&cli)
	if cli.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	output := ziq.Load(cli.File)
	if output == nil {
		log.Fatalf("File %s is not a valid ZIQ file", cli.File)
		os.Exit(1)
	}

	var outputfile *os.File
	var err error
	if outputfile, err = os.Open(cli.OutputFile); !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("Output file exists! Cowardly not overwriting file...")
	} else if errors.Is(err, os.ErrNotExist) {
		if outputfile, err = os.Create(cli.OutputFile); err != nil {
			log.Fatalf("Could not open output file! %s", err.Error())
		}
	}

	divisor := float32(127.0)
	if cli.Normalize {
		divisor = float32(1.0)
	}

	log.Debugf("Writing output file...")
	for !output.Done {
		buf := new(bytes.Buffer)
		samples := output.GetNextChunk(66560)

		for _, val := range samples {
			if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(real(val)/divisor)); err != nil {
				log.Fatalf("Error writing IQ file! %s", err.Error())
			}
			if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(imag(val)/divisor)); err != nil {
				log.Fatalf("Error writing IQ file! %s", err.Error())
			}
		}

		log.Debugf("Wrote chunk of %d samples to output file (%d bytes)", len(samples), len(buf.Bytes()))
		if _, err = outputfile.Write(buf.Bytes()); err != nil {
			log.Fatalf("Could not write to output file! %s", cli.OutputFile)
		}
	}
}
