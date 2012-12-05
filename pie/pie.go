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
	Root          string
	Instruction   []Instruction
	FileIgnore    *regexp.Regexp
	FileFilter    *regexp.Regexp
	NumWorkers    int
	maxFileSize   int64
	totalFileSize int64
	file          []file
}

func (r *Run) numWorkers() int {
	if r.NumWorkers > 0 {
		return r.NumWorkers
	}
	return runtime.NumCPU() * 2
}

func (r *Run) approxBatchSize() int64 {
	const hardMin = int64(1024000) // 1mb
	size := r.totalFileSize / int64(r.numWorkers())
	if hardMin > size {
		return hardMin
	}
	return size
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

func (r *Run) worker(files []file) {
	compiledInstructions, err := r.compileInstruction()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	buf := make([]byte, r.maxFileSize)
	for _, f := range files {
		if err = f.Run(compiledInstructions, buf); err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func (r *Run) prepFiles() error {
	return filepath.Walk(
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
			size := info.Size()
			if size == 0 {
				return nil
			}
			if r.FileIgnore != nil && r.FileIgnore.MatchString(path) {
				return nil
			}
			if r.FileFilter != nil && !r.FileFilter.MatchString(path) {
				return nil
			}
			if size > r.maxFileSize {
				r.maxFileSize = size
			}
			r.totalFileSize += size
			r.file = append(r.file, file{path, info})
			return nil
		})
}

func (r *Run) Run() error {
	if err := r.prepFiles(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	approxBatchSize := r.approxBatchSize()
	fileLen := len(r.file)
	var batchSize int64
	var start, end int
	for end < fileLen {
		for end < fileLen && batchSize < approxBatchSize {
			batchSize += r.file[end].Info.Size()
			end++
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			r.worker(r.file[start:end])
		}(start, end)
		start = end
		batchSize = 0
	}
	wg.Wait()
	return nil
}
