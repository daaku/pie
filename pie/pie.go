// Package pie provides a harness to apply file transforms.
package pie

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
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
	FileFilter  string
	FileIgnore  string
	LogSkip     bool
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

func (r *Run) fileWorker(files chan string) error {
	compiledInstructions, err := r.compileInstruction()
	if err != nil {
		return fmt.Errorf("failed to compile instructions: %s", err)
	}

	var fileFilterRe, fileIgnoreRe *regexp.Regexp
	if r.FileFilter != "" {
		fileFilterRe, err = regexp.Compile(r.FileFilter)
		if err != nil {
			return fmt.Errorf("failed to compile file filter regexp: %s", err)
		}
	}
	if r.FileIgnore != "" {
		fileIgnoreRe, err = regexp.Compile(r.FileIgnore)
		if err != nil {
			return fmt.Errorf("failed to compile file ignore regexp: %s", err)
		}
	}

	for f := range files {
		if fileFilterRe != nil && fileFilterRe.MatchString(f, true, true) < 0 {
			if r.LogSkip {
				log.Printf("skipped %s because of filter", f)
			}
			continue
		}
		if fileIgnoreRe != nil && fileIgnoreRe.MatchString(f, true, true) > -1 {
			if r.LogSkip {
				log.Printf("skipped %s because of ignore", f)
			}
			continue
		}
		if err = r.processFile(f, compiledInstructions); err != nil {
			return fmt.Errorf("failed to process file %s: %s", f, err)
		}
	}
	return nil
}

func (r *Run) Run() (err error) {
	var wg sync.WaitGroup
	files := make(chan string, r.numWorkers()*2)
	for i := 0; i < r.numWorkers(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = r.fileWorker(files)
		}()
	}

	combined := bytes.NewBufferString("(?m)(")
	last := len(r.Instruction) - 1
	for i, instr := range r.Instruction {
		combined.WriteString(instr.MatchRegexpString())
		if i != last {
			combined.WriteString("|")
		}
	}
	combined.WriteString(")")

	re, err := regexp.Compile(combined.String())
	if err != nil {
		return fmt.Errorf(
			"failed to parse combined regexp %s: %s", combined.String(), err)
	}
	q := index.RegexpQuery(re.Syntax)
	post := r.Index.PostingQuery(q)
	for _, fileid := range post {
		files <- r.Index.Name(fileid)
	}
	close(files)
	wg.Wait()
	return
}
