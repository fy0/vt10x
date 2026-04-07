package vt10x

import (
	"bytes"
	"testing"
)

func TestCSIParse(t *testing.T) {
	var csi csiEscape
	csi.reset()
	csi.buf = []byte("s")
	csi.parse()
	if csi.mode != 's' || csi.arg(0, 17) != 17 || len(csi.args) != 0 {
		t.Fatal("CSI parse mismatch")
	}

	csi.reset()
	csi.buf = []byte("31T")
	csi.parse()
	if csi.mode != 'T' || csi.arg(0, 0) != 31 || len(csi.args) != 1 {
		t.Fatal("CSI parse mismatch")
	}

	csi.reset()
	csi.buf = []byte("48;2f")
	csi.parse()
	if csi.mode != 'f' || csi.arg(0, 0) != 48 || csi.arg(1, 0) != 2 || len(csi.args) != 2 {
		t.Fatal("CSI parse mismatch")
	}

	csi.reset()
	csi.buf = []byte("?25l")
	csi.parse()
	if csi.mode != 'l' || csi.arg(0, 0) != 25 || csi.priv != true || len(csi.args) != 1 {
		t.Fatal("CSI parse mismatch")
	}

	csi.reset()
	csi.buf = []byte(">7u")
	csi.parse()
	if csi.mode != 'u' || csi.prefix != '>' || csi.priv || csi.arg(0, 0) != 7 || len(csi.args) != 1 {
		t.Fatal("CSI parse mismatch")
	}
}

func TestXtermStylePrimaryDAResponse(t *testing.T) {
	var reply bytes.Buffer
	term := New(WithWriter(&reply))
	if _, err := term.Write([]byte("\033[c")); err != nil {
		t.Fatal(err)
	}
	if got := reply.String(); got != "\033[?1;2c" {
		t.Fatalf("unexpected DA response: %q", got)
	}
}

func TestPrivateCSIUIgnored(t *testing.T) {
	term := New(WithSize(10, 1))
	if _, err := term.Write([]byte("ab\033[scd\033[?uX")); err != nil {
		t.Fatal(err)
	}
	if got := extractStr(term, 0, 4, 0); got != "abcdX" {
		t.Fatalf("expected private CSI u to be ignored, got %q", got)
	}
}
