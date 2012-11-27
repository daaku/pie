package main

import (
	"fmt"
	"github.com/daaku/pie/pie"
	"os"
	"regexp"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Printf("usage: %s <directory> <target-regexp> <replace-pattern>\n", os.Args[0])
		os.Exit(1)
	}
	r := &pie.Run{
		Root: os.Args[1],
		Rule: []pie.Rule{
			&pie.ReplaceAll{
				Target: regexp.MustCompile(os.Args[2]),
				Repl:   []byte(os.Args[3]),
			},
		},
	}
	err := r.Run()
	if err != nil {
		fmt.Printf("error: %s", err)
		os.Exit(1)
	}
}
