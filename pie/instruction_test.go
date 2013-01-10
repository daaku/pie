package pie_test

import (
	"bytes"
	"testing"

	"github.com/daaku/pie/pie"
)

func TestInstructionFromReader(t *testing.T) {
	input := bytes.NewBufferString("a\tb\nc\td")
	_, err := pie.InstructionFromReader(input)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInstructionFromReaderError(t *testing.T) {
	input := bytes.NewBufferString("a\tb\nc")
	_, err := pie.InstructionFromReader(input)
	if err == nil {
		t.Fatal("was expecting an error")
	}
}

func TestInstructionFromArgs(t *testing.T) {
	_, err := pie.InstructionFromArgs([]string{"a", "b", "c", "d"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestInstructionFromArgsPairError(t *testing.T) {
	_, err := pie.InstructionFromArgs([]string{"a"})
	if err == nil {
		t.Fatal("was expecting an error")
	}
}
