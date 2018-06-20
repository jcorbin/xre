package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

// http://doc.cat-v.org/bell_labs/structural_regexps/se.pdf

type command func([]byte) error
type linker func(command) command

func scanX(s string) (linker, command, error) {
	sep := s[0]
	re, rest, err := scanPat(sep, s[1:])
	if err != nil {
		return nil, nil, err
	}
	switch re.NumSubexp() {
	case 0:
		return withNextCommand(rest, func(next command, buf []byte) error {
			for b := buf; len(b) > 0; {
				loc := re.FindIndex(b)
				if loc == nil {
					break
				}
				m := b[loc[0]:loc[1]] // extracted match
				if i := loc[1] + 1; i < len(b) {
					b = b[i:]
				} else {
					b = nil
				}
				if err := next(m); err != nil {
					return err
				}
			}
			return nil
		})

	case 1:
		return withNextCommand(rest, func(next command, buf []byte) error {
			for b := buf; len(b) > 0; {
				locs := re.FindSubmatchIndex(b)
				if locs == nil {
					break
				}
				m := b[locs[2]:locs[3]] // extracted match
				if i := locs[1] + 1; i < len(b) {
					b = b[i:]
				} else {
					b = nil
				}
				if err := next(m); err != nil {
					return err
				}
			}
			return nil
		})

	default:
		return nil, nil, errors.New("extraction sub-patterns not supported")
	}
}

func scanY(s string) (linker, command, error) {
	sep := s[0]

	start, rest, err := scanPat(sep, s[1:])
	if err != nil {
		return nil, nil, err
	}

	if len(rest) > 0 {
		end, cont, err := scanPat(sep, rest)
		if err != nil && err != errNoSep {
			return nil, nil, err
		}
		return withNextCommand(cont, func(next command, buf []byte) error {
			// TODO inclusive / exclusive control
			for b := buf; len(b) > 0; {
				// find start pattern
				loc := start.FindIndex(b)
				if loc == nil {
					break
				}
				m := b[loc[0]:] // extracted match (start)
				b = b[loc[1]+1:]

				// find end pattern
				off := loc[1] - loc[0]
				loc = end.FindIndex(b)
				if loc == nil {
					break
				}
				m = m[:off+loc[1]+1] // extracted match (end)
				b = b[loc[1]+1:]

				if err := next(m); err != nil {
					return err
				}
			}
			return nil
		})
	}

	return withNextCommand(rest, func(next command, buf []byte) error {
		// TODO inclusive / exclusive control
		b := buf
		for len(b) > 0 {
			loc := start.FindIndex(b)
			if loc == nil {
				break
			}
			m := b[:loc[1]+1] // extracted match
			b = b[loc[1]+1:]
			if err := next(m); err != nil {
				return err
			}
		}
		return next(b)
	})
}

func scanG(s string) (linker, command, error) {
	sep := s[0]
	re, rest, err := scanPat(sep, s[1:])
	if err != nil {
		return nil, nil, err
	}
	return withNextCommand(rest, func(next command, buf []byte) error {
		if re.Match(buf) {
			return next(buf)
		}
		return nil
	})
}

func scanV(s string) (linker, command, error) {
	sep := s[0]
	re, rest, err := scanPat(sep, s[1:])
	if err != nil {
		return nil, nil, err
	}
	return withNextCommand(rest, func(next command, buf []byte) error {
		if !re.Match(buf) {
			return next(buf)
		}
		return nil
	})
}

// TODO reconsider making quoting so first-class
func scanQ(s string) (linker, command, error) {
	w := os.Stdout // TODO support redirection

	if len(s) > 0 {
		return nil, nil, fmt.Errorf("unsupported %q to q command", s)
	}

	return nil, func(b []byte) error {
		_, err := fmt.Fprintf(w, "%q\n", b)
		return err
	}, nil
}

