package helpers

import (
	"math"
	"math/bits"
	"runtime"
)

// These prevent constant folding/propagation,
// ensuring code using them compiles
// and overflows/panics at runtime.

//go:nosplit
func i32(x int32) int32 { return x }

//go:nosplit
func i64(x int64) int64 { return x }

// Memory access byte swapping.

//go:nosplit
func swap16(x uint16) uint16 {
	switch runtime.GOARCH {
	case "386", "amd64", "amd64p32", "alpha", "arm", "arm64", "loong64", "mipsle", "mips64le", "mips64p32le", "nios2", "ppc64le", "riscv", "riscv64", "sh", "wasm":
		return x
	default:
		return bits.ReverseBytes16(x)
	}
}

//go:nosplit
func swap32(x uint32) uint32 {
	switch runtime.GOARCH {
	case "386", "amd64", "amd64p32", "alpha", "arm", "arm64", "loong64", "mipsle", "mips64le", "mips64p32le", "nios2", "ppc64le", "riscv", "riscv64", "sh", "wasm":
		return x
	default:
		return bits.ReverseBytes32(x)
	}
}

//go:nosplit
func swap64(x uint64) uint64 {
	switch runtime.GOARCH {
	case "386", "amd64", "amd64p32", "alpha", "arm", "arm64", "loong64", "mipsle", "mips64le", "mips64p32le", "nios2", "ppc64le", "riscv", "riscv64", "sh", "wasm":
		return x
	default:
		return bits.ReverseBytes64(x)
	}
}

// These prevent constant folding/propagation,
// ensuring correct NaN handling.

//go:nosplit
func f32(x float32) float32 {
	runtime.KeepAlive(&x)
	return x
}

//go:nosplit
func f64(x float64) float64 {
	runtime.KeepAlive(&x)
	return x
}

// Detect signed integer overflow.
// These generate sub-optimal code on amd64.

//go:nosplit
func i32_div_s(x, y int32) int32 {
	if y == -1 && x == math.MinInt32 {
		panic("integer overflow")
	}
	return x / y
}

//go:nosplit
func i64_div_s(x, y int64) int64 {
	if y == -1 && x == math.MinInt64 {
		panic("integer overflow")
	}
	return x / y
}

//go:nosplit
func i32_neg_s(x int32) int32 {
	if x == math.MinInt32 {
		panic("integer overflow")
	}
	return -x
}

//go:nosplit
func i64_neg_s(x int64) int64 {
	if x == math.MinInt64 {
		panic("integer overflow")
	}
	return -x
}

// These are needed for correct y wrap around behavior.

//go:nosplit
func i32_shl(x, y int32) int32 {
	return x << (y & 31)
}

//go:nosplit
func i32_shr_s(x, y int32) int32 {
	return x >> (y & 31)
}

//go:nosplit
func i32_shr_u(x, y int32) int32 {
	return int32(uint32(x) >> (y & 31))
}

//go:nosplit
func i64_shl(x, y int64) int64 {
	return x << (y & 63)
}

//go:nosplit
func i64_shr_s(x, y int64) int64 {
	return x >> (y & 63)
}

//go:nosplit
func i64_shr_u(x, y int64) int64 {
	return int64(uint64(x) >> (y & 63))
}

//go:nosplit
func i32_rotl(x, y int32) int32 {
	return int32(bits.RotateLeft32(uint32(x), +int(y)&31))
}

//go:nosplit
func i32_rotr(x, y int32) int32 {
	return int32(bits.RotateLeft32(uint32(x), -int(y)&31))
}

//go:nosplit
func i64_rotl(x, y int64) int64 {
	return int64(bits.RotateLeft64(uint64(x), +int(y)&63))
}

//go:nosplit
func i64_rotr(x, y int64) int64 {
	return int64(bits.RotateLeft64(uint64(x), -int(y)&63))
}

// These must be implemented as bitwise operations,
// like the math versions are for float64.

//go:nosplit
func f32_abs(x float32) float32 {
	return math.Float32frombits(math.Float32bits(x) &^ (1 << 31))
}

//go:nosplit
func f32_copysign(x, y float32) float32 {
	return math.Float32frombits(math.Float32bits(x)&^(1<<31) | math.Float32bits(y)&(1<<31))
}

// These must return canonical NaNs,
// which they don't on amd64.

