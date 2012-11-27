package main

import (
	"fmt"
	"github.com/daaku/pie/pie"
	"os"
	"regexp"
)

func main() {
	argl := len(os.Args)
	if argl < 4 || argl%2 != 0 {
		fmt.Printf("usage: %s <directory> [<target-regexp> <replace-pattern>]...\n", os.Args[0])
		os.Exit(1)
	}
	r := &pie.Run{
		Root: os.Args[1],
	}
	for x := 2; x < argl; x = x + 2 {
		r.Rule = append(r.Rule, &pie.ReplaceAll{
			Target: regexp.MustCompile(os.Args[x]),
			Repl:   []byte(os.Args[x+1]),
		})
	}
	err := r.Run()
	if err != nil {
		fmt.Printf("error: %s", err)
		os.Exit(1)
	}
}
