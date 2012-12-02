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

func (r *Run) closer(work chan *os.File) {
	var err error
	for f := range work {
		if err = f.Close(); err != nil {
			fmt.Fprint(os.Stderr, "error closing file", err)
			os.Exit(1)
		}
	}
}

func (r *Run) worker(work chan file, closer chan *os.File) {
	compiledInstructions, err := r.compileInstruction()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	for f := range work {
		if err = f.Run(compiledInstructions, closer); err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func (r *Run) Run() error {
	work := make(chan file, runtime.NumCPU()*4)
	closer := make(chan *os.File, 10000)
	for i := 0; i < 2; i++ {
		// note currently we dont enforce closing of files and let the process die
		// without explicitly closing the file
		go r.closer(closer)
	}
	wg := new(sync.WaitGroup)
	for i := 0; i < runtime.NumCPU()*2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.worker(work, closer)
		}()
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
			work <- file{path, info}
			return nil
		})
	close(work)
	wg.Wait()
	close(closer)
	return nil
}
