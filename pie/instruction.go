package pie

import (
	"encoding/csv"
	"io"
)

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
	for x := 1; x < argl; x = x + 2 {
		result = append(result, &ReplaceAll{
			Target: args[x],
			Repl:   []byte(args[x+1]),
		})
	}
	return result, nil
}
