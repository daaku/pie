package pie

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"

	"code.google.com/p/codesearch/index"
	cre "code.google.com/p/codesearch/regexp"
)

var (
	errRequirePairs           = errors.New("argments should be pairs of regexp and replacement")
	errNoInstructionsProvided = errors.New("no instructions provided")
	ErrNonUTF8                = errors.New("not utf8")
)

type replaceAll struct {
	Target string // regular expression string
	Regexp *regexp.Regexp
	Repl   []byte // replacement value
}

// Defines rules that maps to many regexp.ReplaceAll.
type Instruction struct {
	replaceAll        []*replaceAll
	contentMatchQuery *index.Query
}

func (i *Instruction) compile() (err error) {
	combined := bytes.NewBufferString("(?m)(")
	last := len(i.replaceAll) - 1
	for n, r := range i.replaceAll {
		if r.Regexp, err = regexp.Compile(r.Target); err != nil {
			return err
		}
		combined.WriteString(r.Target)
		if n != last {
			combined.WriteString("|")
		}
	}
	combined.WriteString(")")

	re, err := cre.Compile(combined.String())
	if err != nil {
		return fmt.Errorf(
			"failed to parse combined regexp %s: %s", combined.String(), err)
	}
	i.contentMatchQuery = index.RegexpQuery(re.Syntax)

	return
}

// Apply the instructions and return result and a bool indicating if any
// changes were made.
func (i *Instruction) Transform(input []byte) (out []byte, changed bool, err error) {
	if i == nil {
		return nil, false, errNoInstructionsProvided
	}

	out = input
	for index, r := range i.replaceAll {
		if index%100 == 0 {
			runtime.Gosched()
		}
		if r.Regexp == nil {
			return nil, false, fmt.Errorf("regexp is nil")
		}
		if !r.Regexp.Match(out) {
			continue
		}
		out = r.Regexp.ReplaceAll(out, r.Repl)
		changed = true
	}
	return
}

func (i *Instruction) ContentMatch(content []byte) bool {
	return i.contentMatch(i.contentMatchQuery, content, nil)
}

func (i *Instruction) contentMatch(q *index.Query, content []byte, contentTris map[uint32]bool) bool {
	if q.Op == index.QNone {
		return false
	}
	if q.Op == index.QAll {
		return true
	}

	if contentTris == nil {
		var err error
		contentTris, err = trigramSet(content)
		if err != nil {
			if err == ErrNonUTF8 {
				return false
			}
			panic(err)
		}
	}

	switch q.Op {
	case index.QAnd:
		for _, t := range q.Trigram {
			tri := uint32(t[0])<<16 | uint32(t[1])<<8 | uint32(t[2])
			if !contentTris[tri] {
				return false
			}
		}
		for _, sub := range q.Sub {
			if !i.contentMatch(sub, content, contentTris) {
				return false
			}
		}
		return true
	case index.QOr:
		if len(q.Trigram) > 0 {
			found := false
			for _, t := range q.Trigram {
				tri := uint32(t[0])<<16 | uint32(t[1])<<8 | uint32(t[2])
				if contentTris[tri] {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		for _, sub := range q.Sub {
			if !i.contentMatch(sub, content, contentTris) {
				return false
			}
		}
		return true
	}

	panic(fmt.Sprintf("unknown op: %s", q.Op))
}

func trigramSet(buf []byte) (map[uint32]bool, error) {
	var (
		c   = byte(0)
		tv  = uint32(0)
		tri = make(map[uint32]bool)
	)
	for i := 0; i < len(buf); i++ {
		tv = (tv << 8) & (1<<24 - 1)
		c = buf[i]
		tv |= uint32(c)
		if !validUTF8((tv>>8)&0xFF, tv&0xFF) {
			return nil, ErrNonUTF8
		}
		tri[tv] = true
	}
	return tri, nil
}

// validUTF8 reports whether the byte pair can appear in a
// valid sequence of UTF-8-encoded code points.
func validUTF8(c1, c2 uint32) bool {
	switch {
	case c1 < 0x80:
		// 1-byte, must be followed by 1-byte or first of multi-byte
		return c2 < 0x80 || 0xc0 <= c2 && c2 < 0xf8
	case c1 < 0xc0:
		// continuation byte, can be followed by nearly anything
		return c2 < 0xf8
	case c1 < 0xf8:
		// first of multi-byte, must be followed by continuation byte
		return 0x80 <= c2 && c2 < 0xc0
	}
	return false
}

// Parses input as tab delemited pairs of regex and replace pattern.
func InstructionFromReader(r io.Reader) (*Instruction, error) {
	reader := csv.NewReader(r)
	reader.Comma = '\t'
	reader.Comment = '#'
	reader.FieldsPerRecord = 2
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	ra := make([]*replaceAll, 0, len(lines))
	for _, line := range lines {
		ra = append(ra, &replaceAll{
			Target: line[0],
			Repl:   []byte(line[1]),
		})
	}

	if len(ra) == 0 {
		return nil, errNoInstructionsProvided
	}

	i := &Instruction{replaceAll: ra}
	if err := i.compile(); err != nil {
		return nil, err
	}

	return i, nil
}

// Parses args as pairs of regex and replace pattern.
func InstructionFromArgs(args []string) (*Instruction, error) {
	argl := len(args)
	if argl%2 != 0 {
		return nil, errRequirePairs
	}
	ra := make([]*replaceAll, 0, argl/2)
	for x := 0; x < argl; x = x + 2 {
		ra = append(ra, &replaceAll{
			Target: args[x],
			Repl:   []byte(args[x+1]),
		})
	}

	if len(ra) == 0 {
		return nil, errNoInstructionsProvided
	}

	i := &Instruction{replaceAll: ra}
	if err := i.compile(); err != nil {
		return nil, err
	}

	return i, nil
}

// Parse instructions from the specified file.
func InstructionFromFile(file string) (*Instruction, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	instruction, err := InstructionFromReader(f)
	if err != nil {
		return nil, err
	}
	return instruction, nil
}
