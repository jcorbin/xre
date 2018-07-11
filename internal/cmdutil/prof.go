package cmdutil

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"runtime/trace"
)

var (
	profCPUFlag   = flag.String("cpuprofile", "", "write cpu profiling to given file")
	profMemFlag   = flag.String("memprofile", "", "write mem profiling to given file")
	profTraceFlag = flag.String("tracefile", "", "write execution trace to given file")
)

// WithProf runs the given function under whatever profiling the user has
// selected via command line flags. Any error by the function, or profiling
// error, is returned.
func WithProf(fn func() error) (rerr error) {
	var cpuFile, memFile, traceFile *os.File

	defer func() {
		if cpuFile != nil {
			if cerr := cpuFile.Close(); rerr == nil {
				rerr = cerr
			}
		}
		if memFile != nil {
			if cerr := memFile.Close(); rerr == nil {
				rerr = cerr
			}
		}
		if traceFile != nil {
			if cerr := traceFile.Close(); rerr == nil {
				rerr = cerr
			}
		}
	}()

	for _, ff := range []struct {
		flag *string
		fp   **os.File
	}{
		{profCPUFlag, &cpuFile},
		{profMemFlag, &memFile},
		{profTraceFlag, &traceFile},
	} {
		if name := *ff.flag; name != "" {
			f, err := os.Create(name)
			if err != nil {
				return err
			}
			*ff.fp = f
		}
	}

	if cpuFile != nil {
		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			return fmt.Errorf("could not start CPU profiling: %v", err)
		}
		log.Printf("cpu profiling to %s", cpuFile.Name())
		defer pprof.StopCPUProfile()
	}

	if memFile != nil {
		defer func() {
			if rerr == nil {
				return
			}
			if err := pprof.WriteHeapProfile(memFile); err != nil {
				rerr = fmt.Errorf("could not write memory profile: %v", err)
			}
			log.Printf("memory profiling to %s", memFile.Name())
		}()
	}

	if traceFile != nil {
		if err := trace.Start(traceFile); err != nil {
			return fmt.Errorf("could not start execution trace: %v", err)
		}
		log.Printf("tracing profiling to %s", traceFile.Name())
		defer trace.Stop()
	}

	return fn()
}
