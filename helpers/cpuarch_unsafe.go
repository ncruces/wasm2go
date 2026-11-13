package helpers

import "runtime"

// Compiler error if endianess is unknown.
var _ = map[bool]struct{}{big: {}, little: {}}

const (
	// go.dev/src/vendor/golang.org/x/sys/cpu/endian_big.go
	big = false ||
		runtime.GOARCH == "armbe" || runtime.GOARCH == "arm64be" ||
		runtime.GOARCH == "m68k" || runtime.GOARCH == "mips" ||
		runtime.GOARCH == "mips64" || runtime.GOARCH == "mips64p32" ||
		runtime.GOARCH == "ppc" || runtime.GOARCH == "ppc64" ||
		runtime.GOARCH == "s390" || runtime.GOARCH == "s390x" ||
		runtime.GOARCH == "shbe" || runtime.GOARCH == "sparc" ||
		runtime.GOARCH == "sparc64"

	// go.dev/src/vendor/golang.org/x/sys/cpu/endian_little.go
	little = false ||
		runtime.GOARCH == "386" || runtime.GOARCH == "amd64" ||
		runtime.GOARCH == "amd64p32" || runtime.GOARCH == "alpha" ||
		runtime.GOARCH == "arm" || runtime.GOARCH == "arm64" ||
		runtime.GOARCH == "loong64" || runtime.GOARCH == "mipsle" ||
		runtime.GOARCH == "mips64le" || runtime.GOARCH == "mips64p32le" ||
		runtime.GOARCH == "nios2" || runtime.GOARCH == "ppc64le" ||
		runtime.GOARCH == "riscv" || runtime.GOARCH == "riscv64" ||
		runtime.GOARCH == "sh" || runtime.GOARCH == "wasm" ||
		runtime.GOARCH == "wasm32"

	// go.dev/src/cmd/compile/internal/ssa/config.go
	unalignedOK = false ||
		runtime.GOARCH == "386" || runtime.GOARCH == "amd64" ||
		runtime.GOARCH == "arm64" || runtime.GOARCH == "loong64" ||
		runtime.GOARCH == "ppc64" || runtime.GOARCH == "ppc64le" ||
		runtime.GOARCH == "s390x" || runtime.GOARCH == "wasm" ||
		runtime.GOARCH == "wasm32"
)
