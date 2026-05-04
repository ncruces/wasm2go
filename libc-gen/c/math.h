#pragma once

typedef float float_t;
typedef double double_t;

double acos(double);
double acosh(double);
double asin(double);
double asinh(double);
double atan(double);
double atan2(double, double);
double atanh(double);
double cbrt(double);
double ceil(double);
double copysign(double, double);
double cos(double);
double cosh(double);
double erf(double);
double erfc(double);
double exp(double);
double exp2(double);
double expm1(double);
double fabs(double);
double fdim(double, double);
double floor(double);
double fma(double, double, double);
double fmax(double, double);
double fmin(double, double);
double fmod(double, double);
double frexp(double, int*);
double hypot(double, double);
double j0(double);
double j1(double);
double jn(int, double);
double ldexp(double, int);
double lgamma_r(double, int*);
double lgamma(double);
double log(double);
double log10(double);
double log1p(double);
double log2(double);
double logb(double);
double modf(double, double*);
double nextafter(double, double);
double pow(double, double);
double remainder(double, double);
double rint(double);
double round(double);
double roundeven(double);
double sin(double);
double sinh(double);
double sqrt(double);
double tan(double);
double tanh(double);
double tgamma(double);
double trunc(double);
double y0(double);
double y1(double);
double yn(int, double);
int ilogb(double);
long lrint(double);
long long llrint(double);

#define fabs(x) (__builtin_fabs(x))
#define ceil(x) (__builtin_ceil(x))
#define floor(x) (__builtin_floor(x))
#define trunc(x) (__builtin_trunc(x))
#define roundeven(x) (__builtin_roundeven(x))
#define sqrt(x) (__builtin_sqrt(x))
#define copysign(x, y) (__builtin_copysign(x, y))
#define rint(x) (__builtin_rint(x))
#define lrint(x) ((long)__builtin_rint(x))
#define llrint(x) ((long long)__builtin_rint(x))

#define NAN (__builtin_nanf(""))
#define INFINITY (__builtin_inff())
#define HUGE_VAL (__builtin_huge_val())
#define HUGE_VALF (__builtin_huge_valf())

#define FP_FAST_FMA 1

#define FP_NAN 0
#define FP_INFINITE 1
#define FP_ZERO 2
#define FP_SUBNORMAL 3
#define FP_NORMAL 4

#define fpclassify(x) \
  (__builtin_fpclassify(FP_NAN, FP_INFINITE, FP_NORMAL, FP_SUBNORMAL, FP_ZERO, x))

#define isfinite(x) (__builtin_isfinite(x))
#define isinf(x) (__builtin_isinf(x))
#define isnan(x) (__builtin_isnan(x))
#define isnormal(x) (__builtin_isnormal(x))
#define signbit(x) (__builtin_signbit(x))
#define isgreater(x, y) (__builtin_isgreater(x, y))
#define isgreaterequal(x, y) (__builtin_isgreaterequal(x, y))
#define isless(x, y) (__builtin_isless(x, y))
#define islessequal(x, y) (__builtin_islessequal(x, y))
#define islessgreater(x, y) (__builtin_islessgreater(x, y))
#define isunordered(x, y) (__builtin_isunordered(x, y))

#define M_E 2.71828182845904523536028747135266249775724709369995957496696763
#define M_PI 3.14159265358979323846264338327950288419716939937510582097494459
#define M_LN2 0.693147180559945309417232121458176568075500134360255254120680009
#define M_LN10 2.30258509299404568401799145468436420760110148862877297603332790
#define M_SQRT2 1.41421356237309504880168872420969807856967187537694807317667974

#define M_PI_2 1.5707963267948966192313216916397514
#define M_PI_4 0.7853981633974483096156608458198757
#define M_1_PI 0.3183098861837906715377675267450287
#define M_2_PI 0.6366197723675813430755350534900574
#define M_2_SQRTPI 1.1283791670955125738961589031215452

#define M_LOG2E 1.4426950408889634073599246810018921
#define M_LOG10E 0.4342944819032518276511289189166051
#define M_SQRT1_2 0.7071067811865475244008443621048490
