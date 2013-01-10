package pie_test

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/daaku/pie/pie"
)

var (
	removeTemp = flag.Bool("remove-temp", true, "remove temp copies of test data")
	verbose    = flag.Bool("verbose", false, "be more verbose")
)

type TestCase struct {
	Name        string
	Instruction *pie.Instruction
	PathExclude string
	PathInclude string
	NumWorkers  int
}

func parseInstructions(args ...string) *pie.Instruction {
	i, err := pie.InstructionFromArgs(args)
	if err != nil {
		log.Fatal(err)
	}
	return i
}

var cases = []TestCase{
	TestCase{
		Name:        "base",
		Instruction: parseInstructions("hello", "goodbye"),
	},
	TestCase{
		Name:        "empty-file",
		Instruction: parseInstructions("hello", "goodbye"),
	},
	TestCase{
		Name:        "ignore-git",
		Instruction: parseInstructions("hello", "goodbye"),
	},
	TestCase{
		Name:        "ignore-symlink",
		Instruction: parseInstructions("hello", "goodbye"),
	},
	TestCase{
		Name:        "file-ignore",
		PathExclude: "foo",
		Instruction: parseInstructions("hello", "goodbye"),
	},
	TestCase{
		Name:        "file-filter",
		PathInclude: "(a|b)$",
		Instruction: parseInstructions("hello", "goodbye"),
	},
	TestCase{
		Name:       "dedupe",
		NumWorkers: 1,
		Instruction: parseInstructions(
			"hello", "hello1",
			"hello", "hello2"),
	},
	TestCase{
		Name:       "collapse-initial-trigrams",
		NumWorkers: 1,
		Instruction: parseInstructions(
			"hello", "HELLO",
			"world", "WORLD"),
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
			if info.Mode()&os.ModeSymlink != 0 {
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
	pkg, err := build.Import("github.com/daaku/pie/pie/_tests", "", build.FindOnly)
	if err != nil {
		panic(fmt.Sprintf("could not find test data directory %s", err))
	}
	return pkg.Dir
}

func TestRun(t *testing.T) {
	t.Parallel()
	for _, test := range cases {
		tmp, err := test.MakeTempCopy()
		if err != nil {
			t.Fatalf("faled to make temp copy for %s: %s", test.Name, err)
		}

		if *verbose {
			log.Printf("temp copy: %s\n", tmp)
		}
		run := &pie.Run{
			ContentMatcher: test.Instruction,
			Transformer:    test.Instruction,
			NumWorkers:     runtime.NumCPU(),
			Root:           []string{tmp},
			Verbose:        *verbose,
		}

		if test.NumWorkers > 0 {
			run.NumWorkers = test.NumWorkers
		}

		if test.PathExclude != "" {
			if run.PathExclude, err = pie.RegExpPathMatcher(test.PathExclude); err != nil {
				t.Fatalf("invalid path exclude: %s", err)
			}
		}
		if test.PathInclude != "" {
			if run.PathInclude, err = pie.RegExpPathMatcher(test.PathInclude); err != nil {
				t.Fatalf("invalid path include: %s", err)
			}
		}

		err = run.Go()
		if err != nil {
			t.Fatalf("run for %s failed: %s", test.Name, err)
		}
		same, err := test.Compare(tmp)
		if err != nil {
			t.Fatalf("compare for %s failed: %s", test.Name, err)
		}
		if !same {
			t.Fatalf("did not get expected result for %s", test.Name)
		}
		if *removeTemp {
			os.RemoveAll(tmp)
		}
	}
}
