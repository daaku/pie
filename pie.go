package main

import (
	"flag"
	"fmt"
	"github.com/daaku/pie/pie"
	"os"
	"regexp"
	"runtime"
	"runtime/pprof"

	_ "github.com/surma/stacksignal"
)

var (
	goMaxProcs   = flag.Int("gomaxprocs", runtime.NumCPU(), "gomaxprocs")
	ignoreRegexp = flag.String("ignore", "", "file full path ignore regexp")
	filterRegexp = flag.String("filter", "", "file full path filter regexp")
	cpuprofile   = flag.String("cpuprofile", "", "write cpu profile to file")
	debug        = flag.Bool("debug", false, "enable debug mode")
	listOnly     = flag.Bool("list-only", false, "only list target files")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"usage: %s <directory> [<target-regexp> <replace-pattern>]...\n",
			os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	runtime.GOMAXPROCS(*goMaxProcs)

	args := flag.Args()
	argl := len(args)
	if argl%2 == 0 {
		if argl != 1 {
			fmt.Fprintf(os.Stderr, "target/replace must be in pairs\n")
		}
		flag.Usage()
		os.Exit(1)
	}

	r := &pie.Run{
		Root:     args[0],
		Debug:    *debug,
		ListOnly: *listOnly,
	}
	if *ignoreRegexp != "" {
		r.FileIgnore = regexp.MustCompile(*ignoreRegexp)
	}
	if *filterRegexp != "" {
		r.FileFilter = regexp.MustCompile(*filterRegexp)
	}

	var err error
	if argl < 3 {
		r.Instruction, err = pie.InstructionFromReader(os.Stdin)
	} else {
		r.Instruction, err = pie.InstructionFromArgs(args)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
	if len(r.Instruction) == 0 {
		fmt.Fprintf(os.Stderr, "error: no instructions provided on the command line or via stdin\n")
		flag.Usage()
		os.Exit(1)
	}
	err = r.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
