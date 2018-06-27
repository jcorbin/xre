package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var errNoSep = errors.New("missing separator")

type scanner func(string) (command, string, error)

var commands = map[byte]scanner{
	'x': scanX,
	'y': scanY,
	'g': scanG,
	'v': scanV,
	'p': scanP,
}

func parseCommand(s string) (command, error) {
	cmd, s, err := scanCommand(s)
	if err != nil {
		return nil, err
	}
	if s != "" {
		return nil, fmt.Errorf("extraneous input %q after command", s)
	}
	return cmd, nil
}

func scanCommand(s string) (command, string, error) {
	cmd := chain(nil, nil)
	for len(s) > 0 {
		switch s[0] {
		case ' ', '\t', '\r', '\n':
			s = s[1:]
			continue

			// case '{', '}': // TODO grouping support

		default:
			nextCmd, cont, err := scanCommandAtom(s)
			if err != nil {
				return cmd, s, err
			}
			s, cmd = cont, chain(cmd, nextCmd)
		}
	}
	return cmd, s, nil
}

func scanCommandAtom(s string) (command, string, error) {
	if s == "" {
		return nil, s, errors.New("missing command at end of input")
	}
	scan, def := commands[s[0]]
	if !def {
		return nil, s, fmt.Errorf("unrecognized command %q", s[0])
	}
	return scan(s[1:])
}

func runCommand(cmd command, r io.Reader, env environment) error {
	proc, err := create(cmd, env)
	if err == nil {
		err = runProcessor(proc, r)
	}
	return err
}

func runProcessor(proc processor, r io.Reader) error {
	if rf, canReadFrom := proc.(io.ReaderFrom); canReadFrom {
		_, err := rf.ReadFrom(r)
		return err
	}

	// TODO if (some) commands implement io.Writer, then could upgrade to r.(WriterTo)

	rb := readBuf{buf: make([]byte, 0, minRead)} // TODO configurable buffer size
	return rb.Process(proc, r)
}

type command interface {
	Create(command, environment) (processor, error)
}

type processor interface {
	Process(buf []byte, ateof bool) (off int, err error)
}

type commandChain []command

func chain(a, b command) command {
	if a == nil && b == nil {
		return commandChain(nil)
	} else if a == nil {
		return b
	} else if b == nil {
		return a
	}

	as, isAChain := a.(commandChain)
	bs, isBChain := b.(commandChain)
	if isAChain && isBChain {
		if len(as) == 0 {
			return bs
		}
		return append(as, bs...)
	} else if isAChain {
		return append(as, b)
	} else if isBChain {
		bs = append(bs, nil)
		copy(bs[1:], bs)
		bs[0] = a
	}

	return commandChain{a, b}
}

func (cc commandChain) Create(nc command, env environment) (processor, error) {
	if len(cc) == 0 {
		return create(nc, env)
	}
	head := cc[0]
	tail := cc[:copy(cc, cc[1:])]
	if nc != nil {
		tail = append(tail, nc)
	}
	if len(tail) == 0 {
		tail = nil
	}
	return head.Create(tail, env)
}

func (cc commandChain) String() string {
	buf := bytes.NewBuffer(make([]byte, 0, 4*len(cc)))
	for i, c := range cc {
		if i > 0 {
			_ = buf.WriteByte(' ')
		}
		_, _ = fmt.Fprint(buf, c)
	}
	return buf.String()
}

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

func scanString(sep byte, s string) (val, rest string, err error) {
	if val, s, err = scanDelim(sep, s); err == nil {
		val = fmt.Sprintf("%s%s%s", string(sep), val, string(sep))
		val, err = strconv.Unquote(val)
	}
	return val, s, err
}

func regexpString(re *regexp.Regexp) string {
	flags := ""
	s := re.String()
	// TODO share with scanPat that adds the prefix
	if strings.HasPrefix(s, "(?ms:") {
		s = s[5:]
		s = s[:len(s)-1]
	}
	if strings.HasPrefix(s, "(?i:") {
		s = s[4:]
		s = s[:len(s)-1]
		flags += "i"
	}
	return fmt.Sprintf("/%s/%s", s, flags)
}
