package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"code.google.com/p/codesearch/index"
	"github.com/daaku/pie/pie"
)

var usageMessage = `usage: %s [<target-regexp> <replace-pattern>]...

pie relies on the existence of an up-to-date index created ahead of time.
To build or rebuild the index that pie uses, run:

	cindex path...

where path... is a list of directories or individual files to be included in
the index.  If no index exists, this command creates one.  If an index already
exists, cindex overwrites it.  Run cindex -help for more.

pie uses the index stored in $CSEARCHINDEX or, if that variable is unset or
empty, $HOME/.csearchindex.

Options
`

func usage() {
	fmt.Fprintf(os.Stderr, usageMessage, os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func Main() error {
	var (
		goMaxProcs = flag.Int("gomaxprocs", runtime.NumCPU(), "gomaxprocs")
		ignoreRe   = flag.String("ignore", "", "file full path ignore regexp")
		filterRe   = flag.String("filter", "", "file full path filter regexp")
		cpuProfile = flag.String("cpuprofile", "", "write cpu profile to this file")
		inFile     = flag.String("input", "", "read instruction pairs from this file")
	)

	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	runtime.GOMAXPROCS(*goMaxProcs)

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	ix := index.Open(index.File())
	r := &pie.Run{
		Index:      ix,
		FileFilter: *filterRe,
		FileIgnore: *ignoreRe,
	}
	var err error
	if *inFile != "" {
		f, err := os.Open(*inFile)
		if err != nil {
			return err
		}
		r.Instruction, err = pie.InstructionFromReader(f)
		if err != nil {
			return err
		}
	} else {
		r.Instruction, err = pie.InstructionFromArgs(args)
		if err != nil {
			return err
		}
	}
	if len(r.Instruction) == 0 {
		flag.Usage()
	}
	return r.Run()
}

func main() {
	err := Main()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
