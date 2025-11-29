package vt10x

import (
	"io"
	"strings"
	"testing"
)

func extractStr(term Terminal, x0, x1, row int) string {
	var s []rune
	for i := x0; i <= x1; i++ {
		attr := term.Cell(i, row)
		s = append(s, attr.Char)
	}
	return string(s)
}

func TestPlainChars(t *testing.T) {
	term := New()
	expected := "Hello world!"
	_, err := term.Write([]byte(expected))
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}
	actual := extractStr(term, 0, len(expected)-1, 0)
	if expected != actual {
		t.Fatal(actual)
	}
}

func TestNewline(t *testing.T) {
	term := New()
	expected := "Hello world!\n...and more."
	_, err := term.Write([]byte("\033[20h")) // set CRLF mode
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}
	_, err = term.Write([]byte(expected))
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}

	split := strings.Split(expected, "\n")
	actual := extractStr(term, 0, len(split[0])-1, 0)
	actual += "\n"
	actual += extractStr(term, 0, len(split[1])-1, 1)
	if expected != actual {
		t.Fatal(actual)
	}

	// A newline with a color set should not make the next line that color,
	// which used to happen if it caused a scroll event.
	st := (term.(*terminal))
	st.moveTo(0, st.rows-1)
	_, err = term.Write([]byte("\033[1;37m\n$ \033[m"))
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}
	cur := term.Cursor()
	attr := term.Cell(cur.X, cur.Y)
	if attr.FG != DefaultFG {
		t.Fatal(st.cur.X, st.cur.Y, attr.FG, attr.BG)
	}
}

func TestIndexColor(t *testing.T) {
	term := New()
	_, err := term.Write([]byte("\033[30mA"))
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}
	attr := term.Cell(0, 0)
	// The predefined color index 0 is:
	if attr.FG != 0x2e3436 {
		t.Fatal(attr.FG)
	}
}

func TestSGRFaint(t *testing.T) {
	term := New()
	if _, err := term.Write([]byte("\033[32;2mF")); err != nil && err != io.EOF {
		t.Fatal(err)
	}
	attr := term.Cell(0, 0)
	if attr.Mode&attrFaint == 0 {
		t.Fatal("expected faint attribute on cell")
	}
	base := byte2color(2)
	r := (base >> 16) & 0xff
	g := (base >> 8) & 0xff
	b := base & 0xff
	expected := Color((r>>1)<<16 | (g>>1)<<8 | (b >> 1))
	if attr.FG != expected {
		t.Fatalf("expected faint color %06x, got %06x", expected, attr.FG)
	}
}