//go:nosplit
func f32_min(x, y float32) float32 {
	if m := min(x, y); m == m {
		return m
	}
	return math.Float32frombits(0x7fc00000)
}

//go:nosplit
func f32_max(x, y float32) float32 {
	if m := max(x, y); m == m {
		return m
	}
	return math.Float32frombits(0x7fc00000)
}

//go:nosplit
func f64_min(x, y float64) float64 {
	if m := min(x, y); m == m {
		return m
	}
	return math.Float64frombits(0x7ff8000000000000)
}

//go:nosplit
func f64_max(x, y float64) float64 {
	if m := max(x, y); m == m {
		return m
	}
	return math.Float64frombits(0x7ff8000000000000)
}

// Float to int conversions.

// All i64 conversions use >= because both MaxInt64 and MaxUint64
// round up when converted to a float64.

// go.dev/issues/76264 can speed these up

func i32_trunc_f64_s(f float64) int32 {
	x := math.Trunc(f)
	switch {
	case x < math.MinInt32 || x > math.MaxInt32:
		panic("integer overflow")
	case math.IsNaN(x):
		panic("invalid conversion to integer")
	}
	return int32(x)
}

func i32_trunc_f32_s(f float32) int32 {
	x := math.Trunc(float64(f))
	switch {
	case x < math.MinInt32 || x > math.MaxInt32:
		panic("integer overflow")
	case math.IsNaN(x):
		panic("invalid conversion to integer")
	}
	return int32(x)
}

func i32_trunc_f64_u(f float64) int32 {
	x := math.Trunc(f)
	switch {
	case x < 0 || x > math.MaxUint32:
		panic("integer overflow")
	case math.IsNaN(x):
		panic("invalid conversion to integer")
	}
	return int32(uint32(x))
}

func i32_trunc_f32_u(f float32) int32 {
	x := math.Trunc(float64(f))
	switch {
	case x < 0 || x > math.MaxUint32:
		panic("integer overflow")
	case math.IsNaN(x):
		panic("invalid conversion to integer")
	}
	return int32(uint32(x))
}

func i64_trunc_f64_s(f float64) int64 {
	x := math.Trunc(f)
	switch {
	case x < math.MinInt64 || x >= math.MaxInt64:
		panic("integer overflow")
	case math.IsNaN(x):
		panic("invalid conversion to integer")
	}
	return int64(x)
}

func i64_trunc_f32_s(f float32) int64 {
	x := math.Trunc(float64(f))
	switch {
	case x < math.MinInt64 || x >= math.MaxInt64:
		panic("integer overflow")
	case math.IsNaN(x):
		panic("invalid conversion to integer")
	}
	return int64(x)
}

func i64_trunc_f64_u(f float64) int64 {
	x := math.Trunc(f)
	switch {
	case x < 0 || x >= math.MaxUint64:
		panic("integer overflow")
	case math.IsNaN(x):
		panic("invalid conversion to integer")
	}
	return int64(uint64(x))
}

func i64_trunc_f32_u(f float32) int64 {
	x := math.Trunc(float64(f))
	switch {
	case x < 0 || x >= math.MaxUint64:
		panic("integer overflow")
	case math.IsNaN(x):
		panic("invalid conversion to integer")
	}
	return int64(uint64(x))
}

func i32_trunc_sat_f64_s(f float64) int32 {
	x := math.Trunc(f)
	switch {
	case x < math.MinInt32:
		return math.MinInt32
	case x > math.MaxInt32:
		return math.MaxInt32
	case math.IsNaN(x):
		return 0
	}
	return int32(x)
}

func i32_trunc_sat_f32_s(f float32) int32 {
	x := math.Trunc(float64(f))
	switch {
	case x < math.MinInt32:
		return math.MinInt32
	case x > math.MaxInt32:
		return math.MaxInt32
	case math.IsNaN(x):
		return 0
	}
	return int32(x)
}

func i32_trunc_sat_f64_u(f float64) int32 {
	x := math.Trunc(f)
	var i uint32
	switch {
	case x < 0 || math.IsNaN(x):
		i = 0
	case x > math.MaxUint32:
		i = math.MaxUint32
	default:
		i = uint32(x)
	}
	return int32(i)
}

func i32_trunc_sat_f32_u(f float32) int32 {
	x := math.Trunc(float64(f))
	var i uint32
	switch {
	case x < 0 || math.IsNaN(x):
		i = 0
	case x > math.MaxUint32:
		i = math.MaxUint32
	default:
		i = uint32(x)
	}
	return int32(i)
}

