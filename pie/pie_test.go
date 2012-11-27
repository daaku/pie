package pie_test

import (
	"bytes"
	"fmt"
	"github.com/daaku/pie/pie"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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

func (t TestCase) dir(last string) string {
	return filepath.Join(GetDataDir(), t.Name, last)
}

func (t TestCase) MakeTempCopy() (string, error) {
	dir, err := ioutil.TempDir("", "pie-test")
	if err != nil {
		return "", err
	}
	out, err := exec.Command("cp", "-r", t.dir("before"), dir).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(
			"error copying to temp directory %s (%s): %s\n%s", t.Name, dir, err, out)
	}
	return dir, nil
}

func (t TestCase) Compare(dir string) (bool, error) {
	afterDir := t.dir("after")
	same := true
	err := filepath.Walk(
		afterDir,
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			expected, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("error reading expected file %s: %s", path, err)
			}
			actualPath := filepath.Join(dir, "before", strings.Replace(path, afterDir, "", 1))
			actual, err := ioutil.ReadFile(actualPath)
			if err != nil {
				return fmt.Errorf("error reading actual file %s: %s", actualPath, err)
			}
			if bytes.Compare(expected, actual) != 0 {
				same = false
			}
			return nil
		})
	return same, err
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
		run := &pie.Run{
			Root: tmp,
			Rule: test.Rule,
		}
		err = run.Run()
		if err != nil {
			t.Fatal("run for %s failed: %s", test.Name, err)
		}
		same, err := test.Compare(tmp)
		if err != nil {
			t.Fatalf("compare for %s failed: %s", test.Name, err)
		}
		if !same {
			t.Fatal("did not get expected result for %s", test.Name)
		}
		os.RemoveAll(tmp)
	}
}
