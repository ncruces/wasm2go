package libc

import (
	"bytes"
	"strconv"
	"unsafe"
)

func strtod(s, endptr ptr) float64 {
	return strtod_helper(s, endptr, 64)
}

func strtof(s, endptr ptr) float32 {
	return float32(strtod_helper(s, endptr, 32))
}

func strtoll(s, endptr ptr, base int32) int64 {
	return strtoll_helper(s, endptr, base, 64)
}

func strtoull(s, endptr ptr, base int32) uint64 {
	return strtoull_helper(s, endptr, base, 64)
}

func strtol(s, endptr ptr, base int32) int32 {
	return int32(strtoll_helper(s, endptr, base, 32))
}

func strtoul(s, endptr ptr, base int32) uint32 {
	return uint32(strtoull_helper(s, endptr, base, 32))
}

func strtod_helper(s, endptr ptr, bitSize int) float64 {
	m0 := memory[uptr(s):]
	m1 := bytes.TrimLeft(m0, " \t\n\v\f\r")
	m2 := bytes.TrimLeft(m1, "+-.0123456789abcdefinptxyABCDEFINPTXY")
	prefix := len(m0) - len(m1)
	digits := len(m1) - len(m2)

	var val float64
	for ; digits > 0; digits-- {
		var err error
		str := unsafe.String(&m1[0], digits)
		val, err = strconv.ParseFloat(str, bitSize)
		if e, ok := err.(*strconv.NumError); !ok || e.Err == strconv.ErrRange {
			break
		}
	}

	if endptr != 0 {
		if digits > 0 {
			s += ptr(prefix + digits)
		}
		store32(memory[uptr(endptr):], uint32(s))
	}
	return val
}

func strtoll_helper(s, endptr ptr, base int32, bitSize int) int64 {
	m0 := memory[uptr(s):]
	m1 := bytes.TrimLeft(m0, " \t\n\v\f\r")
	m2 := bytes.TrimLeft(m1, "+-0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	prefix := len(m0) - len(m1)
	digits := len(m1) - len(m2)

	var val int64
	for ; digits > 0; digits-- {
		var err error
		str := unsafe.String(&m1[0], digits)
		val, err = strconv.ParseInt(str, int(base), bitSize)
		if e, ok := err.(*strconv.NumError); !ok || e.Err == strconv.ErrRange {
			break
		}
	}

	if endptr != 0 {
		if digits > 0 {
			s += ptr(prefix + digits)
		}
		store32(memory[uptr(endptr):], uint32(s))
	}
	return val
}

func strtoull_helper(s, endptr ptr, base int32, bitSize int) uint64 {
	m0 := memory[uptr(s):]
	m1 := bytes.TrimLeft(m0, " \t\n\v\f\r")

	var neg bool
	switch {
	case bytes.HasPrefix(m1, []byte("-")):
		neg = true
		fallthrough
	case bytes.HasPrefix(m1, []byte("+")):
		m1 = m1[1:]
	}

	m2 := bytes.TrimLeft(m1, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	prefix := len(m0) - len(m1)
	digits := len(m1) - len(m2)

	var val uint64
	for ; digits > 0; digits-- {
		var err error
		str := unsafe.String(&m1[0], digits)
		val, err = strconv.ParseUint(str, int(base), bitSize)
		if e, ok := err.(*strconv.NumError); !ok || e.Err == strconv.ErrRange {
			break
		}
	}

	if neg {
		if bitSize == 32 {
			val = uint64(-uint32(val))
		} else {
			val = -val
		}
	}

	if endptr != 0 {
		if digits > 0 {
			s += ptr(prefix + digits)
		}
		store32(memory[uptr(endptr):], uint32(s))
	}
	return val
}
