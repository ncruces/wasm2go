package libc

import (
	"bytes"
	"strconv"
	"unsafe"
)

func strtod(s, endptr ptr) float64 {
	m0 := memory[uptr(s):]
	m1 := bytes.TrimLeft(m0, " \t\n\v\f\r")
	m2 := bytes.TrimLeft(m1, "+-.0123456789abcdefinptxyABCDEFINPTXY")
	spaces := len(m0) - len(m1)
	digits := len(m1) - len(m2)

	var val float64
	for ; digits > 0; digits-- {
		var err error
		str := unsafe.String(&m1[0], digits)
		val, err = strconv.ParseFloat(str, 64)
		if e, ok := err.(*strconv.NumError); !ok || e.Err == strconv.ErrRange {
			break
		}
	}

	if endptr != 0 {
		if digits > 0 {
			s += ptr(spaces + digits)
		}
		store32(memory[uptr(endptr):], uint32(s))
	}
	return val
}

func strtol(s, endptr ptr, base int32) int32 {
	m0 := memory[uptr(s):]
	m1 := bytes.TrimLeft(m0, " \t\n\v\f\r")
	m2 := bytes.TrimLeft(m1, "+-0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	spaces := len(m0) - len(m1)
	digits := len(m1) - len(m2)

	var val int64
	for ; digits > 0; digits-- {
		var err error
		str := unsafe.String(&m1[0], digits)
		val, err = strconv.ParseInt(str, int(base), 32)
		if e, ok := err.(*strconv.NumError); !ok || e.Err == strconv.ErrRange {
			break
		}
	}

	if endptr != 0 {
		if digits > 0 {
			s += ptr(spaces + digits)
		}
		store32(memory[uptr(endptr):], uint32(s))
	}
	return int32(val)
}
