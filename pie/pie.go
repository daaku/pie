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

type Rule interface {
	Match(src []byte) bool
	Apply(src []byte) []byte
}

type Run struct {
	Root       string
	Rule       []Rule
	BatchSize  uint64
	FileIgnore *regexp.Regexp
	FileFilter *regexp.Regexp
	Debug      bool
}

type ReplaceAll struct {
	Target *regexp.Regexp
	Repl   []byte
}

func (r *ReplaceAll) Match(src []byte) bool {
	return r.Target.Match(src)
}

func (r *ReplaceAll) Apply(src []byte) []byte {
	return r.Target.ReplaceAll(src, r.Repl)
}

type pathFileInfo struct {
	Path string
	Info os.FileInfo
}

func (r *Run) RunFile(path string, info os.FileInfo) error {
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
	for _, rule := range r.Rule {
		if r.Debug {
			fmt.Print("r")
		}
		// optimize for no changes to just work with mmaped file
		if !changed {
			if !rule.Match(mapped) {
				continue
			}
			out = rule.Apply(mapped)
			changed = true
			mapped.UnsafeUnmap()
			file.Close()
		} else {
			if !rule.Match(out) {
				continue
			}
			out = rule.Apply(out)
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

func (r *Run) runBatch(items [][]*pathFileInfo, wg *sync.WaitGroup) {
	if r.Debug {
		fmt.Print("b")
	}
	var err error
	for _, o := range items {
		for _, i := range o {
			err = r.RunFile(i.Path, i.Info)
			if err != nil {
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
	wg.Done()
}

func (r *Run) Run() error {
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
			if batchSize > r.BatchSize {
				all = append(all, batch)
				batchSize = 0
				batch = nil
			}
			return nil
		})
	if batch != nil {
		all = append(all, batch)
	}

	wg := new(sync.WaitGroup)
	allLen := len(all)
	chunk := int(allLen / runtime.NumCPU())
	for i := 0; i < allLen; i += chunk {
		wg.Add(1)
		h := int(math.Min(float64(i+chunk), float64(allLen)))
		go r.runBatch(all[i:h], wg)
	}
	wg.Wait()

	return nil
}
