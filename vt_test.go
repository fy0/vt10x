package vt10x

import (
	"bytes"
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
	if attr.Mode&AttrFaint == 0 {
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

func TestCodexStyleStartupTraceQueries(t *testing.T) {
	var reply bytes.Buffer
	term := New(WithWriter(&reply), WithSize(10, 5))
	st := term.(*terminal)

	writeAll(t, term, "HELLO\033[s\033[3;4H")

	beforeCur := st.cur
	beforeSaved := st.curSaved
	beforeTop := st.top
	beforeBottom := st.bottom
	beforeMode := st.mode
	beforeRows := []string{
		rowString(term, 0),
		rowString(term, 1),
		rowString(term, 2),
		rowString(term, 3),
		rowString(term, 4),
	}

	writeAll(t, term, "\033[6n\033[c\033]10;?\033\\\033]11;?\033\\\033[?u\033[>7u\033[?6n\033[>c")

	expected := "\033[3;4R" +
		"\033[?1;2c" +
		oscColorReply(10, byte2color(int(LightGrey)), "\033\\") +
		oscColorReply(11, byte2color(int(Black)), "\033\\") +
		"\033[?3;4R" +
		"\033[>0;95;0c"
	if got := reply.String(); got != expected {
		t.Fatalf("unexpected startup trace replies: %q", got)
	}

	if st.cur != beforeCur {
		t.Fatalf("cursor changed after startup trace: before=%+v after=%+v", beforeCur, st.cur)
	}
	if st.curSaved != beforeSaved {
		t.Fatalf("saved cursor changed after startup trace: before=%+v after=%+v", beforeSaved, st.curSaved)
	}
	if st.top != beforeTop || st.bottom != beforeBottom {
		t.Fatalf("scroll region changed after startup trace: before=(%d,%d) after=(%d,%d)", beforeTop, beforeBottom, st.top, st.bottom)
	}
	if st.mode != beforeMode {
		t.Fatalf("mode changed after startup trace: before=%v after=%v", beforeMode, st.mode)
	}

	afterRows := []string{
		rowString(term, 0),
		rowString(term, 1),
		rowString(term, 2),
		rowString(term, 3),
		rowString(term, 4),
	}
	for i := range beforeRows {
		if afterRows[i] != beforeRows[i] {
			t.Fatalf("row %d changed after startup trace: before=%q after=%q", i+1, beforeRows[i], afterRows[i])
		}
	}
}

func TestStartupTraceModeQueriesDoNotChangeScreenState(t *testing.T) {
	var reply bytes.Buffer
	term := New(WithWriter(&reply), WithSize(10, 3))
	st := term.(*terminal)

	writeAll(t, term, "abc\033[s\033[2;2H")
	beforeCur := st.cur
	beforeSaved := st.curSaved
	beforeMode := st.mode
	beforeTop := st.top
	beforeBottom := st.bottom
	beforeRows := []string{rowString(term, 0), rowString(term, 1), rowString(term, 2)}

	writeAll(t, term, "\033[?2004$p\033[?1004$p\033[4$p\033[?2026$p")

	if got := reply.String(); got != "\033[?2004;2$y\033[?1004;2$y\033[4;2$y\033[?2026;0$y" {
		t.Fatalf("unexpected mode query replies: %q", got)
	}
	if st.cur != beforeCur {
		t.Fatalf("cursor changed after mode queries: before=%+v after=%+v", beforeCur, st.cur)
	}
	if st.curSaved != beforeSaved {
		t.Fatalf("saved cursor changed after mode queries: before=%+v after=%+v", beforeSaved, st.curSaved)
	}
	if st.mode != beforeMode {
		t.Fatalf("mode changed after mode queries: before=%v after=%v", beforeMode, st.mode)
	}
	if st.top != beforeTop || st.bottom != beforeBottom {
		t.Fatalf("scroll region changed after mode queries: before=(%d,%d) after=(%d,%d)", beforeTop, beforeBottom, st.top, st.bottom)
	}
	for i, want := range beforeRows {
		if got := rowString(term, i); got != want {
			t.Fatalf("row %d changed after mode queries: before=%q after=%q", i+1, want, got)
		}
	}
}
