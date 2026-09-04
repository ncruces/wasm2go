#include <strings.h>

inline int(ffs)(int x) { return __builtin_ffs(x); }

inline void(bcopy)(const void* s1, void* s2, size_t n) {
  __builtin_memmove(s2, s1, n);
}

inline void(bzero)(void* s, size_t n) { __builtin_memset(s, 0, n); }
