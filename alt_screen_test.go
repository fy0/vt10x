package vt10x

import (
	"io"
	"strings"
	"testing"
)

func writeAll(t *testing.T, term Terminal, seq string) {
	t.Helper()
	if _, err := term.Write([]byte(seq)); err != nil && err != io.EOF {
		t.Fatal(err)
	}
}

func rowString(term Terminal, row int) string {
	cols, _ := term.Size()
	return extractStr(term, 0, cols-1, row)
}

func assertScreenRows(t *testing.T, term Terminal, expected []string) {
	t.Helper()
	for row, want := range expected {
		if got := rowString(term, row); got != want {
			t.Fatalf("row %d mismatch: got %q want %q", row+1, got, want)
		}
	}
}

func TestAltScreenResetsScrollRegion(t *testing.T) {
	term := New(WithSize(10, 5))
	writeAll(t, term, "\x1b[2;4r\x1b[?1049h\x1b[0m\x1b[2J\x1b[3J\x1b[H"+
		"1111111111\r\n2222222222\r\n3333333333\r\n4444444444\r\n5555555555")

	assertScreenRows(t, term, []string{
		"1111111111",
		"2222222222",
		"3333333333",
		"4444444444",
		"5555555555",
	})
}

func TestAltScreenKeepsOriginButUsesFullViewportMargins(t *testing.T) {
	term := New(WithSize(10, 5))
	writeAll(t, term, "\x1b[2;4r\x1b[?6h\x1b[?1049h\x1b[0m\x1b[2J\x1b[3J\x1b[H"+
		"1111111111\r\n2222222222\r\n3333333333\r\n4444444444\r\n5555555555")

	assertScreenRows(t, term, []string{
		"1111111111",
		"2222222222",
		"3333333333",
		"4444444444",
		"5555555555",
	})
}

func TestAltScreenRoundTripRestoresNormalScreen(t *testing.T) {
	term := New(WithSize(10, 5))
	writeAll(t, term, "\x1b[3;4HX\x1b[?1049hALT\x1b[?1049l")

	if got := rowString(term, 2); got != "   X      " {
		t.Fatalf("normal screen content mismatch: got %q", got)
	}

	cur := term.Cursor()
	if cur.X != 4 || cur.Y != 2 {
		t.Fatalf("cursor mismatch after 1049 round trip: got (%d,%d)", cur.X, cur.Y)
	}

	writeAll(t, term, "\x1b[?1049h")
	assertScreenRows(t, term, []string{
		"          ",
		"          ",
		"          ",
		"          ",
		"          ",
	})
}

func TestResetClearsEntireViewport(t *testing.T) {
	term := New(WithSize(10, 5))
	rows := []string{
		"AAAAAAAAAA",
		"BBBBBBBBBB",
		"CCCCCCCCCC",
		"DDDDDDDDDD",
		"EEEEEEEEEE",
	}
	writeAll(t, term, strings.Join(rows, "\r\n"))
	writeAll(t, term, "\x1bc")

	assertScreenRows(t, term, []string{
		"          ",
		"          ",
		"          ",
		"          ",
		"          ",
	})
}
