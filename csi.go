package vt10x

import (
	"fmt"
	"strconv"
	"strings"
)

// CSI (Control Sequence Introducer)
// ESC+[
type csiEscape struct {
	buf    []byte
	args   []int
	mode   byte
	prefix byte
	interm string
	priv   bool
}

func (c *csiEscape) reset() {
	c.buf = c.buf[:0]
	c.args = c.args[:0]
	c.mode = 0
	c.prefix = 0
	c.interm = ""
	c.priv = false
}

func (c *csiEscape) put(b byte) bool {
	c.buf = append(c.buf, b)
	if b >= 0x40 && b <= 0x7E || len(c.buf) >= 256 {
		c.parse()
		return true
	}
	return false
}

func (c *csiEscape) parse() {
	c.mode = c.buf[len(c.buf)-1]
	if len(c.buf) == 1 {
		return
	}
	b := c.buf[:len(c.buf)-1]
	c.args = c.args[:0]
	if len(b) > 0 && (b[0] == '?' || b[0] == '>' || b[0] == '<' || b[0] == '=') {
		c.prefix = b[0]
		c.priv = b[0] == '?'
		b = b[1:]
	}
	params := b
	for i, ch := range b {
		if (ch < '0' || ch > '9') && ch != ';' {
			params = b[:i]
			c.interm = string(b[i:])
			break
		}
	}
	if len(params) == 0 {
		return
	}
	ss := strings.Split(string(params), ";")
	for _, p := range ss {
		i, err := strconv.Atoi(p)
		if err != nil {
			//t.logf("invalid CSI arg '%s'\n", p)
			break
		}
		c.args = append(c.args, i)
	}
}

func (c *csiEscape) arg(i, def int) int {
	if i >= len(c.args) || i < 0 {
		return def
	}
	return c.args[i]
}

// maxarg takes the maximum of arg(i, def) and def
func (c *csiEscape) maxarg(i, def int) int {
	return max(c.arg(i, def), def)
}

func (t *State) handleCSI() {
	c := &t.csi
	if c.interm != "" {
		if t.handleIntermediateCSI() {
			return
		}
		goto unknown
	}

	if c.prefix != 0 {
		if t.handlePrefixedCSI() {
			return
		}
		goto unknown
	}

	switch c.mode {
	default:
		goto unknown
	case '@': // ICH - insert <n> blank char
		t.insertBlanks(c.arg(0, 1))
	case 'A': // CUU - cursor <n> up
		t.moveTo(t.cur.X, t.cur.Y-c.maxarg(0, 1))
	case 'B', 'e': // CUD, VPR - cursor <n> down
		t.moveTo(t.cur.X, t.cur.Y+c.maxarg(0, 1))
	case 'c': // DA - device attributes
		if c.arg(0, 0) == 0 {
			t.replyPrimaryDA()
		} else {
			goto unknown
		}
	case 'C', 'a': // CUF, HPR - cursor <n> forward
		t.moveTo(t.cur.X+c.maxarg(0, 1), t.cur.Y)
	case 'D': // CUB - cursor <n> backward
		t.moveTo(t.cur.X-c.maxarg(0, 1), t.cur.Y)
	case 'E': // CNL - cursor <n> down and first col
		t.moveTo(0, t.cur.Y+c.arg(0, 1))
	case 'F': // CPL - cursor <n> up and first col
		t.moveTo(0, t.cur.Y-c.arg(0, 1))
	case 'g': // TBC - tabulation clear
		switch c.arg(0, 0) {
		// clear current tab stop
		case 0:
			t.tabs[t.cur.X] = false
		// clear all tabs
		case 3:
			for i := range t.tabs {
				t.tabs[i] = false
			}
		default:
			goto unknown
		}
	case 'G', '`': // CHA, HPA - Move to <col>
		t.moveTo(c.arg(0, 1)-1, t.cur.Y)
	case 'H', 'f': // CUP, HVP - move to <row> <col>
		t.moveAbsTo(c.arg(1, 1)-1, c.arg(0, 1)-1)
	case 'I': // CHT - cursor forward tabulation <n> tab stops
		n := c.arg(0, 1)
		for i := 0; i < n; i++ {
			t.putTab(true)
		}
	case 'J': // ED - clear screen
		// TODO: sel.ob.x = -1
		switch c.arg(0, 0) {
		case 0: // below
			t.clear(t.cur.X, t.cur.Y, t.cols-1, t.cur.Y)
			if t.cur.Y < t.rows-1 {
				t.clear(0, t.cur.Y+1, t.cols-1, t.rows-1)
			}
		case 1: // above
			if t.cur.Y > 1 {
				t.clear(0, 0, t.cols-1, t.cur.Y-1)
			}
			t.clear(0, t.cur.Y, t.cur.X, t.cur.Y)
		case 2: // all
			t.clear(0, 0, t.cols-1, t.rows-1)
		default:
			goto unknown
		}
	case 'K': // EL - clear line
		switch c.arg(0, 0) {
		case 0: // right
			t.clear(t.cur.X, t.cur.Y, t.cols-1, t.cur.Y)
		case 1: // left
			t.clear(0, t.cur.Y, t.cur.X, t.cur.Y)
		case 2: // all
			t.clear(0, t.cur.Y, t.cols-1, t.cur.Y)
		}
	case 'S': // SU - scroll <n> lines up
		t.scrollUp(t.top, c.arg(0, 1))
	case 'T': // SD - scroll <n> lines down
		t.scrollDown(t.top, c.arg(0, 1))
	case 'L': // IL - insert <n> blank lines
		t.insertBlankLines(c.arg(0, 1))
	case 'l': // RM - reset mode
		t.setMode(false, false, c.args)
	case 'M': // DL - delete <n> lines
		t.deleteLines(c.arg(0, 1))
	case 'X': // ECH - erase <n> chars
		t.clear(t.cur.X, t.cur.Y, t.cur.X+c.arg(0, 1)-1, t.cur.Y)
	case 'P': // DCH - delete <n> chars
		t.deleteChars(c.arg(0, 1))
	case 'Z': // CBT - cursor backward tabulation <n> tab stops
		n := c.arg(0, 1)
		for i := 0; i < n; i++ {
			t.putTab(false)
		}
	case 'd': // VPA - move to <row>
		t.moveAbsTo(t.cur.X, c.arg(0, 1)-1)
	case 'h': // SM - set terminal mode
		t.setMode(false, true, c.args)
	case 'm': // SGR - terminal attribute (color)
		t.setAttr(c.args)
	case 'n':
		switch c.arg(0, 0) {
		case 5: // DSR - device status report
			t.w.Write([]byte("\033[0n"))
		case 6: // CPR - cursor position report
			t.w.Write([]byte(fmt.Sprintf("\033[%d;%dR", t.cur.Y+1, t.cur.X+1)))
		default:
			goto unknown
		}
	case 'r': // DECSTBM - set scrolling region
		t.setScroll(c.arg(0, 1)-1, c.arg(1, t.rows)-1)
		t.moveAbsTo(0, 0)
	case 's': // DECSC - save cursor position (ANSI.SYS)
		t.saveCursor()
	case 'u': // DECRC - restore cursor position (ANSI.SYS)
		t.restoreCursor()
	}
	return
unknown: // TODO: get rid of this goto
	t.logUnknownCSI(c)
	// TODO: c.dump()
}

