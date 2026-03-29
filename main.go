package main

import (
	"flag"
	"log"
	"os"
)

var (
	endian = flag.String("endian", "", "endianness of the generated code (little or big)")
	nanbox = flag.Bool("nanbox", false, "whether to try to canonicalize NaNs")
	nohead = flag.Bool("nohead", false, "disable the header comment (including build tags)")
	nohost = flag.Bool("nohost", false, "disable generating interfaces for imports")
	noopt  = flag.Bool("noopt", false, "disable all optimization passes")
)

// Architectures that are natively little-endian AND unalignedOK:
// https://github.com/golang/go/blob/master/src/cmd/compile/internal/ssagen/ssa.go
const littlend = "386 || amd64 || arm || arm64 || loong64 || ppc64le || wasm"

func main() {
	flag.Parse()
	err := translate(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
