package xre

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

type scanner func(string) (Command, string, error)

var commands = map[byte]scanner{
	'x': scanX,
	'y': scanY,
	'g': scanG,
	'v': scanV,
	'p': scanP,
}

// Command represents a piece of potential XRE processing which; combining it
// with an Environment realizes said potential, resulting in a Processor.
type Command interface {
	Create(next Command, env Environment) (Processor, error)
}

// ParseCommand parses an XRE command from the given string, returning any
// parse error if the string is invalid.
func ParseCommand(s string) (Command, error) {
	cmd, s, err := scanCommand(s)
	if err != nil {
		return nil, err
	}
	if s != "" {
		return nil, fmt.Errorf("extraneous input %q after command", s)
	}
	return cmd, nil
}

func scanCommand(s string) (Command, string, error) {
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

func scanCommandAtom(s string) (Command, string, error) {
	if s == "" {
		return nil, s, errors.New("missing command at end of input")
	}
	scan, def := commands[s[0]]
	if !def {
		return nil, s, fmt.Errorf("unrecognized command %q", s[0])
	}
	return scan(s[1:])
}

func createProcessor(cmd Command, env Environment) (Processor, error) {
	if cmd == nil {
		return env.Default(), nil
	}
	return cmd.Create(nil, env)
}

func createProcessorIO(cmd Command, env Environment) (processorIO, error) {
	proc, err := createProcessor(cmd, env)
	if err != nil {
		return nil, err
	}
	if procio, canio := proc.(processorIO); canio {
		return procio, nil
	}
	// TODO scrap this adaptor, it's insane
	return procIOAdaptor{Processor: proc}, nil
}

// RunCommand runs the given command, processing all bytes available, unless an
// error occurs (reading, processing, or writing); any such error is returned.
func RunCommand(cmd Command, r io.Reader, env Environment) error {
	procio, err := createProcessorIO(cmd, env)
	if err == nil {
		_, err = procio.ReadFrom(r)
	}
	return err
}

type procIOAdaptor struct {
	Processor
	buf readBuf
}

func (proc procIOAdaptor) ReadFrom(r io.Reader) (int64, error) {
	return proc.buf.ProcessFrom(r, func(buf *readBuf) error {
		err := proc.Process(buf.Bytes(), buf.Err() != nil)
		buf.Advance(buf.Len())
		return err
	})
}

type commandChain []Command

func chain(a, b Command) Command {
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

func (cc commandChain) Create(nc Command, env Environment) (Processor, error) {
	if len(cc) == 0 {
		return createProcessor(nc, env)
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

flagScan:
	for len(rest) > 0 {
		switch rest[0] {
		case 'i', 's', 'U':
			pat = fmt.Sprintf("(?%s:%s)", rest[:1], pat)
			rest = rest[1:]
		default:
			break flagScan
		}
	}

	pat = "(?m:" + pat + ")"
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

flagScan:
	for len(s) > 5 && strings.HasPrefix(s, "(?") && s[3] == ':' {
		switch s[2] {
		case 'i', 's', 'U':
			flags += s[2:3]
			fallthrough
		case 'm':
			s = s[4 : len(s)-1]
		default:
			break flagScan
		}
	}

	return fmt.Sprintf("/%s/%s", s, flags)
}