func (t *State) handleIntermediateCSI() bool {
	c := &t.csi
	switch c.interm {
	case "$":
		switch c.mode {
		case 'p': // DECRQM / ANSI RMQ - request mode
			if len(c.args) == 0 {
				return false
			}
			t.replyMode(c.priv, c.arg(0, 0), t.modeStatus(c.priv, c.arg(0, 0)))
			return true
		}
	}

	return false
}

func (t *State) handlePrefixedCSI() bool {
	c := &t.csi
	switch c.prefix {
	case '?':
		switch c.mode {
		case 'h':
			t.setMode(true, true, c.args)
			return true
		case 'l':
			t.setMode(true, false, c.args)
			return true
		case 'n':
			switch c.arg(0, 0) {
			case 6: // DECXCPR - DEC-specific cursor position report
				_, _ = t.w.Write([]byte(fmt.Sprintf("\033[?%d;%dR", t.cur.Y+1, t.cur.X+1)))
				return true
			}
		}
	case '>':
		switch c.mode {
		case 'c': // DA2 - secondary device attributes
			if c.arg(0, 0) == 0 {
				_, _ = t.w.Write([]byte(t.secondaryDA))
				return true
			}
		}
	}

	return false
}

func (t *State) replyPrimaryDA() {
	// Reply with an xterm-style primary DA for compatibility with modern TUIs.
	// Strict VT102 would traditionally report ESC[?6c here.
	_, _ = t.w.Write([]byte("\033[?1;2c"))
}

func (t *State) replyMode(priv bool, mode, status int) {
	prefix := ""
	if priv {
		prefix = "?"
	}
	_, _ = t.w.Write([]byte(fmt.Sprintf("\033[%s%d;%d$y", prefix, mode, status)))
}

func (t *State) modeStatus(priv bool, mode int) int {
	set := false
	known := true

	if priv {
		switch mode {
		case 1:
			set = t.mode&ModeAppCursor != 0
		case 6:
			set = t.cur.State&cursorOrigin != 0
		case 7:
			set = t.mode&ModeWrap != 0
		case 25:
			set = t.mode&ModeHide == 0
		case 47, 1047, 1049:
			set = t.mode&ModeAltScreen != 0
		case 66:
			set = t.mode&ModeAppKeypad != 0
		case 1000:
			set = t.mode&ModeMouseButton != 0
		case 1002:
			set = t.mode&ModeMouseMotion != 0
		case 1003:
			set = t.mode&ModeMouseMany != 0
		case 1004:
			set = t.mode&ModeFocus != 0
		case 1006:
			set = t.mode&ModeMouseSgr != 0
		case 1034:
			set = t.mode&Mode8bit != 0
		case 2004:
			set = t.mode&ModeBracketedPaste != 0
		default:
			known = false
		}
	} else {
		switch mode {
		case 2:
			set = t.mode&ModeKeyboardLock != 0
		case 4:
			set = t.mode&ModeInsert != 0
		case 12:
			set = t.mode&ModeEcho != 0
		case 20:
			set = t.mode&ModeCRLF != 0
		default:
			known = false
		}
	}

	if !known {
		return 0
	}
	if set {
		return 1
	}
	return 2
}

func (t *State) logUnknownCSI(c *csiEscape) {
	if c.prefix != 0 || c.interm != "" {
		t.logf("unknown CSI sequence '%c%s%c'\n", c.prefix, c.interm, c.mode)
		return
	}
	t.logf("unknown CSI sequence '%c'\n", c.mode)
}
