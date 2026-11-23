//go:build ignore

// Read build metadata from an ELF binary without executing it.
package main

import (
	"debug/buildinfo"
	"debug/elf"
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: read_go_buildinfo BINARY")
		os.Exit(2)
	}
	info, err := buildinfo.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	binary, err := elf.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer binary.Close()
	result := struct {
		*debug.BuildInfo
		ELFMachine string
	}{info, binary.Machine.String()}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
