package main

import (
	"flag"
	"fmt"
	"github.com/daaku/pie/pie"
	"os"
	"regexp"
	"runtime"
)

var (
	goMaxProcs   = flag.Int("gomaxprocs", runtime.NumCPU(), "gomaxprocs")
	parallelSize = flag.Int("parallel", runtime.NumCPU(), "number of goroutines")
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
	runtime.GOMAXPROCS(*goMaxProcs)
	args := flag.Args()
	argl := len(args)
	if argl < 3 || argl%2 == 0 {
		flag.Usage()
		os.Exit(1)
	}
	r := &pie.Run{
		Root:     args[0],
		Parallel: *parallelSize,
	}
	for x := 1; x < argl; x = x + 2 {
		r.Rule = append(r.Rule, &pie.ReplaceAll{
			Target: regexp.MustCompile(args[x]),
			Repl:   []byte(args[x+1]),
		})
	}
	err := r.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err)
		os.Exit(1)
	}
}
