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

where path... is a list of directories or individual files to be included in the index.
If no index exists, this command creates one.  If an index already exists, cindex
overwrites it.  Run cindex -help for more.

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
		goMaxProcs   = flag.Int("gomaxprocs", runtime.NumCPU(), "gomaxprocs")
		ignoreRegexp = flag.String("ignore", "", "file full path ignore regexp")
		filterRegexp = flag.String("filter", "", "file full path filter regexp")
		cpuProfile   = flag.String("cpuprofile", "", "write cpu profile to this file")
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

	argl := len(args)
	if argl%2 == 1 {
		if argl != 0 {
			fmt.Fprintf(os.Stderr, "target/replace must be in pairs\n")
		}
		flag.Usage()
		os.Exit(1)
	}

	ix := index.Open(index.File())
	r := &pie.Run{
		Index:      ix,
		FileFilter: *filterRegexp,
		FileIgnore: *ignoreRegexp,
	}
	var err error
	if argl < 2 {
		r.Instruction, err = pie.InstructionFromReader(os.Stdin)
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
		fmt.Fprintf(os.Stderr, "error: no instructions provided on the command line or via stdin\n")
		flag.Usage()
		os.Exit(1)
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
