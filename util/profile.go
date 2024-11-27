package util

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
)

func CpuProfileFunc(file string, fu func()) error {
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("failed to create file for profiling, file: %v, %w", file, err)
	}
	f.Truncate(0)
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		return fmt.Errorf("failed to start cpu profile, %w", err)
	}
	defer pprof.StopCPUProfile()
	fu()
	return nil
}

func MemoryProfileFunc(file string, fu func()) error {
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("failed to create file for profiling, file: %v, %w", file, err)
	}
	f.Truncate(0)
	defer f.Close()

	runtime.GC()
	fu()
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("failed to write heap profile, %w", err)
	}
	return nil
}
