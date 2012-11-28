package pie

import (
	"code.google.com/p/rog-go/parallel"
	"fmt"
	"io/ioutil"
	"launchpad.net/gommap"
	"os"
	"path/filepath"
	"regexp"
)

type Rule interface {
	Match(src []byte) bool
	Apply(src []byte) []byte
}

type Run struct {
	Root       string
	Rule       []Rule
	Parallel   int
	BatchSize  int64
	FileIgnore *regexp.Regexp
	FileFilter *regexp.Regexp
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
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening file %s: %s", path, err)
	}
	defer file.Close()
	mapped, err := gommap.Map(file.Fd(), gommap.PROT_READ, gommap.MAP_PRIVATE)
	if err != nil {
		return fmt.Errorf("error mmaping file %s: %s", path, err)
	}
	defer mapped.UnsafeUnmap()
	var out []byte
	changed := false
	for _, rule := range r.Rule {
		// optimize for no changes to just work with mmaped file
		if !changed {
			if !rule.Match(mapped) {
				continue
			}
			out = rule.Apply(mapped)
			changed = true
		} else {
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
	}
	return nil
}

func (r *Run) runBatch(items []pathFileInfo) func() error {
	return func() error {
		var err error
		for _, i := range items {
			err = r.RunFile(i.Path, i.Info)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func (r *Run) Run() error {
	run := parallel.NewRun(r.Parallel)
	var batch []pathFileInfo
	var batchSize int64
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
			batchSize += size
			batch = append(batch, pathFileInfo{path, info})
			if batchSize > r.BatchSize {
				run.Do(r.runBatch(batch))
				batchSize = 0
				batch = nil
			}
			return nil
		})
	if batch != nil {
		run.Do(r.runBatch(batch))
	}
	return run.Wait()
}
