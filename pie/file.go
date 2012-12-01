// Package pie provides a harness to apply file transforms.
package pie

import (
	"fmt"
	"io/ioutil"
	"launchpad.net/gommap"
	"os"
	"runtime"
)

type file struct {
	Path  string
	Info  os.FileInfo
	Debug bool
}

func (f *file) Run(compiledInstructions []CompiledInstruction) error {
	if f.Debug {
		fmt.Print("f")
	}
	file, err := os.Open(f.Path)
	if err != nil {
		return fmt.Errorf("error opening file %s: %s", f.Path, err)
	}
	mapped, err := gommap.Map(file.Fd(), gommap.PROT_READ, gommap.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("error mmaping file %s: %s", f.Path, err)
	}
	if isBinary(mapped) {
		if f.Debug {
			fmt.Print("s")
		}
		mapped.UnsafeUnmap()
		file.Close()
		return nil
	}
	var out []byte
	changed := false
	for _, compiledInstruction := range compiledInstructions {
		runtime.Gosched()
		if f.Debug {
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
			runtime.GC()
		} else {
			if !compiledInstruction.Match(out) {
				continue
			}
			out = compiledInstruction.Apply(out)
			runtime.GC()
		}
	}
	if changed {
		err := os.Remove(f.Path)
		if err != nil {
			return fmt.Errorf("error removing old file %s: %s", f.Path, err)
		}
		err = ioutil.WriteFile(f.Path, out, f.Info.Mode())
		if err != nil {
			return fmt.Errorf("error writing new file %s: %s", f.Path, err)
		}
	} else {
		mapped.UnsafeUnmap()
		file.Close()
	}
	return nil
}
