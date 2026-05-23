#include <unistd.h>

#define PAGESIZE 65536

void* sbrk(intptr_t increment) {
  if (increment == 0) return (void*)(__builtin_wasm_memory_size(0) * PAGESIZE);
  if (increment < 0 || increment % PAGESIZE != 0) abort();

  size_t old = __builtin_wasm_memory_grow(0, (size_t)increment / PAGESIZE);
  if (old == SIZE_MAX) return (void*)old;
  return (void*)(old * PAGESIZE);
}
