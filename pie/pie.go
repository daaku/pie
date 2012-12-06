// Package pie provides a harness to apply file transforms.
package pie

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"sync"

	"code.google.com/p/codesearch/index"
	"code.google.com/p/codesearch/regexp"
)

type Run struct {
	Index       *index.Index
	Instruction []Instruction
	NumWorkers  int
}

func (r *Run) numWorkers() int {
	if r.NumWorkers > 0 {
		return r.NumWorkers
	}
	return runtime.NumCPU() * 2
}

func (r *Run) compileInstruction() (CompiledInstructions, error) {
	compiledInstructions := make(CompiledInstructions, len(r.Instruction))
	var err error
	for i, instruction := range r.Instruction {
		compiledInstructions[i], err = instruction.Compile()
		if err != nil {
			return nil, err
		}
	}
	return compiledInstructions, nil
}

func (r *Run) processFile(path string, i CompiledInstructions) error {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file %s: %s", path, err)
	}

	buf, changed := i.Apply(buf)
	if changed {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("error stat old file %s: %s", path, err)
		}
		err = os.Remove(path)
		if err != nil {
			return fmt.Errorf("error removing old file %s: %s", path, err)
		}
		err = ioutil.WriteFile(path, buf, info.Mode())
		if err != nil {
			return fmt.Errorf("error writing new file %s: %s", path, err)
		}
	}
	return nil
}

func (r *Run) fileWorker(files chan string) {
	compiledInstructions, err := r.compileInstruction()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	for f := range files {
		if err = r.processFile(f, compiledInstructions); err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func (r *Run) Run() error {
	var wg sync.WaitGroup
	files := make(chan string, r.numWorkers()*2)
	for i := 0; i < r.numWorkers(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.fileWorker(files)
		}()
	}

	for _, instr := range r.Instruction {
		re, err := regexp.Compile("(?m)" + instr.MatchRegexpString())
		if err != nil {
			return err
		}
		q := index.RegexpQuery(re.Syntax)
		post := r.Index.PostingQuery(q)
		for _, fileid := range post {
			files <- r.Index.Name(fileid)
		}
	}
	close(files)
	wg.Wait()
	return nil
}
