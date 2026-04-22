#include <math.h>

__attribute__((always_inline)) double(ceil)(double x) {
  return __builtin_ceil(x);
}

__attribute__((always_inline)) double(floor)(double x) {
  return __builtin_floor(x);
}

__attribute__((always_inline)) double(rint)(double x) {
  return __builtin_rint(x);
}

__attribute__((always_inline)) double(roundeven)(double x) {
  return __builtin_roundeven(x);
}

__attribute__((always_inline)) double(sqrt)(double x) {
  return __builtin_sqrt(x);
}

__attribute__((always_inline)) double(trunc)(double x) {
  return __builtin_trunc(x);
}
