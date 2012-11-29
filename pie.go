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
)

var (
	goMaxProcs   = flag.Int("gomaxprocs", runtime.NumCPU(), "gomaxprocs")
	parallelSize = flag.Int("parallel", runtime.NumCPU(), "number of goroutines")
	ignoreRegexp = flag.String("ignore", "", "file full path ignore regexp")
	filterRegexp = flag.String("filter", "", "file full path filter regexp")
	batchSize    = flag.Int64("batch-size", 104857600, "approximate batch size in bytes")
	cpuprofile   = flag.String("cpuprofile", "", "write cpu profile to file")
)

func addFromStdin(r *pie.Run) {
	reader := csv.NewReader(os.Stdin)
	reader.Comma = '\t'
	reader.Comment = '#'
	reader.FieldsPerRecord = 2
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	rules, err := reader.ReadAll()
	if err != nil {
		panic(fmt.Sprintf("failed reading rules from stdin: %s", err))
	}
	for _, rule := range rules {
		r.Rule = append(r.Rule, &pie.ReplaceAll{
			Target: regexp.MustCompile(rule[0]),
			Repl:   []byte(rule[1]),
		})
	}
}

func addFromArgs(r *pie.Run, args []string) {
	argl := len(args)
	for x := 1; x < argl; x = x + 2 {
		r.Rule = append(r.Rule, &pie.ReplaceAll{
			Target: regexp.MustCompile(args[x]),
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
		Root:     args[0],
		Parallel: *parallelSize,
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
	if len(r.Rule) == 0 {
		fmt.Fprintf(os.Stderr, "error: no rules provided on the command line or via stdin\n")
		flag.Usage()
		os.Exit(1)
	}
	err := r.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
