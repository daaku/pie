package pie

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
)

var (
	errRequirePairs           = errors.New("argments should be pairs of regexp and replacement")
	errNoInstructionsProvided = errors.New("no instructions provided")
)

type replaceAll struct {
	Target string // regular expression string
	Regexp *regexp.Regexp
	Repl   []byte // replacement value
}

// Defines rules that maps to many regexp.ReplaceAll.
type Instruction struct {
	replaceAll []*replaceAll
}

func (i *Instruction) compile() (err error) {
	for _, r := range i.replaceAll {
		if r.Regexp, err = regexp.Compile(r.Target); err != nil {
			return err
		}
	}
	return
}

func (i *Instruction) ContentMatch(content []byte) bool {
	return true
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
