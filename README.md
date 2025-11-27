# LRITTools

This is a collection of simple tools used to inspect or process LRIT and ZIQ files.  

## `lritcat`
`cat` but for LRIT files. Extracts primary and secondary LRIT headers from files, and dumps the data section of the file. Files that contain compressed non-image data will be decompressed on the fly. 

```
Usage: lritcat <paths> ... [flags]

Arguments:
  <paths> ...    Path to LRIT file

Flags:
  -h, --help             Show context-sensitive help.
      --verbose          Prints debug output by default
      --list-headers     Print the available secondary headers
      --no-decompress    Disable zip decompression
      --non-text         Displays non-text data; This might corrupt your terminal environment if not piped!
```


To install: `go install github.com/jrwynneiii/lrittools/cmd/lritcat`

## `lritviewer`
A simple TUI LRIT file browser tool. When dealing with a large number of LRIT files, this can be useful for comparison. Use `<TAB>` to navigate between panes, and the arrow keys, or vim movements to navigate within a pane

```
Usage: lritviewer <dir>

Arguments:
  <dir>    Path to directory containing LRIT files

Flags:
  -h, --help    Show context-sensitive help.
```

To install: `go install github.com/jrwynneiii/lrittools/cmd/lritviewer`

## `ziq2lrit`
[SatDump](https://github.com/SatDump/SatDump) can output a baseband IQ recording in a custom, zstd compressed file type that it calls `ziq`. Since this is custom to SatDump, there aren't many tools for processing this kind of data. `ziq2lrit` will read in a ziq baseband file and demodulate a GOES HRIT/LRIT signal, and output the resulting LRIT files. By default, it will open a TUI so that you can more easily observe the processing, but this can be disabled. (NOTE: at time of writing, only cs8 ziq data is supported!)
```
Usage: ziq2lrit [flags]

Flags:
  -h, --help                    Show context-sensitive help.
      --verbose                 Prints debug output by default
      --file=STRING             Path to a ziq IQ file
      --output-dir=STRING       Directory to output LRIT files
      --no-tui                  Disable the TUI and just use the cli
      --sample-rate=FLOAT-64    Sample rate of input ZIQ file
```

To install: `go install github.com/jrwynneiii/lrittools/cmd/ziq2lrit`

## `unziq`
`unziq` will process and decompress SatDump's `ziq` baseband files and dump the IQ stream into another file, so that other tools can more easily process it (NOTE: at time of writing, only cs8 ziq data is supported, and output IQ stream will be of type CF32!).
```
Usage: unziq <file> <output-file> [flags]

Arguments:
  <file>           Path to a ziq IQ file
  <output-file>    File path to output file

Flags:
  -h, --help         Show context-sensitive help.
      --verbose      Prints debug output by default
      --normalize    Writes normalized complex64 values to output
```

To install: `go install github.com/jrwynneiii/lrittools/cmd/unziq`
