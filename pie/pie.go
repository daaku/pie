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

func (r *Run) Run() error {
	run := parallel.NewRun(r.Parallel)
	filepath.Walk(
		r.Root,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Base(path) == ".git" {
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
			run.Do(func() error { return r.RunFile(path, info) })
			return nil
		})
	return run.Wait()
}
