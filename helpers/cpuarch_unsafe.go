package helpers

import "runtime"

// Compiler error if endianess is unknown.
var _ = map[bool]struct{}{big: {}, little: {}}

// go.dev/src/cmd/compile/internal/ssa/config.go
const (
	big = false ||
		runtime.GOARCH == "ppc64" || runtime.GOARCH == "s390x" ||
		runtime.GOARCH == "mips" || runtime.GOARCH == "mips64"

	little = false ||
		runtime.GOARCH == "386" || runtime.GOARCH == "amd64" ||
		runtime.GOARCH == "arm" || runtime.GOARCH == "arm64" ||
		runtime.GOARCH == "riscv64" || runtime.GOARCH == "wasm" ||
		runtime.GOARCH == "ppc64le" || runtime.GOARCH == "loong64" ||
		runtime.GOARCH == "mipsle" || runtime.GOARCH == "mips64le"

	unalignedOK = false ||
		runtime.GOARCH == "386" || runtime.GOARCH == "amd64" ||
		runtime.GOARCH == "arm64" || runtime.GOARCH == "loong64" ||
		runtime.GOARCH == "ppc64" || runtime.GOARCH == "ppc64le" ||
		runtime.GOARCH == "s390x" || runtime.GOARCH == "wasm"
)
