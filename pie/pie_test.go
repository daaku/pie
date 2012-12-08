package pie_test

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"code.google.com/p/codesearch/index"
	"github.com/daaku/pie/pie"
)

var removeTemp = flag.Bool("remove-temp", true, "remove temp copies of test data")

type TestCase struct {
	Name        string
	Instruction []pie.Instruction
	FileIgnore  string
	FileFilter  string
	NumWorkers  int
}

var cases = []TestCase{
	TestCase{
		Name: "base",
		Instruction: []pie.Instruction{
			&pie.ReplaceAll{
				Target: "hello",
				Repl:   []byte("goodbye"),
			},
		},
	},
	TestCase{ // test case without instruction
		Name: "empty-file",
	},
	TestCase{
		Name: "empty-file",
		Instruction: []pie.Instruction{
			&pie.ReplaceAll{
				Target: "hello",
				Repl:   []byte("goodbye"),
			},
		},
	},
	TestCase{
		Name: "ignore-git",
		Instruction: []pie.Instruction{
			&pie.ReplaceAll{
				Target: "hello",
				Repl:   []byte("goodbye"),
			},
		},
	},
	TestCase{
		Name: "ignore-symlink",
		Instruction: []pie.Instruction{
			&pie.ReplaceAll{
				Target: "hello",
				Repl:   []byte("goodbye"),
			},
		},
	},
	TestCase{
		Name:       "file-ignore",
		FileIgnore: "foo",
		Instruction: []pie.Instruction{
			&pie.ReplaceAll{
				Target: "hello",
				Repl:   []byte("goodbye"),
			},
		},
	},
	TestCase{
		Name:       "file-filter",
		FileFilter: "(a|b)$",
		Instruction: []pie.Instruction{
			&pie.ReplaceAll{
				Target: "hello",
				Repl:   []byte("goodbye"),
			},
		},
	},
	TestCase{
		Name:       "dedupe",
		NumWorkers: 1,
		Instruction: []pie.Instruction{
			&pie.ReplaceAll{
				Target: "hello",
				Repl:   []byte("hello1"),
			},
			&pie.ReplaceAll{
				Target: "hello",
				Repl:   []byte("hello2"),
			},
		},
	},
	TestCase{
		Name:       "collapse-initial-trigrams",
		NumWorkers: 1,
		Instruction: []pie.Instruction{
			&pie.ReplaceAll{
				Target: "hello",
				Repl:   []byte("HELLO"),
			},
			&pie.ReplaceAll{
				Target: "world",
				Repl:   []byte("WORLD"),
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

		ixFile := filepath.Join(tmp, ".csearchindex")
		ixw := index.Create(ixFile)
		ixw.AddPaths([]string{tmp})
		filepath.Walk(tmp, func(path string, info os.FileInfo, err error) error {
			if _, elem := filepath.Split(path); elem != "" {
				// Skip various temporary or "hidden" files or directories.
				if elem[0] == '.' || elem[0] == '#' || elem[0] == '~' || elem[len(elem)-1] == '~' {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if info != nil && info.Mode()&os.ModeType == 0 {
				ixw.AddFile(path)
			}
			return nil
		})
		ixw.Flush()

		run := &pie.Run{
			Index:       index.Open(ixFile),
			Instruction: test.Instruction,
			FileIgnore:  test.FileIgnore,
			FileFilter:  test.FileFilter,
			NumWorkers:  test.NumWorkers,
		}
		err = run.Run()
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

func TestInstructionFromReader(t *testing.T) {
	input := bytes.NewBufferString("a\tb\nc\td")
	i, err := pie.InstructionFromReader(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(i) != 2 {
		t.Fatalf("was expecting 2 instructions but got %d", len(i))
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
	i, err := pie.InstructionFromArgs([]string{"a", "b", "c", "d"})
	if err != nil {
		t.Fatal(err)
	}
	if len(i) != 2 {
		t.Fatalf("was expecting 2 instructions but got %d", len(i))
	}
}

func TestInstructionFromArgsPairError(t *testing.T) {
	_, err := pie.InstructionFromArgs([]string{"a"})
	if err == nil {
		t.Fatal("was expecting an error")
	}
}
