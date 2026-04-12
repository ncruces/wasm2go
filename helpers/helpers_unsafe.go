//go:build ignore

package helpers

import (
	"encoding/binary"
	"math/bits"
	"runtime"
	"unsafe"
)

// Faster memory access, using unsafe.

// Architectures that are unalignedOK:
// go.dev/src/cmd/compile/internal/ssa/config.go

//go:nosplit
func load16(b []byte) uint16 {
	switch runtime.GOARCH {
	case "386", "amd64", "arm64", "loong64", "ppc64", "ppc64le", "s390x", "wasm":
		v := *(*uint16)(unsafe.Pointer((*[2]byte)(b)))
		switch runtime.GOARCH {
		case "ppc64", "s390x":
			return bits.ReverseBytes16(v)
		}
		return v
	}
	return binary.LittleEndian.Uint16(b)
}

//go:nosplit
func store16(b []byte, v uint16) {
	switch runtime.GOARCH {
	default:
		binary.LittleEndian.PutUint16(b, v)
	case "ppc64", "s390x":
		v = bits.ReverseBytes16(v)
		fallthrough
	case "386", "amd64", "arm64", "loong64", "ppc64le", "wasm":
		*(*uint16)(unsafe.Pointer((*[2]byte)(b))) = v
	}
}

//go:nosplit
func load32(b []byte) uint32 {
	switch runtime.GOARCH {
	case "386", "amd64", "arm64", "loong64", "ppc64", "ppc64le", "s390x", "wasm":
		v := *(*uint32)(unsafe.Pointer((*[4]byte)(b)))
		switch runtime.GOARCH {
		case "ppc64", "s390x":
			return bits.ReverseBytes32(v)
		}
		return v
	}
	return binary.LittleEndian.Uint32(b)
}

//go:nosplit
func store32(b []byte, v uint32) {
	switch runtime.GOARCH {
	default:
		binary.LittleEndian.PutUint32(b, v)
	case "ppc64", "s390x":
		v = bits.ReverseBytes32(v)
		fallthrough
	case "386", "amd64", "arm64", "loong64", "ppc64le", "wasm":
		*(*uint32)(unsafe.Pointer((*[4]byte)(b))) = v
	}
}

//go:nosplit
func load64(b []byte) uint64 {
	switch runtime.GOARCH {
	case "386", "amd64", "arm64", "loong64", "ppc64", "ppc64le", "s390x", "wasm":
		v := *(*uint64)(unsafe.Pointer((*[8]byte)(b)))
		switch runtime.GOARCH {
		case "ppc64", "s390x":
			return bits.ReverseBytes64(v)
		}
		return v
	}
	return binary.LittleEndian.Uint64(b)
}

//go:nosplit
func store64(b []byte, v uint64) {
	switch runtime.GOARCH {
	default:
		binary.LittleEndian.PutUint64(b, v)
	case "ppc64", "s390x":
		v = bits.ReverseBytes64(v)
		fallthrough
	case "386", "amd64", "arm64", "loong64", "ppc64le", "wasm":
		*(*uint64)(unsafe.Pointer((*[8]byte)(b))) = v
	}
}
