package helpers

import (
	"math"
	"math/bits"
)

func i32_div_s(x, y int32) int32 {
	if x == math.MinInt32 && y == -1 {
		panic("integer overflow")
	}
	return x / y
}

func i64_div_s(x, y int64) int64 {
	if x == math.MinInt64 && y == -1 {
		panic("integer overflow")
	}
	return x / y
}

func i32_shl(x, y int32) int32 {
	return x << (y & 31)
}

func i32_shr_s(x, y int32) int32 {
	return x >> (y & 31)
}

func i32_shr_u(x, y int32) int32 {
	return int32(uint32(x) >> (y & 31))
}

func i64_shl(x, y int64) int64 {
	return x << (y & 63)
}

func i64_shr_s(x, y int64) int64 {
	return x >> (y & 63)
}

func i64_shr_u(x, y int64) int64 {
	return int64(uint64(x) >> (y & 63))
}

func i32_rotl(x, y int32) int32 {
	return int32(bits.RotateLeft32(uint32(x), +int(y)&31))
}

func i32_rotr(x, y int32) int32 {
	return int32(bits.RotateLeft32(uint32(x), -int(y)&31))
}

func i64_rotl(x, y int64) int64 {
	return int64(bits.RotateLeft64(uint64(x), +int(y)&63))
}

func i64_rotr(x, y int64) int64 {
	return int64(bits.RotateLeft64(uint64(x), -int(y)&63))
}

// go.dev/issues/76264 can speed these up

func i32_trunc_f64_s(f float64) int32 {
	x := math.Trunc(f)
	if math.IsNaN(x) ||
		x < math.MinInt32 ||
		x > math.MaxInt32 {
		panic("invalid conversion to integer")
	}
	return int32(x)
}

func i32_trunc_f32_s(f float32) int32 {
	x := math.Trunc(float64(f))
	if math.IsNaN(x) ||
		x < math.MinInt32 ||
		x > math.MaxInt32 {
		panic("invalid conversion to integer")
	}
	return int32(x)
}

func i32_trunc_f64_u(f float64) int32 {
	x := math.Trunc(f)
	if math.IsNaN(x) ||
		x < 0 ||
		x > math.MaxUint32 {
		panic("invalid conversion to integer")
	}
	return int32(uint32(x))
}

func i32_trunc_f32_u(f float32) int32 {
	x := math.Trunc(float64(f))
	if math.IsNaN(x) ||
		x < 0 ||
		x > math.MaxUint32 {
		panic("invalid conversion to integer")
	}
	return int32(uint32(x))
}

func i64_trunc_f64_s(f float64) int64 {
	x := math.Trunc(f)
	if math.IsNaN(x) ||
		x < math.MinInt64 ||
		x >= math.MaxInt64 {
		panic("invalid conversion to integer")
	}
	return int64(x)
}

func i64_trunc_f32_s(f float32) int64 {
	x := math.Trunc(float64(f))
	if math.IsNaN(x) ||
		x < math.MinInt64 ||
		x >= math.MaxInt64 {
		panic("invalid conversion to integer")
	}
	return int64(x)
}

func i64_trunc_f64_u(f float64) int64 {
	x := math.Trunc(f)
	if math.IsNaN(x) ||
		x < 0 ||
		x >= math.MaxUint64 {
		panic("invalid conversion to integer")
	}
	return int64(uint64(x))
}

func i64_trunc_f32_u(f float32) int64 {
	x := math.Trunc(float64(f))
	if math.IsNaN(x) || x < 0 ||
		x >= math.MaxUint64 {
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
		//
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
		//
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
		//
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
		//
	case x >= math.MaxUint64:
		i = math.MaxUint64
	default:
		i = uint64(x)
	}
	return int64(i)
}
