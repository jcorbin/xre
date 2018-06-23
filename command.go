package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

type command interface {
	Process(buf []byte, ateof bool) (off int, err error)
}

type filelike interface {
	Name() string
	Stat() (os.FileInfo, error)
	Fd() uintptr
}

func runCommand(cmd command, r io.Reader, useMmap bool) error {
	if f, canMmap := r.(filelike); useMmap && canMmap {
		buf, fin, err := mmap(f)
		if err == nil {
			defer fin()
			_, err = cmd.Process(buf, true)
		}
		return err
	}

	if rf, canReadFrom := cmd.(io.ReaderFrom); canReadFrom {
		_, err := rf.ReadFrom(r)
		return err
	}

	// TODO if (some) commands implement io.Writer, then could upgrade to r.(WriterTo)

	rb := readBuf{buf: make([]byte, 0, minRead)} // TODO configurable buffer size
	return rb.Process(cmd, r)
}

func mmap(f filelike) ([]byte, func() error, error) {
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

//// parsing

var errNoSep = errors.New("missing separator")

type scanner func(string) (linker, string, error) // TODO consider upgrading to []linker

var commands = map[byte]scanner{
	'x': scanX,
	'y': scanY,
	'g': scanG,
	'v': scanV,
	'p': scanP,
}

// NOTE not actually a "scanner" due to needing to de-confuse the `type scanner` as noted above.
func scanCommand(s string) ([]linker, error) {
	var lnks []linker
	for len(s) > 0 {
		scan, def := commands[s[0]]
		if !def {
			return nil, fmt.Errorf("unrecognized command %q", s[0])
		}
		lnk, cont, err := scan(s[1:])
		if err != nil {
			return nil, err
		}
		lnks = append(lnks, lnk)
		s = cont
	}
	return lnks, nil
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
	s := re.String()
	// TODO share with scanPat that adds the prefix
	if strings.HasPrefix(s, "(?ms:") {
		s = s[5:]
		s = s[:len(s)-1]
	}
	return s
}