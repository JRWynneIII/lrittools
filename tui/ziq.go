package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gdamore/tcell/v2"
	"github.com/jrwynneiii/ccsds_tools"
	"github.com/jrwynneiii/ccsds_tools/layers/datalink"
	"github.com/jrwynneiii/ccsds_tools/layers/physical"
	"github.com/jrwynneiii/ccsds_tools/layers/session"
	"github.com/jrwynneiii/ccsds_tools/lrit"
	"github.com/jrwynneiii/ccsds_tools/pipeline"
	"github.com/rivo/tview"
)

type TuiConf struct {
	EnableLogOutput     bool    `json:"enable_log_output" hcl:"enable_log_output"`
	RefreshMs           int     `json:"refresh_ms" hcl:"refresh_ms"`
	RsThresholdCritPct  float64 `json:"rs_threshold_crit_pct" hcl:"rs_threshold_crit_pct"`
	RsThresholdWarnPct  float64 `json:"rs_threshold_warn_pct" hcl:"rs_threshold_warn_pct"`
	VitThresholdCritPct float64 `json:"vit_threshold_crit_pct" hcl:"vit_threshold_crit_pct"`
	VitThresholdWarnPct float64 `json:"vit_threshold_warn_pct" hcl:"vit_threshold_warn_pct"`
}

var LogOut *tview.TextView

type LRITTableData struct {
	tview.TableContentReadOnly
}

var LRITTableList LRITList = LRITList{}

type LRITList struct {
	Files []*lrit.File
}

func (l *LRITTableData) GetRowCount() int {
	return len(LRITTableList.Files)
}

func (l *LRITTableData) GetColumnCount() int {
	return 1
}

func (l *LRITTableData) GetCell(row, column int) *tview.TableCell {
	color := "[lightskyblue]"
	if valid, _ := LRITTableList.Files[row].IsValid(); !valid {
		color = "[red]"
	}
	return tview.NewTableCell(fmt.Sprintf("VCID: %d File: %s%s", LRITTableList.Files[row].VCID, color, LRITTableList.Files[row].GetName()))
}

type LockTableData struct {
	tview.TableContentReadOnly
}

type Channel struct {
	ID                int
	Name              string
	NumPackets        int
	NumPacketsDropped int
}

type DecoderStats struct {
	FrameLock           bool
	TotalPackets        int
	TotalDroppedPackets int
	SNR                 float64
	AvgSNR              float64
	PeakSNR             float64
}

var overallDecoderStats = DecoderStats{
	false, 0, 0, 0.0, 0.0, 0.0,
}

var DecoderStatsMutex sync.RWMutex

func ReadOverallDecoderStats() DecoderStats {
	DecoderStatsMutex.RLock()
	defer DecoderStatsMutex.RUnlock()
	return overallDecoderStats
}

func WriteOverallDecoderStats(d DecoderStats) {
	DecoderStatsMutex.Lock()
	defer DecoderStatsMutex.Unlock()

	overallDecoderStats = d
}

func (l *LockTableData) GetRowCount() int {
	return 6
}

func (l *LockTableData) GetColumnCount() int {
	return 2
}

func (l *LockTableData) GetCell(row, column int) *tview.TableCell {
	switch row {
	case 0:
		if column == 0 {
			return tview.NewTableCell("Frame lock:")
		}

		color := tcell.ColorGreen
		if !ReadOverallDecoderStats().FrameLock {
			color = tcell.ColorRed
		}
		return tview.NewTableCell(fmt.Sprintf("%v", ReadOverallDecoderStats().FrameLock)).SetTextColor(color)
	case 1:
		if column == 0 {
			return tview.NewTableCell("Total Packets Rx'd:")
		}

		return tview.NewTableCell(fmt.Sprintf("%d", ReadOverallDecoderStats().TotalPackets))
	case 2:
		if column == 0 {
			return tview.NewTableCell("Total LRIT Files Rx'd:")
		}

		return tview.NewTableCell(fmt.Sprintf("%d", len(LRITTableList.Files)))
	case 3:
		if column == 0 {
			return tview.NewTableCell("SNR:")
		}

		snr := ReadOverallDecoderStats().SNR
		color := ""
		if snr < 1.0 {
			color = "[red]"
		} else {
			color = "[green]"
		}

		return tview.NewTableCell(fmt.Sprintf("%s%f", color, snr))
	case 4:
		if column == 0 {
			return tview.NewTableCell("Average SNR:")
		}

		snr := ReadOverallDecoderStats().AvgSNR
		color := ""
		if snr < 1.0 {
			color = "[red]"
		} else {
			color = "[green]"
		}

		return tview.NewTableCell(fmt.Sprintf("%s%f", color, snr))
	case 5:
		if column == 0 {
			return tview.NewTableCell("Peak SNR:")
		}

		snr := ReadOverallDecoderStats().PeakSNR
		color := ""
		if snr < 1.0 {
			color = "[red]"
		} else {
			color = "[green]"
		}

		return tview.NewTableCell(fmt.Sprintf("%s%f", color, snr))
	default:
		return tview.NewTableCell("ERROR")
	}
	return tview.NewTableCell("ERROR")
}

