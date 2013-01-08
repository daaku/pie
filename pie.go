package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"

	"code.google.com/p/codesearch/index"
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

func defaultIndexFile() string {
	base := ""
	if shm, err := os.Stat("/dev/shm"); err == nil && shm.IsDir() {
		base = "/dev/shm"
	}

	f, err := ioutil.TempFile(base, "pie")
	if err != nil {
		return ""
	}
	defer f.Close()
	return f.Name()
}

func defaultRoot() string {
	p, _ := os.Getwd()
	return p
}

func Main() error {
	var (
		goMaxProcs = flag.Int("gomaxprocs", runtime.NumCPU(), "gomaxprocs")
		ignoreRe   = flag.String("ignore", "", "file full path ignore regexp")
		filterRe   = flag.String("filter", "", "file full path filter regexp")
		cpuProfile = flag.String("cpuprofile", "", "write cpu profile to this file")
		inFile     = flag.String("input", "", "read instruction pairs from this file")
		indexFile  = flag.String("index", defaultIndexFile(), "default index file location")
		roots      = flag.String("root", defaultRoot(), "comma separated target paths")
		logSkip    = flag.Bool("logskip", false, "log skipped files")
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

	// parse replacement instructions
	r := &pie.Run{
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

	// make the index
	iw := index.Create(*indexFile)
	iw.LogSkip = *logSkip
	for _, arg := range strings.Split(*roots, ",") {
		filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
			if _, elem := filepath.Split(path); elem != "" {
				// Skip various temporary or "hidden" files or directories.
				if elem[0] == '.' || elem[0] == '#' || elem[0] == '~' || elem[len(elem)-1] == '~' {
					if *logSkip {
						log.Printf("%s: special, ignoring\n", path)
					}
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			if err != nil {
				log.Printf("%s: %s", path, err)
				return nil
			}
			if info != nil && info.Mode()&os.ModeType == 0 {
				iw.AddFile(path)
			}
			return nil
		})
	}
	iw.Flush()
	defer os.Remove(*indexFile)
	r.Index = index.Open(*indexFile)

	return r.Run()
}

func main() {
	err := Main()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
