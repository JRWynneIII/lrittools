package main

import (
	"os"
	"sync"
	"time"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/log"
	"github.com/jrwynneiii/ccsds_tools"
	"github.com/jrwynneiii/ccsds_tools/layers/datalink"
	"github.com/jrwynneiii/ccsds_tools/layers/physical"
	"github.com/jrwynneiii/ccsds_tools/layers/session"
	"github.com/jrwynneiii/ccsds_tools/lrit"
	"github.com/jrwynneiii/ccsds_tools/pipeline"
	"github.com/jrwynneiii/lrittools/tui"
	"github.com/jrwynneiii/lrittools/ziq"
)

var cli struct {
	Verbose    bool    `help:"Prints debug output by default"`
	File       string  `help:"Path to a ziq IQ file"`
	OutputDir  string  `help:"Directory to output LRIT files"`
	NoTui      bool    `help:"Disable the TUI and just use the cli"`
	SampleRate float64 `help:"Sample rate of input ZIQ file"`
}

var options map[string]any = map[string]any{
	"agc.gain":                      1.0,
	"agc.max_gain":                  4000.0,
	"agc.rate":                      0.01,
	"agc.reference":                 0.5,
	"clockrecovery.alpha":           0.0037,
	"clockrecovery.mu":              0.5,
	"clockrecovery.omega_limit":     0.005,
	"tui.enable_log_output":         true,
	"tui.refresh_ms":                500,
	"tui.rs_threshold_crit_pct":     5.0,
	"tui.rs_threshold_warn_pct":     2.0,
	"tui.vit_threshold_crit_pct":    5.0,
	"tui.vit_threshold_warn_pct":    3.0,
	"viterbi.max_errors":            500,
	"xrit.chunk_size":               66560,
	"xrit.decimation_factor":        1,
	"xrit.do_fft":                   true,
	"xrit.lowpass_transition_width": 200000.0,
	"xrit.pll_alpha":                0.001,
	"xrit.rrc_alpha":                0.3,
	"xrit.rrc_taps":                 31,
	"xrit.symbol_rate":              927000.0,
	"xritframe.frame_size":          1024,
	"xritframe.last_frame_size":     8,
}

func main() {
	_ = kong.Parse(&cli)
	if cli.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	options["radio.sample_rate"] = 2048000.0
	if cli.SampleRate != 0 {
		options["radio.sample_rate"] = cli.SampleRate
	}

	xritChunkSize := options["xrit.chunk_size"].(int)
	log.Debugf("Starting CCSDS pipeline")

	pipeline := pipeline.NewWithOptionsMap(options)
	pipeline.Register(ccsds_tools.PhysicalLayer)
	pipeline.Register(ccsds_tools.DataLinkLayer)
	pipeline.Register(ccsds_tools.TransportLayer)
	pipeline.Register(ccsds_tools.SessionLayer)

	sessionOut := pipeline.Layers[ccsds_tools.SessionLayer].(*session.LRITGen).GetOutput().(*chan *lrit.File)
	samplesIn := pipeline.Layers[ccsds_tools.PhysicalLayer].GetInput().(*chan []complex64)
	demod := pipeline.Layers[ccsds_tools.PhysicalLayer].(*physical.Demodulator)
	decode := pipeline.Layers[ccsds_tools.DataLinkLayer].(*datalink.Decoder)

	output := ziq.Load(cli.File)
	if output == nil {
		log.Fatalf("File %s is not a valid ZIQ file", cli.File)
		os.Exit(1)
	}
	log.Debugf("ZIQ Header: %##v", output.Header)
	go func() {
		for !output.Done {
			chunk := output.GetNextChunk(int(xritChunkSize))
			*samplesIn <- chunk
		}
		log.Infof("Finished reading ZIQ file")
	}()

	pipeline.Start()

	defer pipeline.Destroy()

	var wg sync.WaitGroup
	if cli.NoTui {
		go func() {
			for {
				select {
				case f := <-*sessionOut:
					log.Infof("Got LRIT file (Version: %d, VCDUVersion: %d) with primary header: %##v, and secondary headers: %##v", f.Version, f.VCDUVersion, f.PrimaryHeader, f.SecondaryHeaders)
					f.WriteFile(cli.OutputDir)
				default:
					time.Sleep(50 * time.Millisecond)
				}
			}
		}()
	}

	go func() {
		wg.Add(1)
		emptyCounter := 0
		for {
			time.Sleep(5 * time.Second)
			if len(*samplesIn) > 0 {
				log.Infof("Locked: %v\tCurrent SNR: %f\tDecoded Packets: %v\tDropped packets: %v", decode.FrameLock, demod.CurrentSNR, decode.RxPacketsPerChannel, decode.DroppedPacketsPerChannel)
				log.Infof("Buffers: samplesIn: %d, transportOut: %d", len(*samplesIn), len(*sessionOut))
			}

			if len(*samplesIn) == 0 {
				emptyCounter += 1
				if emptyCounter > 2 {
					pipeline.Flush()
					pipeline.Destroy()
					break
				}
			}
		}
		wg.Done()
	}()

	if !cli.NoTui {
		tuiDef := tui.TuiConf{
			RefreshMs:           options["tui.refresh_ms"].(int),
			RsThresholdWarnPct:  options["tui.rs_threshold_warn_pct"].(float64),
			RsThresholdCritPct:  options["tui.rs_threshold_crit_pct"].(float64),
			VitThresholdWarnPct: options["tui.vit_threshold_warn_pct"].(float64),
			VitThresholdCritPct: options["tui.vit_threshold_crit_pct"].(float64),
			EnableLogOutput:     options["tui.enable_log_output"].(bool),
		}

		tui.StartZiq2LRITUI(pipeline, decode, demod, cli.OutputDir, tuiDef)
	}

	time.Sleep(1 * time.Second)
	if cli.NoTui {
		wg.Wait()
	}
}
