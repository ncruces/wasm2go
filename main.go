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

// https://pkg.go.dev/golang.org/x/sys/cpu#pkg-constants
const littlend = "386 || amd64 || amd64p32 || alpha || arm || arm64 || loong64 || mipsle || mips64le || mips64p32le || nios2 || ppc64le || riscv || riscv64 || sh || wasm"

func main() {
	flag.Parse()
	err := translate(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
