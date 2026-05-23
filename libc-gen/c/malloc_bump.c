// A simple bump allocator that never frees memory.
// Takes over the initial heap, then grows it as needed.
// Assumes that new memory is zero-initialized,
// and that the heap base is 16 byte aligned.
// It allocates in 16 byte chunks and keeps no size metadata.

#include <stdint.h>
#include <stdlib.h>

#define CHUNKSIZE 16
#define PAGESIZE 65536

extern char __heap_base[];
extern char __heap_end[];

static char* __arena_beg = __heap_base;
static char* __arena_end = __heap_end;

__attribute__((always_inline)) void free(void*) {}

void* malloc(size_t size) {
  if (size == 0) return NULL;
  size = __builtin_align_up(size, CHUNKSIZE);

  for (;;) {
    // Does the arena have enough free space?
    size_t avail = __arena_end - __arena_beg;
    if (size <= avail) break;
    // Grow the linear memory.
    size_t npages = (size - avail + PAGESIZE - 1) / PAGESIZE;
    size_t old = __builtin_wasm_memory_grow(0, npages);
    if (old == SIZE_MAX) return NULL;
    // Did we grow our current arena?
    if (old * PAGESIZE == (size_t)__arena_end) {
      __arena_end += npages * PAGESIZE;
      break;
    }
    // Memory was grown elsewhere, this is a new arena.
    __arena_beg = (char*)((old) * PAGESIZE);
    __arena_end = (char*)((old + npages) * PAGESIZE);
  }

  void* res = __arena_beg;
  __arena_beg += size;
  return res;
}

void* calloc(size_t nelem, size_t elsize) {
  size_t need;
  if (__builtin_mul_overflow(nelem, elsize, &need)) return NULL;
  // Assumes new memory is zero-initialized.
  return malloc(need);
}

void* aligned_alloc(size_t align, size_t size) {
  // Ensure non-zero power-of-two.
  if (align <= 0 || (align & (align - 1))) return NULL;
  size_t need;
  if (__builtin_add_overflow(size, align - 1, &need)) return NULL;

  char* res = malloc(need);
  if (res != NULL) {
    // Align the pointer up.
    res = __builtin_align_up(res, align);
    // Return excess memory.
    __arena_beg = __builtin_align_up(res + size, CHUNKSIZE);
  }
  return res;
}

void* realloc(void* ptr, size_t size) {
  if (ptr == NULL) return malloc(size);
  // No need to move the first chunk.
  if (size <= CHUNKSIZE) return ptr;
  // Worst case size of existing object.
  size_t copy = __arena_beg - (char*)ptr;
  // Rewind the last chunk.
  if (copy == CHUNKSIZE) __arena_beg = (char*)ptr;

  void* res = malloc(size);
  if (res != NULL && res != ptr) {
    if (copy > size) copy = size;
    __builtin_memcpy(res, ptr, copy);
  }
  return res;
}
