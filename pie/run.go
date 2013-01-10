package pie

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var (
	errZeroWorkers       = errors.New("must configure number of workers")
	errNilContentMatcher = errors.New("must provide a content matcher")
	errNilTransformer    = errors.New("must provide a transformer")
)

type Run struct {
	PathInclude    PathMatcher
	PathExclude    PathMatcher
	ContentMatcher ContentMatcher
	Transformer    Transformer
	NumWorkers     int
	Verbose        bool
	Root           []string
}

func (r *Run) fileWorker(files chan File) error {
	buf := make([]byte, 1<<20)
	for f := range files {
		if r.Verbose {
			log.Printf("got file %s\n", f.Path)
		}
		size := f.Info.Size()
		if int64(cap(buf)) < size {
			buf = make([]byte, size)
		}

		f.Content = buf

		if err := f.Process(); err != nil {
			return fmt.Errorf("failed to process file %s: %s", f.Path, err)
		}
	}
	return nil
}

func (r *Run) Go() (err error) {
	if r.NumWorkers == 0 {
		return errZeroWorkers
	}
	if r.ContentMatcher == nil {
		return errNilContentMatcher
	}
	if r.Transformer == nil {
		return errNilTransformer
	}

	var wg sync.WaitGroup
	files := make(chan File, r.NumWorkers)
	for i := 0; i < r.NumWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = r.fileWorker(files)
		}()
	}

	for _, root := range r.Root {
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if r.PathInclude != nil && !r.PathInclude.PathMatch(path) {
				if r.Verbose {
					log.Printf("skipping %s: not included\n", path)
				}
				return nil
			}
			if r.PathExclude != nil && r.PathExclude.PathMatch(path) {
				if r.Verbose {
					log.Printf("skipping %s: excluded\n", path)
				}
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if _, p := filepath.Split(path); p[0] == '.' || p[0] == '#' || p[0] == '~' || p[len(p)-1] == '~' {
				if r.Verbose {
					log.Printf("skipping %s: special name\n", path)
				}
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			files <- File{
				Path:           path,
				Info:           info,
				ContentMatcher: r.ContentMatcher,
				Transformer:    r.Transformer,
				Verbose:        r.Verbose,
			}
			return nil
		})
	}

	close(files)
	wg.Wait()
	return
}
