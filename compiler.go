package main

import (
	"errors"
	"fmt"
	"io"
	"regexp"
)

var errNoCommands = errors.New("no command(s) given")

type linker func(command) (command, error)

func compileCommands(args []string, w io.Writer) (cmd command, err error) {
	// TODO support more complex command pipelines than single straight lines
	var lnks []linker
	for _, arg := range args {
		more, err := scanCommand(arg)
		if err != nil {
			return nil, err
		}

		lnks = append(lnks, more...)

		if err != nil {
			return nil, err
		}
	}
	if len(lnks) == 0 {
		return nil, errNoCommands
	}

	cmd = writer{w}
	for i := len(lnks) - 1; i >= 0; i-- {
		cmd, err = lnks[i](cmd)
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}
	return cmd, nil
}

func xReLinker(pat *regexp.Regexp) (linker, error) {
	return func(next command) (command, error) {
		switch n := pat.NumSubexp(); n {
		case 0:
			return extract{pat, next}, nil

		case 1:
			return extractSub{pat, next}, nil

		default:
			return nil, fmt.Errorf("extraction with %v sub-patterns not supported", n)
		}
	}, nil
}

func xBalLinker(start, end byte, inc bool) (linker, error) {
	return func(next command) (command, error) {
		if inc {
			return extractBalancedInc{start, end, next}, nil
		}
		return extractBalanced{start, end, next}, nil
	}, nil
}

func yLinker(start, end *regexp.Regexp) (linker, error) {
	return func(next command) (command, error) {
		if end != nil {
			return between{start, end, next}, nil
		}
		return betweenDelim{start, next}, nil
	}, nil
}

func gLinker(pat *regexp.Regexp, negate bool) (linker, error) {
	return func(next command) (command, error) {
		if negate {
			return filterNeg{pat, next}, nil
		}
		return filter{pat, next}, nil
	}, nil
}

func pLinker(format string, delim []byte) (linker, error) {
	return func(next command) (command, error) {
		if format != "" && delim != nil {
			format, delim = fmt.Sprintf("%s%s", format, delim), nil
		}
		if format == "" && len(delim) == 0 {
			return next, nil
		}
		// from here on we have either have format or delim

		switch nc := next.(type) {
		case writer:
			if format != "" {
				return fmtWriter{fmt: format, w: nc.w}, nil
			}
			return delimWriter{delim: delim, w: nc.w}, nil

		case fmtWriter:
			if format != "" {
				return fmtWriter{fmt: format + nc.fmt, w: nc.w}, nil
			}
			return delimWriter{delim: delim, w: nc.w}, nil

		case delimWriter:
			if format != "" {
				return fmtWriter{fmt: format, w: nc.w}, nil
			}
			return delimWriter{delim: append(delim, nc.delim...), w: nc.w}, nil

		default:
			if format != "" {
				return fmter{fmt: format, next: next}, nil
			}
			return delimer{delim: delim, next: next}, nil
		}
	}, nil
}
