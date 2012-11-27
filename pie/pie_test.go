package pie_test

import (
	"fmt"
	"github.com/daaku/pie/pie"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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

func (t TestCase) MakeTempCopy() (string, error) {
	dir, err := ioutil.TempDir("", "pie-test")
	if err != nil {
		return "", err
	}
	out, err := exec.Command(
		"cp", "-r", filepath.Join(GetDataDir(), t.Name, "before"), dir).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(
			"error copying to temp directory %s (%s): %s\n%s", t.Name, dir, err, out)
	}
	return dir, nil
}

func GetDataDir() string {
	return "/home/naitik/usr/go/src/pkg/github.com/daaku/pie/pie/_tests"
}

func TestAll(t *testing.T) {
	for _, test := range cases {
		tmp, err := test.MakeTempCopy()
		if err != nil {
			t.Fatal("faled to make temp copy for %s: %s", test.Name, err)
		}
		defer os.RemoveAll(tmp)
		run := &pie.Run{
			Root: tmp,
			Rule: test.Rule,
		}
		err = run.Run()
		if err != nil {
			t.Fatal("run for %s failed: %s", test.Name, err)
		}
	}
}
