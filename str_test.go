package vt10x

import (
	"bytes"
	"fmt"
	"testing"
)

func oscColorReply(num int, c Color, term string) string {
	r, g, b := rgb(int(c))
	return fmt.Sprintf("\033]%d;rgb:%02x%02x/%02x%02x/%02x%02x%s", num, r, r, g, g, b, b, term)
}

func TestSTRParse(t *testing.T) {
	var str strEscape
	str.reset()
	str.buf = []rune("0;some text")
	str.parse()
	if str.arg(0, 17) != 0 || str.argString(1, "") != "some text" {
		t.Fatal("STR parse mismatch")
	}
}

func TestParseColor(t *testing.T) {
	type testCase struct {
		name    string
		input   string
		r, g, b int
	}

	for _, tc := range []testCase{
		{
			"rgb 4 bit zero",
			"rgb:0/0/0",
			0, 0, 0,
		},
		{
			"rgb 4 bit max",
			"rgb:f/f/f",
			255, 255, 255,
		},
		{
			"rgb 4 bit values",
			"rgb:1/2/3",
			17, 34, 51,
		},
		{
			"rgb 8 bit zero",
			"rgb:00/00/00",
			0, 0, 0,
		},
		{
			"rgb 8 bit max",
			"rgb:ff/ff/ff",
			255, 255, 255,
		},
		{
			"rgb 8 bit values",
			"rgb:11/22/33",
			17, 34, 51,
		},
		{
			"rgb 12 bit zero",
			"rgb:000/000/000",
			0, 0, 0,
		},
		{
			"rgb 12 bit max",
			"rgb:fff/fff/fff",
			255, 255, 255,
		},
		{
			"rgb 12 bit values",
			"rgb:111/222/333",
			17, 34, 51,
		},
		{
			"rgb 16 bit zero",
			"rgb:0000/0000/0000",
			0, 0, 0,
		},
		{
			"rgb 16 bit max",
			"rgb:ffff/ffff/ffff",
			255, 255, 255,
		},
		{
			"rgb 16 bit values",
			"rgb:1111/2222/3333",
			17, 34, 51,
		},
		{
			"rgb 16 bit values",
			"rgb:1111/2222/3333",
			17, 34, 51,
		},
		{
			"hash 4 bit zero",
			"#000",
			0, 0, 0,
		},
		{
			"hash 4 bit max",
			"#fff",
			240, 240, 240,
		},
		{
			"hash 4 bit values",
			"#123",
			16, 32, 48,
		},
		{
			"hash 8 bit zero",
			"#000000",
			0, 0, 0,
		},
		{
			"hash 8 bit max",
			"#ffffff",
			255, 255, 255,
		},
		{
			"hash 8 bit values",
			"#112233",
			17, 34, 51,
		},
		{
			"hash 12 bit zero",
			"#000000000",
			0, 0, 0,
		},
		{
			"hash 12 bit max",
			"#fffffffff",
			255, 255, 255,
		},
		{
			"hash 12 bit values",
			"#111222333",
			17, 34, 51,
		},
		{
			"hash 16 bit zero",
			"#000000000000",
			0, 0, 0,
		},
		{
			"hash 16 bit max",
			"#ffffffffffff",
			255, 255, 255,
		},
		{
			"hash 16 bit values",
			"#111122223333",
			17, 34, 51,
		},
		{
			"rgb upper case",
			"RGB:0/A/F",
			0, 170, 255,
		},
		{
			"hash upper case",
			"#FFF",
			240, 240, 240,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, g, b, err := parseColor(tc.input)
			if err != nil {
				t.Fatalf("failed to parse color: %s", err)
			}

			if r != tc.r || g != tc.g || b != tc.b {
				t.Fatalf("expected (%d, %d, %d), got (%d, %d, %d)", tc.r, tc.g, tc.b, r, g, b)
			}
		})
	}
}

func TestOSCColorQueriesReplyWithMatchingTerminator(t *testing.T) {
	var reply bytes.Buffer
	term := New(WithWriter(&reply))

	writeAll(t, term, "\033]10;?\a\033]11;?\033\\\033]12;?\a")

	expected := oscColorReply(10, byte2color(int(LightGrey)), "\a") +
		oscColorReply(11, byte2color(int(Black)), "\033\\") +
		oscColorReply(12, byte2color(int(LightGrey)), "\a")
	if got := reply.String(); got != expected {
		t.Fatalf("unexpected OSC color replies: %q", got)
	}
}

func TestOSC4ColorQueryUsesPaletteIndex(t *testing.T) {
	var reply bytes.Buffer
	term := New(WithWriter(&reply))

	writeAll(t, term, "\033]4;1;rgb:12/34/56\a")
	reply.Reset()
	writeAll(t, term, "\033]4;1;?\a")

	if got := reply.String(); got != "\033]4;1;rgb:1212/3434/5656\a" {
		t.Fatalf("unexpected OSC 4 color reply: %q", got)
	}
}

func TestOSCCursorColorSetAndQuery(t *testing.T) {
	var reply bytes.Buffer
	term := New(WithWriter(&reply))

	writeAll(t, term, "\033]12;rgb:01/23/45\a")
	reply.Reset()
	writeAll(t, term, "\033]12;?\033\\")

	if got := reply.String(); got != "\033]12;rgb:0101/2323/4545\033\\" {
		t.Fatalf("unexpected OSC 12 color reply: %q", got)
	}
}
