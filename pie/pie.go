package pie

import (
	"fmt"
	"io/ioutil"
	"launchpad.net/gommap"
	"math"
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
}

type replaceAllCompiled struct {
	Target *regexp.Regexp
	Repl   []byte
}

func (r *replaceAllCompiled) Match(src []byte) bool {
	return r.Target.Match(src)
}

func (r *replaceAllCompiled) Apply(src []byte) []byte {
	return r.Target.ReplaceAll(src, r.Repl)
}

type ReplaceAll struct {
	Target string
	Repl   []byte
}

func (r *ReplaceAll) Compile() (CompiledInstruction, error) {
	re, err := regexp.Compile(r.Target)
	if err != nil {
		return nil, err
	}
	return &replaceAllCompiled{
		Target: re,
		Repl:   r.Repl,
	}, nil
}

func (r *Run) RunFile(compiledInstructions []CompiledInstruction, path string, info os.FileInfo) error {
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
			err = r.RunFile(compiledInstructions, i.Path, i.Info)
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
			batchSize += uint64(size)
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

	allLen := len(all)
	chunk := int(allLen / runtime.NumCPU())
	if allLen < 2 {
		r.runBatch(all, nil)
		return nil
	}

	h := 0
	wg := new(sync.WaitGroup)
	for i := 0; i < allLen; i += chunk {
		wg.Add(1)
		h = int(math.Min(float64(i+chunk), float64(allLen)))
		go r.runBatch(all[i:h], wg)
	}
	wg.Wait()

	return nil
}
