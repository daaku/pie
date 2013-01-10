package pie_test

import (
	"testing"

	"github.com/daaku/pie/pie"
)

func TestSimpleRegExpPathMatcher(t *testing.T) {
	p, err := pie.RegExpPathMatcher("foo")
	if err != nil {
		t.Fatal(err)
	}
	if !p.PathMatch("/baz/foo") {
		t.Fatal("expected match")
	}
}

func TestRegExpPathMatcher(t *testing.T) {
	p, err := pie.RegExpPathMatcher("(a|b)$")
	if err != nil {
		t.Fatal(err)
	}
	if !p.PathMatch("/baz/a") {
		t.Fatal("expected match")
	}
	if p.PathMatch("/baz/c") {
		t.Fatal("expected no match")
	}
}
