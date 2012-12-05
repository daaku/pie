package pie

import (
	"encoding/csv"
	"errors"
	"io"
	"runtime"
)

var errRequirePairs = errors.New("argments should be pairs of regexp and replacement")

// Instructions describe the modification. Instructions are compiled once for
// parallel goroutine of execution allowing some per goroutine work.
type Instruction interface {
	Compile() (CompiledInstruction, error)
}

// A compiled instruction is used repeatedly across files.
type CompiledInstruction interface {
	// This is called first to avoid copying data if there is not match.
	Match(src []byte) bool

	// This applies the instruction and returns a copy of the transformed data.
	Apply(src []byte) []byte
}

type CompiledInstructions []CompiledInstruction

// Parses input as tab delemited pairs of regex and replace pattern.
func InstructionFromReader(r io.Reader) (result []Instruction, err error) {
	reader := csv.NewReader(r)
	reader.Comma = '\t'
	reader.Comment = '#'
	reader.FieldsPerRecord = 2
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	instructions, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	for _, instruction := range instructions {
		result = append(result, &ReplaceAll{
			Target: instruction[0],
			Repl:   []byte(instruction[1]),
		})
	}
	return result, nil
}

// Parses args as pairs of regex and replace pattern.
func InstructionFromArgs(args []string) (result []Instruction, err error) {
	argl := len(args)
	if argl%2 != 0 {
		return nil, errRequirePairs
	}
	for x := 0; x < argl; x = x + 2 {
		result = append(result, &ReplaceAll{
			Target: args[x],
			Repl:   []byte(args[x+1]),
		})
	}
	return result, nil
}

// Apply the instructions and return either the new data and a bool indicating
// if any changes were made.
func (c CompiledInstructions) Apply(input []byte) (out []byte, changed bool) {
	out = input
	for index, instr := range c {
		if index%100 == 0 {
			runtime.Gosched()
		}
		if !instr.Match(out) {
			continue
		}
		out = instr.Apply(out)
		changed = true
	}
	return
}
