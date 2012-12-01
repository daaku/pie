// Package pie provides a harness to apply file transforms.
package pie

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
)

type Run struct {
	Root        string
	Instruction []Instruction
	FileIgnore  *regexp.Regexp
	FileFilter  *regexp.Regexp
}

func (r *Run) compileInstruction() ([]CompiledInstruction, error) {
	compiledInstructions := make([]CompiledInstruction, len(r.Instruction))
	var err error
	for i, instruction := range r.Instruction {
		compiledInstructions[i], err = instruction.Compile()
		if err != nil {
			return nil, err
		}
	}
	return compiledInstructions, nil
}

func (r *Run) worker(work chan file, wg *sync.WaitGroup) {
	compiledInstructions, err := r.compileInstruction()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	for {
		f, ok := <-work
		if !ok {
			return
		}
		err = f.Run(compiledInstructions)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		wg.Done()
	}
}

func (r *Run) Run() error {
	work := make(chan file, 500)
	wg := new(sync.WaitGroup)
	for i := 0; i < runtime.NumCPU(); i++ {
		go r.worker(work, wg)
	}
	filepath.Walk(
		r.Root,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			if info.IsDir() {
				return nil
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return nil
			}
			if info.Size() == 0 {
				return nil
			}
			if r.FileIgnore != nil && r.FileIgnore.MatchString(path) {
				return nil
			}
			if r.FileFilter != nil && !r.FileFilter.MatchString(path) {
				return nil
			}
			wg.Add(1)
			work <- file{path, info}
			return nil
		})
	wg.Wait()
	close(work)
	return nil
}
