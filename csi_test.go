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

func TestXtermStyleSecondaryDAResponse(t *testing.T) {
	var reply bytes.Buffer
	term := New(WithWriter(&reply))
	if _, err := term.Write([]byte("\033[>c")); err != nil {
		t.Fatal(err)
	}
	if got := reply.String(); got != "\033[>0;95;0c" {
		t.Fatalf("unexpected secondary DA response: %q", got)
	}
}

func TestConfigurableSecondaryDAResponse(t *testing.T) {
	var reply bytes.Buffer
	term := New(
		WithWriter(&reply),
		WithXtermStyle(),
	)
	if _, err := term.Write([]byte("\033[>c")); err != nil {
		t.Fatal(err)
	}
	if got := reply.String(); got != "\033[>0;276;0c" {
		t.Fatalf("unexpected configurable secondary DA response: %q", got)
	}
}

func TestPrivateDSRCPRResponse(t *testing.T) {
	var reply bytes.Buffer
	term := New(WithWriter(&reply), WithSize(10, 5))
	writeAll(t, term, "\033[3;4H\033[?6n")
	if got := reply.String(); got != "\033[?3;4R" {
		t.Fatalf("unexpected private CPR response: %q", got)
	}
	if cur := term.Cursor(); cur.X != 3 || cur.Y != 2 {
		t.Fatalf("unexpected cursor after private CPR: (%d,%d)", cur.X, cur.Y)
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

func TestUnsupportedPrefixedCSIIsIgnored(t *testing.T) {
	term := New(WithSize(5, 3))
	writeAll(t, term, "abcde\033[s")

	st := term.(*terminal)
	beforeCur := st.cur
	beforeSaved := st.curSaved
	beforeTop := st.top
	beforeBottom := st.bottom
	beforeMode := st.mode

	writeAll(t, term, "\033[>2J\033[=2r\033[<s\033[?u")

	if got := extractStr(term, 0, 4, 0); got != "abcde" {
		t.Fatalf("expected prefixed CSI to leave screen unchanged, got %q", got)
	}
	if st.cur != beforeCur {
		t.Fatalf("cursor changed after ignored prefixed CSI: before=%+v after=%+v", beforeCur, st.cur)
	}
	if st.curSaved != beforeSaved {
		t.Fatalf("saved cursor changed after ignored prefixed CSI: before=%+v after=%+v", beforeSaved, st.curSaved)
	}
	if st.top != beforeTop || st.bottom != beforeBottom {
		t.Fatalf("scroll region changed after ignored prefixed CSI: before=(%d,%d) after=(%d,%d)", beforeTop, beforeBottom, st.top, st.bottom)
	}
	if st.mode != beforeMode {
		t.Fatalf("mode changed after ignored prefixed CSI: before=%v after=%v", beforeMode, st.mode)
	}
}
