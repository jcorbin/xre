package main

import (
	"fmt"
	"regexp"
	"strings"
)

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
func (bd betweenDelim) String() string {
	return fmt.Sprintf("y/%v/%v", regexpString(bd.pat), bd.next)
}

func (fl filter) String() string    { return fmt.Sprintf("g/%v/%v", regexpString(fl.pat), fl.next) }
func (fn filterNeg) String() string { return fmt.Sprintf("v/%v/%v", regexpString(fn.pat), fn.next) }

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