func scanP(s string) (linker, command, error) {
	w := os.Stdout // TODO support redirection

	var trailer string

	if s[0] == '"' {
		var err error
		trailer, s, err = scanDelim('"', s[1:])
		if err != nil {
			return nil, nil, err
		}
		trailer, err = strconv.Unquote("\"" + trailer + "\"")
		if err != nil {
			return nil, nil, err
		}
	}

	if len(s) > 0 {
		return nil, nil, fmt.Errorf("unsupported %q to p command", s)
	}

	if trailer != "" {
		return nil, func(b []byte) error {
			_, err := w.Write(b)
			if err == nil {
				_, err = io.WriteString(w, trailer)
			}
			return err
		}, nil
	}

	return nil, func(b []byte) error {
		_, err := w.Write(b)
		return err
	}, nil
}

func withNextCommand(rest string, f func(next command, buf []byte) error) (linker, command, error) {
	lnk := func(next command) command {
		return func(buf []byte) error {
			return f(next, buf)
		}
	}
	if rest == "" {
		return lnk, nil, nil
	}
	return scanCommand(lnk, rest)
}

func compileCommands(args []string) (cmd command, err error) {
	var lnk linker
	for _, arg := range args {
		lnk, cmd, err = scanCommand(lnk, arg)
		if err != nil {
			return nil, err
		}
		if cmd != nil {
			// TODO support multiple terminals
			return cmd, err
		}
	}
	if lnk != nil {
		// TODO default to print stdout?
		return nil, errors.New("unterminated command")
	}
	return nil, errors.New("no command given")
}

func scanOneCommand(s string) (linker, command, error) {
	switch s[0] {
	case 'x':
		return scanX(s[1:])
	case 'y':
		return scanY(s[1:])
	case 'g':
		return scanG(s[1:])
	case 'v':
		return scanV(s[1:])
	case 'q':
		return scanQ(s[1:])
	case 'p':
		return scanP(s[1:])
	default:
		return nil, nil, fmt.Errorf("unrecognized command %q", s[0])
	}
}

func scanCommand(under linker, s string) (linker, command, error) {
	lnk, cmd, err := scanOneCommand(s)
	if err != nil {
		return nil, nil, err
	}
	if lnk != nil {
		if under == nil {
			return lnk, cmd, nil
		}
		return func(cmd command) command {
			return under(lnk(cmd))
		}, nil, nil
	}
	if cmd == nil {
		panic("inconceivable")
	}
	if under != nil {
		return lnk, under(cmd), nil
	}
	return lnk, cmd, nil
}

var errNoSep = errors.New("missing separator")

func scanDelim(sep byte, r string) (part, rest string, err error) {
	// TODO support escaping
	i := strings.Index(r, string(sep))
	if i < 0 {
		return "", "", errNoSep
	}
	return r[:i], r[i+1:], nil
}

func scanPat(sep byte, r string) (*regexp.Regexp, string, error) {
	pat, rest, err := scanDelim(sep, r)
	if err != nil {
		return nil, "", err
	}

	if sep == '"' || sep == '\'' {
		pat = regexp.QuoteMeta(pat)
	}

	if len(rest) > 0 && rest[0] == 'i' {
		// TODO reconsider the case insensitivity affordance
		pat = "(?i:" + pat + ")"
		rest = rest[1:]
	}

	pat = "(?ms:" + pat + ")"
	re, err := regexp.Compile(pat)
	return re, rest, err
}

func run() error {
	buf, fin, err := mmap(os.Stdin)
	if err != nil {
		return err
	}
	defer fin()

	// TODO SIGPIPE handler

	flag.Parse()

	cmd, err := compileCommands(flag.Args())
	if err != nil {
		return err
	}
	return cmd(buf)
}

func mmap(f *os.File) ([]byte, func() error, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("mmap: cannot stat %q: %v", f.Name(), err)
	}

	size := fi.Size()
	if size <= 0 {
		return nil, nil, fmt.Errorf("mmap: file %q has negative size", f.Name())
	}
	if size != int64(int(size)) {
		return nil, nil, fmt.Errorf("mmap: file %q is too large", f.Name())
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, nil, err
	}
	return data, func() error {
		return syscall.Munmap(data)
	}, nil
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}
