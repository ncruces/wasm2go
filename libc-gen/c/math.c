#include <math.h>

__attribute__((always_inline)) double(sqrt)(double x) {
  return __builtin_sqrt(x);
}

__attribute__((always_inline)) double(fabs)(double x) {
  return __builtin_fabs(x);
}

__attribute__((always_inline)) double(ceil)(double x) {
  return __builtin_ceil(x);
}

__attribute__((always_inline)) double(floor)(double x) {
  return __builtin_floor(x);
}

__attribute__((always_inline)) double(trunc)(double x) {
  return __builtin_trunc(x);
}

__attribute__((always_inline)) double(roundeven)(double x) {
  return __builtin_roundeven(x);
}

__attribute__((always_inline)) double(copysign)(double x, double y) {
  return __builtin_copysign(x, y);
}

__attribute__((always_inline)) double(rint)(double x) {
  return __builtin_rint(x);
}

__attribute__((always_inline)) long(lrint)(double x) {
  return (long)__builtin_rint(x);
}

__attribute__((always_inline)) long long(llrint)(double x) {
  return (long long)__builtin_rint(x);
}

__attribute__((always_inline)) float(sqrtf)(float x) {
  return __builtin_sqrtf(x);
}

__attribute__((always_inline)) float(fabsf)(float x) {
  return __builtin_fabsf(x);
}

__attribute__((always_inline)) float(ceilf)(float x) {
  return __builtin_ceilf(x);
}

__attribute__((always_inline)) float(floorf)(float x) {
  return __builtin_floorf(x);
}

__attribute__((always_inline)) float(truncf)(float x) {
  return __builtin_truncf(x);
}

__attribute__((always_inline)) float(roundevenf)(float x) {
  return __builtin_roundevenf(x);
}

__attribute__((always_inline)) float(copysignf)(float x, float y) {
  return __builtin_copysignf(x, y);
}

__attribute__((always_inline)) float(rintf)(float x) {
  return __builtin_rintf(x);
}

__attribute__((always_inline)) long(lrintf)(float x) {
  return (long)__builtin_rintf(x);
}

__attribute__((always_inline)) long long(llrintf)(float x) {
  return (long long)__builtin_rintf(x);
}
