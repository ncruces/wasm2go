// A simple bump allocator.
// Takes over the initial heap, then grows it as needed.
// Assumes that new memory is zero-initialized,
// and that the heap base is aligned to max_align_t.
// It allocates aligned chunks and keeps no size metadata.
//
// In a default build, the last allocation can be freed or resized in place.
// With BUMP_NO_FREE, returned memory is never reclaimed.

#include <stdalign.h>
#include <stdint.h>
#include <stdlib.h>

#define PAGESIZE 65536
#define ALIGN_SIZE alignof(max_align_t)

extern char __heap_base[];
extern char __heap_end[];

static char* bump_free_beg = __heap_base;
static char* bump_free_end = __heap_end;
static char* bump_high_ptr = __heap_base;
static char* bump_last_ptr;

void free(void* ptr) {
#ifndef BUMP_NO_FREE
  // Reclaim space if this was the last allocation.
  if (ptr != NULL && ptr == bump_last_ptr) {
    bump_free_beg = ptr;
    bump_last_ptr = NULL;
  }
#endif
}

size_t malloc_good_size(size_t size) {
  if (size == 0 || size > PTRDIFF_MAX) return size;
  return __builtin_align_up(size, ALIGN_SIZE);
}

void* malloc(size_t size) {
  if (size == 0 || size > PTRDIFF_MAX) return NULL;
  size = __builtin_align_up(size, ALIGN_SIZE);

  for (;;) {
    // Do we have enough free space?
    size_t avail = bump_free_end - bump_free_beg;
    if (size <= avail) break;
    // Grow the linear memory.
    size_t npages = __builtin_align_up(size - avail, PAGESIZE) / PAGESIZE;
    size_t old = __builtin_wasm_memory_grow(0, npages);
    if (old == SIZE_MAX) return NULL;
    // Did we grow our current arena?
    if (old * PAGESIZE == (size_t)bump_free_end) {
      bump_free_end += npages * PAGESIZE;
      break;
    }
    // Memory was grown elsewhere, this is a new arena.
    bump_free_beg = (char*)(old * PAGESIZE);
    bump_free_end = (char*)((old + npages) * PAGESIZE);
    bump_last_ptr = NULL;  // Can't rewind into the old arena.
  }

  void* res = bump_free_beg;
  bump_free_beg += size;
  bump_last_ptr = res;
#ifndef BUMP_NO_FREE
  if (bump_free_beg > bump_high_ptr) bump_high_ptr = bump_free_beg;
#endif
  return res;
}

void* memalign(size_t align, size_t size) {
  if (size == 0 || size > PTRDIFF_MAX) return NULL;
  if (__builtin_popcountg(align) != 1) return NULL;
  if (align <= ALIGN_SIZE) return malloc(size);

  size_t need;
  if (__builtin_add_overflow(size, align - ALIGN_SIZE, &need)) return NULL;

  char* res = malloc(need);
  if (res != NULL) {
    // Align the pointer up.
    res = __builtin_align_up(res, align);
    // Return excess memory.
    bump_free_beg = __builtin_align_up(res + size, ALIGN_SIZE);
    bump_last_ptr = res;
  }
  return res;
}

void* realloc(void* ptr, size_t size) {
  if (ptr == NULL) return malloc(size);
  if (size == 0) {
    free(ptr);
    return NULL;
  }
  // No need to move the first chunk.
  if (size <= ALIGN_SIZE && ptr != bump_last_ptr) return ptr;

  char* free_beg = bump_free_beg;
  // Worst case size of existing object.
  size_t copy = free_beg - (char*)ptr;
  if (ptr == bump_last_ptr) {
#ifdef BUMP_NO_FREE
    if (size <= copy) return ptr;  // Never shrink.
#endif
    bump_free_beg = (char*)ptr;  // Rewind to grow/shrink in place.
  }

  void* res = malloc(size);
  if (res == NULL) {  // Allocation failed.
    // Undo the rewind, unless malloc moved to a new arena.
    if (bump_free_beg == (char*)ptr) bump_free_beg = free_beg;
    return NULL;
  }
  if (res != ptr) {  // Allocation moved.
    if (copy > size) copy = size;
    __builtin_memcpy(res, ptr, copy);
  }
  return res;
}

void* calloc(size_t nelem, size_t elsize) {
  size_t need;
  if (__builtin_mul_overflow(nelem, elsize, &need)) return NULL;
#ifndef BUMP_NO_FREE
  char* high = bump_high_ptr;
#endif
  char* res = malloc(need);
#ifndef BUMP_NO_FREE
  // Clear memory that might've been reclaimed.
  if (res != NULL && res < high) {
    size_t dirty = high - res;
    __builtin_memset(res, 0, dirty < need ? dirty : need);
  }
#endif
  return res;
}

void* aligned_alloc(size_t align, size_t size) {
  if (__builtin_popcountg(align) != 1) return NULL;
  if (!__builtin_is_aligned(size, align)) return NULL;
  return memalign(align, size);
}

void* reallocarray(void* ptr, size_t nelem, size_t elsize) {
  size_t need;
  if (__builtin_mul_overflow(nelem, elsize, &need)) return NULL;
  return realloc(ptr, need);
}
