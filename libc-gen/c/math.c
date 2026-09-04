#include <math.h>

inline double(sqrt)(double x) { return __builtin_sqrt(x); }

inline double(fabs)(double x) { return __builtin_fabs(x); }

inline double(ceil)(double x) { return __builtin_ceil(x); }

inline double(floor)(double x) { return __builtin_floor(x); }

inline double(trunc)(double x) { return __builtin_trunc(x); }

inline double(roundeven)(double x) { return __builtin_roundeven(x); }

inline double(copysign)(double x, double y) { return __builtin_copysign(x, y); }

inline double(rint)(double x) { return __builtin_rint(x); }

inline long(lrint)(double x) { return (long)__builtin_rint(x); }

inline long long(llrint)(double x) { return (long long)__builtin_rint(x); }

inline float(sqrtf)(float x) { return __builtin_sqrtf(x); }

inline float(fabsf)(float x) { return __builtin_fabsf(x); }

inline float(ceilf)(float x) { return __builtin_ceilf(x); }

inline float(floorf)(float x) { return __builtin_floorf(x); }

inline float(truncf)(float x) { return __builtin_truncf(x); }

inline float(roundevenf)(float x) { return __builtin_roundevenf(x); }

inline float(copysignf)(float x, float y) { return __builtin_copysignf(x, y); }

inline float(rintf)(float x) { return __builtin_rintf(x); }

inline long(lrintf)(float x) { return (long)__builtin_rintf(x); }

inline long long(llrintf)(float x) { return (long long)__builtin_rintf(x); }
