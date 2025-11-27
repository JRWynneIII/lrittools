package ziq

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/log"
	//"github.com/klauspost/compress/zstd"
	"github.com/DataDog/zstd"
)

type Ziq struct {
	path    string
	Header  ZiqHeader
	body    []complex64
	idx     int
	file    *os.File
	Done    bool
	decoder io.Reader
}

type ZiqHeader struct {
	Signature        string
	Compressed       bool
	BitsPerSample    uint8
	SampleRate       uint64
	AnnotationLength uint64
	Annotation       string
}

func Load(path string) *Ziq {
	log.Debugf("Opening ziq file: %s", path)
	if f, err := os.Open(path); err == nil {
		z := Ziq{
			path: path,
			file: f,
		}
		if err := z.parseHeader(); err != nil {
			log.Fatal(err)
			return nil
		}
		log.Debugf("Found ziq header %##v", z.Header)
		if z.Header.Compressed {
			log.Debugf("Ziq body is compressed...decompressing")
			z.decoder = zstd.NewReader(z.file)
		} else {
			log.Debugf("Ziq body is not compressed")
			z.decoder = io.NopCloser(f)
		}
		return &z
	} else {
		log.Errorf("Could not load ziq file %s", path, err)
		return nil
	}
}

func bytesToComplexSlice(bps uint8, input []byte, normalize bool) []complex64 {
	var output []complex64
	divisor := float32(1.0)
	if normalize {
		divisor = float32(127.0)
	}

	switch bps {
	case 8:
		// 8 bit ints
		for i := 0; i < len(input); i += 2 {
			r := float32(int8(input[i])) / divisor
			i := float32(int8(input[i+1])) / divisor
			output = append(output, complex(r, i))
		}
	//case 16:
	//	//16 bit ints
	//	var in16 []int16
	//	for i := 0; i < len(input); i += 2 {
	//		in16 = append(in16, int16(binary.LittleEndian.Uint16(input[i:i+1])))
	//	}

	//	for i := 0; i < len(in16); i += 2 {
	//		r := float32(in16[i]) / divisor
	//		i := float32(in16[i+1]) / divisor
	//		output = append(output, complex(r, i))
	//	}
	//case 32:
	//	//32 bit floats
	//	var in32 []float32
	//	for i := 0; i < len(input); i += 4 {
	//		in32 = append(in32, float32(binary.LittleEndian.Uint32(input[i:i+3])))
	//	}

	//	for i := 0; i < len(in32); i += 2 {
	//		r := in32[i] / divisor
	//		i := in32[i+1] / divisor
	//		output = append(output, complex(r, i))
	//	}
	default:
		log.Fatalf("Could not convert samples to complex64!")
	}

	return output
}

func (z *Ziq) parseHeader() error {
	h := ZiqHeader{}
	sig := make([]byte, 4)
	binary.Read(z.file, binary.LittleEndian, &sig)
	binary.Read(z.file, binary.LittleEndian, &h.Compressed)
	binary.Read(z.file, binary.LittleEndian, &h.BitsPerSample)
	binary.Read(z.file, binary.LittleEndian, &h.SampleRate)
	binary.Read(z.file, binary.LittleEndian, &h.AnnotationLength)
	h.Signature = string(sig)
	if h.Signature != "ZIQ_" {
		return fmt.Errorf("Invalid ziq file found; header does not contain ZIQ_")
	}

	if h.AnnotationLength > 0 {
		annotation := make([]byte, h.AnnotationLength)
		io.ReadFull(z.file, annotation)
		h.Annotation = string(annotation)
	}
	z.Header = h
	return nil
}

func (z *Ziq) GetNextChunk(size int) []complex64 {
	data := make([]byte, size*2)
	if _, err := io.ReadFull(z.decoder, data); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Fatalf("Could not read ZIQ data %s", err)
		} else {
			z.Done = true
		}
	}
	return bytesToComplexSlice(z.Header.BitsPerSample, data, true)
}
