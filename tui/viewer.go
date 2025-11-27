package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/gdamore/tcell/v2"
	"github.com/jrwynneiii/ccsds_tools/lrit"
	"github.com/rivo/tview"
)

type LRITFilesData struct {
	tview.TableContentReadOnly
}

var LRITFilesList LRITNameList = LRITNameList{}

type LRITNameList struct {
	Files []string
}

func (l *LRITFilesData) GetRowCount() int {
	return len(LRITFilesList.Files)
}

func (l *LRITFilesData) GetColumnCount() int {
	return 1
}

func (l *LRITFilesData) GetCell(row, column int) *tview.TableCell {
	color := "[lightskyblue]"
	return tview.NewTableCell(fmt.Sprintf("%s%s", color, LRITFilesList.Files[row]))
}

func StartLRITViewerUI(files []string, dir string) {
	app := tview.NewApplication()
	LRITFilesList.Files = files

	lritData := &LRITFilesData{}
	lritTable := tview.NewTable().SetContent(lritData)
	lritTable.SetSelectable(true, true).SetBorder(false)

	lritBox := tview.NewFlex()
	lritBox.SetDirection(tview.FlexRow)
	lritBox.AddItem(lritTable, 0, 1, false)
	lritBox.SetTitle("LRIT Files Rx'd")
	lritBox.SetBorder(true)

	// Init our page and columns
	page := tview.NewFlex().SetDirection(tview.FlexColumn)

	leftCol := tview.NewFlex().SetDirection(tview.FlexRow)
	leftCol.AddItem(lritBox, 0, 6, false)

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

	page.AddItem(leftCol, 0, 2, false)
	page.AddItem(rightCol, 0, 5, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			if lritTable.HasFocus() {
				app.SetFocus(lritDescBox)
			} else {
				app.SetFocus(lritTable)
			}
		case tcell.KeyEnter:
			selectedRow, _ := lritTable.GetSelection()
			name := LRITFilesList.Files[selectedRow]
			lf, err := lrit.NewExistingFile(filepath.Join(dir, name))
			if err != nil {
				lritDescBox.Clear()
				fmt.Fprint(lritDescBox, err.Error())
				break
			}
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
	//go func() {
	//	for {
	//		app.Draw()
	//		//Sleep half a second
	//		time.Sleep(50 * time.Millisecond)

	//	}
	//}()

	// Start the TUI
	if err := app.SetRoot(page, true).EnableMouse(false).SetFocus(lritTable).Run(); err != nil {
		log.Fatalf("Could not start UI: %v", err)
	}
}
