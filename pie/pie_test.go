package pie_test

import (
	"github.com/daaku/pie/pie"
	"os"
	"regexp"
	"testing"
)

type TestCase struct {
	Name string
	Rule []pie.Rule
}

var cases = []TestCase{
	TestCase{
		Name: "base",
		Rule: []pie.Rule{
			&pie.ReplaceAll{
				Target: regexp.MustCompile("hello"),
				Repl:   []byte("goodbye"),
			},
		},
	},
}

func (t TestCase) MakeTempCopy() string {
	return "/home/naitik/usr/go/src/pkg/github.com/daaku/pie/pie/_tests/base/before"
}

func GetDataDir() string {
	return "/home/naitik/usr/go/src/pkg/github.com/daaku/pie/pie/_tests"
}

func TestAll(t *testing.T) {
	for _, test := range cases {
		tmp := test.MakeTempCopy()
		defer os.RemoveAll(tmp)
		run := &pie.Run{
			Root: tmp,
			Rule: test.Rule,
		}
		err := run.Run()
		if err != nil {
			t.Fatal("run for %s failed: %s", test.Name, err)
		}
	}
}
