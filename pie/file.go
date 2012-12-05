// Package pie provides a harness to apply file transforms.
package pie

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

type file struct {
	Path string
	Info os.FileInfo
}

func (f file) Run(i CompiledInstructions, buf []byte) error {
	file, err := os.Open(f.Path)
	if err != nil {
		return fmt.Errorf("error opening file %s: %s", f.Path, err)
	}
	defer file.Close()

	const firstFewBytes = 8000
	bufl := len(buf)
	limit, err := file.Read(buf[0:min(bufl, firstFewBytes)])
	if err != nil && err != io.EOF {
		return err
	}
	if bytes.IndexByte(buf[0:limit], byte(0)) != -1 {
		return nil
	}
	if err == nil {
		rest, err := file.Read(buf[limit:bufl])
		if err != nil && err != io.EOF {
			return err
		}
		limit += rest
	}

	buf, changed := i.Apply(buf[0:limit])
	if changed {
		err := os.Remove(f.Path)
		if err != nil {
			return fmt.Errorf("error removing old file %s: %s", f.Path, err)
		}
		err = ioutil.WriteFile(f.Path, buf, f.Info.Mode())
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
