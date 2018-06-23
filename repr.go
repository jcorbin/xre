package main

import (
	"fmt"
	"regexp"
	"strings"
)

func (ar addrRange) String() string { return fmt.Sprintf("%v:%v%v", ar.start, ar.end, ar.next) }

func (ex extract) String() string    { return fmt.Sprintf("x/%v/%v", regexpString(ex.pat), ex.next) }
func (ex extractSub) String() string { return fmt.Sprintf("x/%v/%v", regexpString(ex.pat), ex.next) }
func (ex extractBalanced) String() string {
	return fmt.Sprintf("x%s%v", string(ex.open), ex.next)
}
func (ex extractBalancedInc) String() string {
	return fmt.Sprintf("x%s%s%v", string(ex.open), string(ex.close), ex.next)
}

func (by between) String() string {
	return fmt.Sprintf("y/%v/%v/%v", regexpString(by.start), regexpString(by.end), by.next)
}
func (bd betweenDelimRe) String() string {
	return fmt.Sprintf("y/%v/%v", regexpString(bd.pat), bd.next)
}
func (bd betweenDelimSplit) String() string {
	return fmt.Sprintf("y%v%v", bd.split, bd.next)
}

func (ls lineSplitter) String() string        { return fmt.Sprintf("%q", strings.Repeat(`\n`, int(ls))) }
func (bs byteSplitter) String() string        { return fmt.Sprintf("%q", string(bs)) }
func (bss bytesSplitter) String() string      { return fmt.Sprintf("%q", []byte(bss)) }
func (bst byteSplitTrimmer) String() string   { return fmt.Sprintf("%q~%q", bst.delim, bst.cutset) }
func (bsst bytesSplitTrimmer) String() string { return fmt.Sprintf("%q~%q", bsst.delim, bsst.cutset) }

func (fl filter) String() string    { return fmt.Sprintf("g/%v/%v", regexpString(fl.pat), fl.next) }
func (fn filterNeg) String() string { return fmt.Sprintf("v/%v/%v", regexpString(fn.pat), fn.next) }

func (ac accum) String() string       { return fmt.Sprintf("a%s", fmt.Sprint(ac.next)[1:]) }
func (fr fmter) String() string       { return fmt.Sprintf("p%%%q%v", fr.fmt, fr.next) }
func (dr delimer) String() string     { return fmt.Sprintf("p%q%v", dr.delim, dr.next) }
func (wr writer) String() string      { return "p" }
func (fw fmtWriter) String() string   { return fmt.Sprintf("p%%%q", fw.fmt) }
func (dw delimWriter) String() string { return fmt.Sprintf("p%q", dw.delim) }

func regexpString(re *regexp.Regexp) string {
	s := re.String()
	// TODO share with scanPat that adds the prefix
	if strings.HasPrefix(s, "(?ms:") {
		s = s[5:]
		s = s[:len(s)-1]
	}
	return s
}
