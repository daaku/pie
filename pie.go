package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"

	"github.com/daaku/pie/pie"
)

var usageMessage = `usage: %s [<target-regexp> <replace-pattern>]...

Options
`

func usage() {
	fmt.Fprintf(os.Stderr, usageMessage, os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

func defaultRoot() string {
	p, _ := os.Getwd()
	return p
}

func Main() error {
	var (
		goMaxProcs = flag.Int("gomaxprocs", runtime.NumCPU(), "gomaxprocs")
		numWorkers = flag.Int("num-workers", runtime.NumCPU()*2, "number of workers")
		excludeRe  = flag.String("ignore", "", "full file path exclude regexp")
		includeRe  = flag.String("filter", "", "full file path include regexp")
		cpuProfile = flag.String("cpuprofile", "", "write cpu profile to this file")
		inFile     = flag.String("input", "", "read instruction pairs from this file")
		root       = flag.String("root", defaultRoot(), "comma separated target paths")
		verbose    = flag.Bool("verbose", false, "verbose logging")

		err error
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

	run := &pie.Run{
		NumWorkers: *numWorkers,
		Verbose:    *verbose,
		Root:       strings.Split(*root, ","),
	}

	// search & replace instructions
	var instruction *pie.Instruction
	if *inFile != "" {
		instruction, err = pie.InstructionFromFile(*inFile)
	} else {
		instruction, err = pie.InstructionFromArgs(args)
	}
	if err != nil {
		return err
	}
	run.ContentMatcher = instruction
	run.Transformer = instruction

	// include/exclude regexps
	if *includeRe != "" {
		if run.PathInclude, err = pie.RegExpPathMatcher(*includeRe); err != nil {
			return err
		}
	}
	if *excludeRe != "" {
		if run.PathExclude, err = pie.RegExpPathMatcher(*excludeRe); err != nil {
			return err
		}
	}

	return run.Go()
}

func main() {
	err := Main()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
