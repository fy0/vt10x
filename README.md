# vt10x

[![Build Status](https://travis-ci.org/hinshun/vt10x.svg?branch=master)](https://travis-ci.org/hinshun/vt10x)
[![GoDoc](https://godoc.org/github.com/hinshun/vt10x?status.svg)](https://godoc.org/github.com/hinshun/vt10x)

Package vt10x is a vt10x terminal emulation backend, influenced
largely by st, rxvt, xterm, and iTerm as reference. Use it for terminal
muxing, a terminal emulation frontend, or wherever else you need
terminal emulation. It also answers common xterm-style startup probes
such as CPR, primary/secondary DA, and OSC 10/11/12 color queries.
Secondary DA stays conservative by default, and callers can opt into a
broader xterm-oriented compatibility profile with `WithXtermStyle()`.

## Partial xterm compatibility

vt10x is not a full xterm clone, but it intentionally supports a growing
set of xterm-style startup and query behaviors used by modern TUIs.

Currently supported xterm-oriented behavior includes:

- `CSI c` primary DA reply with an xterm-style `ESC[?1;2c` response.
- `ESC Z` DECID compatibility, mapped to the same primary DA reply.
- `CSI > c` secondary DA reply, conservative by default and xterm.js-like
  when `WithXtermStyle()` is enabled.
- `CSI 6n` CPR and `CSI ? 6n` DECXCPR cursor position reports.
- `CSI Ps $ p` and `CSI ? Ps $ p` request-mode reports for supported ANSI and
  DEC private modes, including bracketed paste and focus reporting.
- `OSC 10;?`, `OSC 11;?`, and `OSC 12;?` foreground/background/cursor
  color queries.
- OSC color replies that mirror the incoming BEL vs ST terminator.
- Prefixed CSI parsing that does not fall through to ordinary non-prefixed
  handlers when the prefixed form is unsupported.

This keeps vt10x conservative by default while making it easier to move
closer to xterm / xterm.js behavior over time.
