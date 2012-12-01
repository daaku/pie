// Package pie provides a harness to apply file transforms.
package pie

import (
	"bytes"
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
	jobSize     uint64
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

func (r *Run) runBatch(items [][]*file, wg *sync.WaitGroup) {
	compiledInstructions, err := r.compileInstruction()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	for _, o := range items {
		for _, i := range o {
			err = i.Run(compiledInstructions)
			if err != nil {
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
	if wg != nil {
		wg.Done()
	}
}

func (r *Run) Run() error {
	const batchUnitTarget = 1048576 // 1 mb
	var all [][]*file
	var batch []*file
	var batchSize uint64
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
			size := uint64(info.Size())
			if size == 0 {
				return nil
			}
			if r.FileIgnore != nil && r.FileIgnore.MatchString(path) {
				return nil
			}
			if r.FileFilter != nil && !r.FileFilter.MatchString(path) {
				return nil
			}
			batchSize += size
			r.jobSize += size
			batch = append(batch, &file{path, info})
			if batchSize > batchUnitTarget {
				all = append(all, batch)
				batchSize = 0
				batch = nil
			}
			return nil
		})
	if batch != nil {
		all = append(all, batch)
	}

	allLen := len(all)
	chunk := int(allLen / runtime.NumCPU())
	if allLen < 2 || chunk == 0 {
		r.runBatch(all, nil)
		return nil
	}

	wg := new(sync.WaitGroup)
	h := 0
	for i := 0; i < allLen; i += chunk {
		wg.Add(1)
		h = min(i+chunk, allLen)
		go r.runBatch(all[i:h], wg)
	}
	wg.Wait()

	return nil
}

func min(x, y int) int {
	if x <= y {
		return x
	}
	return y
}

// based on buffer_is_binary in git
func isBinary(d []byte) bool {
	const firstFewBytes = 8000
	limit := min(firstFewBytes, len(d))
	return bytes.IndexByte(d[0:limit], byte(0)) != -1
}
