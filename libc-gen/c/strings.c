#include <strings.h>

__attribute__((always_inline)) int(ffs)(int x) { return __builtin_ffs(x); }

__attribute__((always_inline)) void(bcopy)(const void* s1, void* s2, size_t n) {
  __builtin_memmove(s2, s1, n);
}

__attribute__((always_inline)) void(bzero)(void* s, size_t n) {
  __builtin_memset(s, 0, n);
}
