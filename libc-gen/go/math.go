package libc

import "math"

func acos(x float64) float64           { return math.Acos(x) }
func acosh(x float64) float64          { return math.Acosh(x) }
func asin(x float64) float64           { return math.Asin(x) }
func asinh(x float64) float64          { return math.Asinh(x) }
func atan(x float64) float64           { return math.Atan(x) }
func atan2(y, x float64) float64       { return math.Atan2(y, x) }
func atanh(x float64) float64          { return math.Atanh(x) }
func cbrt(x float64) float64           { return math.Cbrt(x) }
func copysign(x, y float64) float64    { return math.Copysign(x, y) }
func cos(x float64) float64            { return math.Cos(x) }
func cosh(x float64) float64           { return math.Cosh(x) }
func erf(x float64) float64            { return math.Erf(x) }
func erfc(x float64) float64           { return math.Erfc(x) }
func exp(x float64) float64            { return math.Exp(x) }
func exp2(x float64) float64           { return math.Exp2(x) }
func expm1(x float64) float64          { return math.Expm1(x) }
func fdim(x, y float64) float64        { return math.Dim(x, y) }
func fma(x, y, z float64) float64      { return math.FMA(x, y, z) }
func fmod(x, y float64) float64        { return math.Mod(x, y) }
func hypot(x, y float64) float64       { return math.Hypot(x, y) }
func j0(x float64) float64             { return math.J0(x) }
func j1(x float64) float64             { return math.J1(x) }
func jn(n int32, x float64) float64    { return math.Jn(int(n), x) }
func ldexp(x float64, n int32) float64 { return math.Ldexp(x, int(n)) }
func log(x float64) float64            { return math.Log(x) }
func log10(x float64) float64          { return math.Log10(x) }
func log1p(x float64) float64          { return math.Log1p(x) }
func log2(x float64) float64           { return math.Log2(x) }
func logb(x float64) float64           { return math.Logb(x) }
func nextafter(x, y float64) float64   { return math.Nextafter(x, y) }
func pow(x, y float64) float64         { return math.Pow(x, y) }
func remainder(x, y float64) float64   { return math.Remainder(x, y) }
func round(x float64) float64          { return math.Round(x) }
func sin(x float64) float64            { return math.Sin(x) }
func sinh(x float64) float64           { return math.Sinh(x) }
func sqrt(x float64) float64           { return math.Sqrt(x) }
func tan(x float64) float64            { return math.Tan(x) }
func tanh(x float64) float64           { return math.Tanh(x) }
func tgamma(x float64) float64         { return math.Gamma(x) }
func y0(x float64) float64             { return math.Y0(x) }
func y1(x float64) float64             { return math.Y1(x) }
func yn(n int32, x float64) float64    { return math.Yn(int(n), x) }
func ilogb(x float64) int32            { return int32(math.Ilogb(x)) }

func lgamma(x float64) float64 {
	x, _ = math.Lgamma(x)
	return x
}

func lgamma_r(x float64, sptr ptr) float64 {
	x, sign := math.Lgamma(x)
	store32(memory[uptr(sptr):], uint32(sign))
	return x
}

func frexp(x float64, eptr ptr) float64 {
	x, exp := math.Frexp(x)
	store32(memory[uptr(eptr):], uint32(exp))
	return x
}

func modf(x float64, iptr ptr) (f float64) {
	if math.IsInf(x, 0) {
		f = math.Copysign(0, x)
	} else {
		x, f = math.Modf(x)
	}
	store64(memory[uptr(iptr):], math.Float64bits(x))
	return f
}

func fmax(x, y float64) float64 {
	switch r := max(x, y); {
	case !math.IsNaN(r):
		return r
	case math.IsNaN(x):
		return y
	default:
		return x
	}
}

func fmin(x, y float64) float64 {
	switch r := min(x, y); {
	case !math.IsNaN(r):
		return r
	case math.IsNaN(x):
		return y
	default:
		return x
	}
}
