package pie

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

// Used to include or exclude files by matching their paths.
type PathMatcher interface {
	// Gets the full path to the file. Must return a bool indiciating if the file
	// should be included or not.
	PathMatch(p string) bool
}

// Used to apply the transform to a file.
type Transformer interface {
	// Transform some content, returning the new content, a bool indicating if
	// changes were made, or possibly an error.
	Transform(content []byte) ([]byte, bool, error)
}

// Check if the content is eligible and we should apply the transform.
type ContentMatcher interface {
	// Returns true if content needs to be transformed.
	ContentMatch(content []byte) bool
}

// The per file level processing logic for pie. This handles things like
// skipping file, an efficient trigram based eligibility check and the actual
// processing of the file as necessary.
type File struct {
	Path           string
	Info           os.FileInfo
	Content        []byte
	ContentMatcher ContentMatcher
	Transformer    Transformer
	Verbose        bool
}

// Read the content into the existing buffer.
func (f *File) ReadContent() error {
	fd, err := os.Open(f.Path)
	if err != nil {
		return err
	}
	defer fd.Close()
	if _, err = io.ReadFull(fd, f.Content[0:f.Info.Size()]); err != nil {
		return err
	}
	return nil
}

// Process the file.
func (f *File) Process() error {
	if f.Info.Mode()&os.ModeType != 0 {
		if f.Verbose {
			log.Printf("skipping %s: non standard mode\n", f.Path)
		}
		return nil
	}

	if err := f.ReadContent(); err != nil {
		return err
	}

	if f.Content == nil {
		return fmt.Errorf("nil content for %s", f.Path)
	}

	if !f.ContentMatcher.ContentMatch(f.Content) {
		if f.Verbose {
			log.Printf("skipping %s: no content match\n", f.Path)
		}
		return nil
	}

	buf, changed, err := f.Transformer.Transform(f.Content[0:f.Info.Size()])
	if err != nil {
		return err
	}
	if changed {
		if err = os.Remove(f.Path); err != nil {
			return fmt.Errorf("error removing old file %s: %s", f.Path, err)
		}
		if err = ioutil.WriteFile(f.Path, buf, f.Info.Mode()); err != nil {
			return fmt.Errorf("error writing new file %s: %s", f.Path, err)
		}
	}
	return nil
}