func StartZiq2LRITUI(pipeline *pipeline.Pipeline, decoder *datalink.Decoder, demodulator *physical.Demodulator, outputDir string, tuiConf TuiConf) {
	app := tview.NewApplication()

	LogOut = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	var logMutex sync.Mutex
	LogOut.SetChangedFunc(func() {
		logMutex.Lock()
		LogOut.ScrollToEnd()
		app.Draw()
		logMutex.Unlock()
	})

	LogOut.SetBorder(true).SetTitle("Log Output")
	log.SetOutput(LogOut)

	// Init our tables
	lockData := &LockTableData{}
	//channelStats := tview.NewTable().SetContent(channelData)
	lockTable := tview.NewTable().SetContent(lockData)
	//channelStats.SetSelectable(false, false).SetBorder(true).SetTitle("Per-Channel Stats")
	lockTable.SetSelectable(false, false).SetBorder(false)

	lritData := &LRITTableData{}
	lritTable := tview.NewTable().SetContent(lritData)
	lritTable.SetSelectable(true, true).SetBorder(false)

	lritBox := tview.NewFlex()
	lritBox.SetDirection(tview.FlexRow)
	lritBox.AddItem(lritTable, 0, 1, false)
	lritBox.SetTitle("LRIT Files Rx'd")
	lritBox.SetBorder(true)
	//lritBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
	//	switch event.Key() {
	//	case tcell.KeyDown:
	//	case tcell.KeyUp:
	//	case tcell.KeyEnter:
	//	}
	//})

	decoderStats := tview.NewFlex().SetDirection(tview.FlexRow)
	decoderStats.AddItem(lockTable, 0, 6, false)
	decoderStats.SetBorder(true)
	decoderStats.SetTitle("Decoder Status")

	// Init our page and columns
	page := tview.NewFlex().SetDirection(tview.FlexColumn)

	leftCol := tview.NewFlex().SetDirection(tview.FlexRow)
	leftCol.AddItem(lritBox, 0, 6, false)
	leftCol.AddItem(decoderStats, 0, 1, false)

	lritDescBox := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetScrollable(true)

	lritDescBox.SetChangedFunc(func() {
		app.Draw()
	})

	lritDescBox.SetBorder(true)
	lritDescBox.SetTitle("LRIT File Contents")

	rightCol := tview.NewFlex().SetDirection(tview.FlexRow)
	rightCol.AddItem(lritDescBox, 0, 4, true)
	if tuiConf.EnableLogOutput {
		rightCol.AddItem(LogOut, 0, 2, false)
	}
	page.AddItem(leftCol, 0, 2, false)
	page.AddItem(rightCol, 0, 5, false)

	sessionOut := pipeline.Layers[ccsds_tools.SessionLayer].(*session.LRITGen).GetOutput().(*chan *lrit.File)
	//descHasFocus := false

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			if lritTable.HasFocus() {
				app.SetFocus(lritDescBox)
			} else if lritDescBox.HasFocus() {
				app.SetFocus(LogOut)
			} else {
				app.SetFocus(lritTable)
			}
		case tcell.KeyEnter:
			selectedRow, _ := lritTable.GetSelection()
			lf := LRITTableList.Files[selectedRow]
			var ftype string
			switch lf.PrimaryHeader.FileType {
			case 0:
				ftype = "Image Data"
			case 1:
				ftype = "Service Message"
			case 2:
				ftype = "Alphanumeric Text"
			case 3:
				ftype = "Encryption Key Msg"
			case 128:
				ftype = "Meteorological Data"
			default:
				ftype = fmt.Sprintf("%d", lf.PrimaryHeader.FileType)
			}

			secondaryHeaderList := make(map[string]string)
			for _, sh := range lf.SecondaryHeaders {
				switch sh.(type) {
				case lrit.ImageStructureHeader:
					msg := `
		Type: %d
		Length: %d
		BitsPerPixel: %d
		Number of Columns: %d
		Number of Rows: %d
		Compression Flag: %d`
					secondaryHeaderList["Image Structure Header"] = fmt.Sprintf(msg, sh.(lrit.ImageStructureHeader).Type, sh.(lrit.ImageStructureHeader).Length, sh.(lrit.ImageStructureHeader).BitsPerPixel, sh.(lrit.ImageStructureHeader).NumCols, sh.(lrit.ImageStructureHeader).NumRows, sh.(lrit.ImageStructureHeader).IsCompressed)

				case lrit.ImageNavigationHeader:
					//TODO: Replace the projection name with a string to represent it
					msg := `
		Type: %d
		Length: %d
		Projection Name: %q
		Column Scaling Factor: %d
		Line Scaling Factor: %d
		Column Offset: %d
		Line Offset: %d`
					secondaryHeaderList["Image Navigation Header"] = fmt.Sprintf(msg, sh.(lrit.ImageNavigationHeader).Type, sh.(lrit.ImageNavigationHeader).Length, sh.(lrit.ImageNavigationHeader).ProjectionName, sh.(lrit.ImageNavigationHeader).ColumnScalingFactor, sh.(lrit.ImageNavigationHeader).LineScalingFactor, sh.(lrit.ImageNavigationHeader).ColumnOffset, sh.(lrit.ImageNavigationHeader).LineOffset)
				case lrit.ImageDataFunctionHeader:
					msg := `
		Type: %d
		Length: %d
		Data Definition: %q`
					secondaryHeaderList["Image Data Function Header"] = fmt.Sprintf(msg, sh.(lrit.ImageDataFunctionHeader).Type, sh.(lrit.ImageDataFunctionHeader).Length, sh.(lrit.ImageDataFunctionHeader).DataDefinition)
				case lrit.AnnotationHeader:
					msg := `
		Type: %d
		Length: %d
		Text: %q`
					secondaryHeaderList["Annotation Header"] = fmt.Sprintf(msg, sh.(lrit.AnnotationHeader).Type, sh.(lrit.AnnotationHeader).Length, sh.(lrit.AnnotationHeader).Text)
				case lrit.TimestampHeader:
					msg := `
		Type: %d
		Length: %d
		Time: %d`
					secondaryHeaderList["Timestamp Header"] = fmt.Sprintf(msg, sh.(lrit.TimestampHeader).Type, sh.(lrit.TimestampHeader).Length, sh.(lrit.TimestampHeader).Time)
				case lrit.AncillaryTextHeader:
					msg := `
		Type: %d
		Length: %d
		Text: %q`
					secondaryHeaderList["Ancillary Text Header"] = fmt.Sprintf(msg, sh.(lrit.AncillaryTextHeader).Type, sh.(lrit.AncillaryTextHeader).Length, sh.(lrit.AncillaryTextHeader).Text)
				case lrit.KeyHeader:
					msg := `
		Type: %d
		Length: %d`
					secondaryHeaderList["Key Header"] = fmt.Sprintf(msg, sh.(lrit.KeyHeader).Type, sh.(lrit.KeyHeader).Length)
				case lrit.SegmentIdentificationHeader:
					msg := `
		Type: %d
		Length: %d
		ImageIdentifier: %d
		SequenceNumber: %d
		StartColumn: %d
		StartLine: %d    
		MaxSegment:  %d   
		MaxColumn: %d   
		MaxRow: %d`

					secondaryHeaderList["Segment Identification Header"] = fmt.Sprintf(msg, sh.(lrit.SegmentIdentificationHeader).Type, sh.(lrit.SegmentIdentificationHeader).Length, sh.(lrit.SegmentIdentificationHeader).ImageIdentifier, sh.(lrit.SegmentIdentificationHeader).SequenceNumber, sh.(lrit.SegmentIdentificationHeader).StartColumn, sh.(lrit.SegmentIdentificationHeader).StartLine, sh.(lrit.SegmentIdentificationHeader).MaxSegment, sh.(lrit.SegmentIdentificationHeader).MaxColumn, sh.(lrit.SegmentIdentificationHeader).MaxRow)
				case lrit.NOAASpecificHeader:
					// TODO: Turn Product ID and sub product id to strings
					msg := `
		Type: %d
		Length: %d
		Agency: %q
		ProductID: %d
		ProductSubID: %d
		Parameter: %d    
		NOAASpecificCompression: %d`

					secondaryHeaderList["NOAA Specific Header"] = fmt.Sprintf(msg, sh.(lrit.NOAASpecificHeader).Type, sh.(lrit.NOAASpecificHeader).Length, sh.(lrit.NOAASpecificHeader).Agency, sh.(lrit.NOAASpecificHeader).ProductID, sh.(lrit.NOAASpecificHeader).ProductSubID, sh.(lrit.NOAASpecificHeader).Parameter, sh.(lrit.NOAASpecificHeader).NOAASpecificCompression)
				case lrit.HeaderStructureRecordHeader:
					msg := `
		Type: %d
		Length: %d
		Structure: %q`

					secondaryHeaderList["Header Structure Header"] = fmt.Sprintf(msg, sh.(lrit.HeaderStructureRecordHeader).Type, sh.(lrit.HeaderStructureRecordHeader).Length, sh.(lrit.HeaderStructureRecordHeader).Structure)
				case lrit.RiceCompressionHeader:
					msg := `
		Type: %d
		Length: %d
		Flags: %d
		PixelsPerBlock: %d
		ScanlinesPerPacket: %d`

					secondaryHeaderList["Rice Compression Header"] = fmt.Sprintf(msg, sh.(lrit.RiceCompressionHeader).Type, sh.(lrit.RiceCompressionHeader).Length, sh.(lrit.RiceCompressionHeader).Flags, sh.(lrit.RiceCompressionHeader).PixelsPerBlock, sh.(lrit.RiceCompressionHeader).ScanlinesPerPacket)
				}
			}

			msg := `VCID: %d
Version: %d, VCDU Version: %d
Primary Header:
	Type: %d
	Length: %d bytes
	File Type: %q
	All Header Length: %d bytes
	Data zone Length: %d bytes
	Actual length of data zone: %d bytes
Secondary Headers:
	%s
CRC passed: %t
Data: `
			var secondaryHeaderStr string
			for shname, sh := range secondaryHeaderList {
				secondaryHeaderStr = fmt.Sprintf("%s\n\t%s:%s", secondaryHeaderStr, shname, sh)
			}

			output := fmt.Sprintf(msg, lf.VCID, lf.Version, lf.VCDUVersion, lf.PrimaryHeader.Type, lf.PrimaryHeader.Length, ftype, lf.PrimaryHeader.AllHeaderLength, lf.PrimaryHeader.DataLength/8, len(lf.Data), secondaryHeaderStr, lf.CRCGood)
			dataOut := string(lf.Data)
			if lf.ContainsZipArchive() {
				dataOut = ""
				if unzipped, err := lf.UnzipToBuffer(); err == nil {
					for name, data := range unzipped {
						dataOut = fmt.Sprintf("%s%s:\n%s\n", dataOut, name, string(data))
					}
				} else {
					log.Errorf("Can't unzip LRIT file data!")
				}
			}
			output = strings.Join([]string{output, dataOut}, "\n")
			lritDescBox.Clear()
			fmt.Fprint(lritDescBox, output)
		}
		switch event.Rune() {
		case 'q':
			app.Stop()
		}
		return event
	})
	//Update all data in our UI.
	go func() {
		for {
			// Gather stats from decoder
			decoder.StatsMutex.RLock()
			frameLock := decoder.FrameLock
			totalFrames := decoder.TotalFramesProcessed

			decoder.StatsMutex.RUnlock()

			//Update signal plot data and SNR
			demodulator.FFTMutex.RLock()
			fft := demodulator.CurrentFFT
			snr := demodulator.CurrentSNR
			snravg := demodulator.AvgSNR
			snrpeak := demodulator.PeakSNR
			demodulator.FFTMutex.RUnlock()

			//Update decoder stats
			WriteOverallDecoderStats(DecoderStats{
				FrameLock:    frameLock,
				TotalPackets: totalFrames,
				SNR:          snr,
				AvgSNR:       snravg,
				PeakSNR:      snrpeak,
			})

			if len(fft) > 0 {
				var bins []float64
				for _, val := range fft {
					bins = append(bins, val)
				}
			}

			app.Draw()
			//Sleep half a second
			time.Sleep(time.Duration(tuiConf.RefreshMs) * time.Millisecond)

		}
	}()

	go func() {
		for {
			select {
			case f := <-*sessionOut:
				LRITTableList.Files = append(LRITTableList.Files, f)
				log.Infof("Got file %s from session layer", f.GetName())
				if len(outputDir) > 0 {
					f.WriteFile(outputDir)
				}
			}
		}
	}()

	// Start the TUI
	if err := app.SetRoot(page, true).EnableMouse(false).SetFocus(lritTable).Run(); err != nil {
		log.Fatalf("Could not start UI: %v", err)
	}
}