func i64_trunc_sat_f64_s(f float64) int64 {
	x := math.Trunc(f)
	switch {
	case x < math.MinInt64:
		return math.MinInt64
	case x >= math.MaxInt64:
		return math.MaxInt64
	case math.IsNaN(x):
		return 0
	}
	return int64(x)
}

func i64_trunc_sat_f32_s(f float32) int64 {
	x := math.Trunc(float64(f))
	switch {
	case x < math.MinInt64:
		return math.MinInt64
	case x >= math.MaxInt64:
		return math.MaxInt64
	case math.IsNaN(x):
		return 0
	}
	return int64(x)
}

func i64_trunc_sat_f64_u(f float64) int64 {
	x := math.Trunc(f)
	var i uint64
	switch {
	case x < 0 || math.IsNaN(x):
		i = 0
	case x >= math.MaxUint64:
		i = math.MaxUint64
	default:
		i = uint64(x)
	}
	return int64(i)
}

func i64_trunc_sat_f32_u(f float32) int64 {
	x := math.Trunc(float64(f))
	var i uint64
	switch {
	case x < 0 || math.IsNaN(x):
		i = 0
	case x >= math.MaxUint64:
		i = math.MaxUint64
	default:
		i = uint64(x)
	}
	return int64(i)
}

// Bulk memory operations.

func memory_grow(mem *[]byte, delta, max int64) int64 {
	buf := *mem
	len := len(buf)
	old := int64(len) >> 16
	if delta == 0 {
		return old
	}
	new := old + delta
	add := int(new)<<16 - len
	if new > max || add < 0 {
		return -1
	}
	*mem = append(buf, make([]byte, add)...)
	return old
}

func memory_init[T uint32 | uint64](mem []byte, data string, dest T, src, n uint32) {
	x := uint(min(uint64(dest), math.MaxUint))
	z := uint(src)
	y := x + uint(n)
	w := z + uint(n)
	copy(mem[x:y], data[z:w])
}

func memory_copy[T uint32 | uint64](mem []byte, dest, src, n T) {
	x := uint(min(uint64(dest), math.MaxUint))
	z := uint(min(uint64(src), math.MaxUint))
	c := uint(min(uint64(n), math.MaxUint))
	y := x + c
	w := z + c
	copy(mem[x:y], mem[z:w])
}

func memory_fill[T uint32 | uint64](mem []byte, dest T, val int32, n T) {
	x := uint(min(uint64(dest), math.MaxUint))
	y := x + uint(min(uint64(n), math.MaxUint))
	buf := mem[x:y]
	if len(buf) > 0 {
		buf[0] = byte(val)
		for i := 1; i < len(buf); {
			chunk := min(i, 8192)
			i += copy(buf[i:], buf[:chunk])
		}
	}
}

func memory_zero[T uint32 | uint64](mem []byte, dest, n T) {
	x := uint(min(uint64(dest), math.MaxUint))
	y := x + uint(min(uint64(n), math.MaxUint))
	clear(mem[x:y])
}

func table_init[T int32 | int64](tab, elems []any, dest, src, n T) {
	x := uint(dest)
	z := uint(src)
	y := x + uint(n)
	w := z + uint(n)
	copy(tab[x:y], elems[z:w])
}

func table_copy[T1, T2, T3 int32 | int64](dst, tab []any, dest T1, src T2, n T3) {
	x := uint(dest)
	z := uint(src)
	y := x + uint(n)
	w := z + uint(n)
	copy(dst[x:y], tab[z:w])
}

func table_grow[T int32 | int64](tab *[]any, val any, delta, max T) T {
	buf := *tab
	len := len(buf)
	old := T(len)
	if delta == 0 {
		return old
	}
	new := old + delta
	add := int(new) - len
	if new > max || add < 0 {
		return -1
	}
	buf = append(buf, make([]any, add)...)
	if val != nil {
		cpy := buf[len:]
		for i := range cpy {
			cpy[i] = val
		}
	}
	*tab = buf
	return old
}

func table_fill[T int32 | int64](tab []any, dest T, val any, n T) {
	x := uint(dest)
	y := x + uint(n)
	buf := tab[x:y]
	if val == nil {
		clear(buf)
		return
	}
	for i := range buf {
		buf[i] = val
	}
}
