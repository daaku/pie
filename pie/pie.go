package pie

import (
	"fmt"
	"io/ioutil"
	"launchpad.net/gommap"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
)

// Instructions describe the modification. This happens once for each parallel logical
// thread of execution.
type Instruction interface {
	Compile() (CompiledInstruction, error)
}

type CompiledInstruction interface {
	Match(src []byte) bool
	Apply(src []byte) []byte
}

type Run struct {
	Root        string
	Instruction []Instruction
	FileIgnore  *regexp.Regexp
	FileFilter  *regexp.Regexp
	Debug       bool
	jobSize     uint64
}

func (r *Run) runFile(compiledInstructions []CompiledInstruction, path string, info os.FileInfo) error {
	if r.Debug {
		fmt.Print("f")
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening file %s: %s", path, err)
	}
	mapped, err := gommap.Map(file.Fd(), gommap.PROT_READ, gommap.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("error mmaping file %s: %s", path, err)
	}
	var out []byte
	changed := false
	for _, compiledInstruction := range compiledInstructions {
		runtime.Gosched()
		if r.Debug {
			fmt.Print("r")
		}
		// optimize for no changes to just work with mmaped file
		if !changed {
			if !compiledInstruction.Match(mapped) {
				continue
			}
			out = compiledInstruction.Apply(mapped)
			changed = true
			mapped.UnsafeUnmap()
			file.Close()
		} else {
			if !compiledInstruction.Match(out) {
				continue
			}
			out = compiledInstruction.Apply(out)
		}
	}
	if changed {
		err := os.Remove(path)
		if err != nil {
			return fmt.Errorf("error removing old file %s: %s", path, err)
		}
		err = ioutil.WriteFile(path, out, info.Mode())
		if err != nil {
			return fmt.Errorf("error writing new file %s: %s", path, err)
		}
	} else {
		mapped.UnsafeUnmap()
		file.Close()
	}
	return nil
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

type pathFileInfo struct {
	Path string
	Info os.FileInfo
}

func (r *Run) runBatch(items [][]*pathFileInfo, wg *sync.WaitGroup) {
	if r.Debug {
		fmt.Print("b")
	}
	compiledInstructions, err := r.compileInstruction()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	for _, o := range items {
		for _, i := range o {
			err = r.runFile(compiledInstructions, i.Path, i.Info)
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
	var all [][]*pathFileInfo
	var batch []*pathFileInfo
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
			batch = append(batch, &pathFileInfo{path, info})
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
	if r.Debug {
		fmt.Printf("job size: %d", r.jobSize)
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
