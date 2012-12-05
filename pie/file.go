// Package pie provides a harness to apply file transforms.
package pie

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
)

type file struct {
	Path string
	Info os.FileInfo
}

func (f file) Run(i CompiledInstructions) error {
	file, err := os.Open(f.Path)
	if err != nil {
		return fmt.Errorf("error opening file %s: %s", f.Path, err)
	}
	defer file.Close()
	if isBinary(file) {
		return nil
	}
	out, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("error reading file %s: %s", f.Path, err)
	}
	out, changed := i.Apply(out)
	if changed {
		err := os.Remove(f.Path)
		if err != nil {
			return fmt.Errorf("error removing old file %s: %s", f.Path, err)
		}
		err = ioutil.WriteFile(f.Path, out, f.Info.Mode())
		if err != nil {
			return fmt.Errorf("error writing new file %s: %s", f.Path, err)
		}
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
func isBinary(f *os.File) bool {
	const firstFewBytes = 8000
	d := make([]byte, firstFewBytes)
	limit, _ := f.Read(d)
	f.Seek(0, 0)
	return bytes.IndexByte(d[0:limit], byte(0)) != -1
}
