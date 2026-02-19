package helpers

import "math"

func i64_trunc_sat_f64_s(f float64) int64 {
	f = math.Trunc(f)
	switch {
	case f < math.MinInt64:
		return math.MinInt64
	case f >= math.MaxInt64:
		return math.MaxInt64
	case math.IsNaN(f):
		return 0
	}
	return int64(f)
}
