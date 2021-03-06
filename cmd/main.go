// Inspired by http://doc.cat-v.org/bell_labs/structural_regexps/se.pdf

package main

import (
	"bufio"
	"flag"
	"log"
	"os"

	"github.com/jcorbin/xre"
	"github.com/jcorbin/xre/internal/cmdutil"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

var (
	listIn  = false
	mainEnv = xre.Stdenv // TODO support redirection
)

func run() (rerr error) {
	flag.BoolVar(&listIn, "l", false, "read list of input filenames from stdin or given argument files")
	flag.Parse()

	// TODO SIGPIPE handler

	args := flag.Args()

	var prog string
	if len(args) > 0 {
		prog = args[0]
		args = args[1:]
	}

	if listIn {
		scanInfiles(args)
	} else {
		passArgfiles(args)
	}

	return cmdutil.WithProf(func() error {
		return xre.RunCommand(prog, &mainEnv)
	})
}

func passArgfiles(args []string) {
	if len(args) > 0 {
		mainEnv.AddInput(os.Open(args[0]))
		go func() {
			defer mainEnv.CloseInputs()
			for _, arg := range args[1:] {
				mainEnv.AddInput(os.Open(arg))
			}
		}()
	}
}

func scanInfiles(args []string) {
	mainEnv.AddInput(nil, nil)
	go func() {
		defer mainEnv.CloseInputs()
		if len(args) > 0 {
			for _, arg := range args {
				scanInfile(os.Open(arg))
			}
		} else {
			scanInfile(mainEnv.DefaultInfile, nil)
		}
	}()
}

func scanInfile(f *os.File, err error) {
	if err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			mainEnv.AddInput(os.Open(sc.Text()))
		}
		err = sc.Err()
	}
	mainEnv.AddInput(nil, err)
	if f != nil {
		mainEnv.AddInput(nil, f.Close())
	}
}
