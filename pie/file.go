// Package pie provides a harness to apply file transforms.
package pie

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"launchpad.net/gommap"
	"os"
)

type file struct {
	Path string
	Info os.FileInfo
}

func (f file) Run(compiledInstructions []CompiledInstruction) error {
	file, err := os.Open(f.Path)
	if err != nil {
		return fmt.Errorf("error opening file %s: %s", f.Path, err)
	}
	mapped, err := gommap.Map(file.Fd(), gommap.PROT_READ, gommap.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("error mmaping file %s: %s", f.Path, err)
	}
	if isBinary(mapped) {
		mapped.UnsafeUnmap()
		file.Close()
		return nil
	}
	var out []byte
	changed := false
	for _, compiledInstruction := range compiledInstructions {
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
