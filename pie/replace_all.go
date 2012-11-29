package pie

import (
	"regexp"
)

type replaceAllCompiled struct {
	Target *regexp.Regexp
	Repl   []byte
}

func (r *replaceAllCompiled) Match(src []byte) bool {
	return r.Target.Match(src)
}

func (r *replaceAllCompiled) Apply(src []byte) []byte {
	return r.Target.ReplaceAll(src, r.Repl)
}

// Defines a rule that maps to regexp.ReplaceAll.
type ReplaceAll struct {
	Target string // regular expression string
	Repl   []byte // replacement value
}

func (r *ReplaceAll) Compile() (CompiledInstruction, error) {
	re, err := regexp.Compile(r.Target)
	if err != nil {
		return nil, err
	}
	return &replaceAllCompiled{
		Target: re,
		Repl:   r.Repl,
	}, nil
}
