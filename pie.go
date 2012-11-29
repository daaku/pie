package main

import (
	"encoding/csv"
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
)

func addFromStdin(r *pie.Run) {
	reader := csv.NewReader(os.Stdin)
	reader.Comma = '\t'
	reader.Comment = '#'
	reader.FieldsPerRecord = 2
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	instructions, err := reader.ReadAll()
	if err != nil {
		panic(fmt.Sprintf("failed reading instructions from stdin: %s", err))
	}
	for _, instruction := range instructions {
		r.Instruction = append(r.Instruction, &pie.ReplaceAll{
			Target: instruction[0],
			Repl:   []byte(instruction[1]),
		})
	}
}

func addFromArgs(r *pie.Run, args []string) {
	argl := len(args)
	for x := 1; x < argl; x = x + 2 {
		r.Instruction = append(r.Instruction, &pie.ReplaceAll{
			Target: args[x],
			Repl:   []byte(args[x+1]),
		})
	}
}

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
		Root:  args[0],
		Debug: *debug,
	}
	if *ignoreRegexp != "" {
		r.FileIgnore = regexp.MustCompile(*ignoreRegexp)
	}
	if *filterRegexp != "" {
		r.FileFilter = regexp.MustCompile(*filterRegexp)
	}
	if argl < 3 {
		addFromStdin(r)
	} else {
		addFromArgs(r, args)
	}
	if len(r.Instruction) == 0 {
		fmt.Fprintf(os.Stderr, "error: no instructions provided on the command line or via stdin\n")
		flag.Usage()
		os.Exit(1)
	}
	err := r.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
