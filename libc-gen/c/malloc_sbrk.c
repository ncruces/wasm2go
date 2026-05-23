// Configures and includes Doug Lea's malloc
// for use with Wasm.
// Optionally, call init_allocator to have
// malloc take over the initial heap.
// Grows memory in increments of 64K,
// and expects it to be contiguous,
// but can cope with non-contiguities.

#include <stdint.h>
#include <stdlib.h>

#define PAGESIZE 65536

#define LACKS_FCNTL_H
#define LACKS_SCHED_H
#define LACKS_SYS_MMAN_H
#define LACKS_SYS_PARAM_H
#define LACKS_TIME_H // prefer determinism

#define HAVE_MMAP 0
#define MALLOC_ALIGNMENT 16
#define MALLOC_FAILURE_ACTION
#define MORECORE_CANNOT_TRIM 1
#define NO_MALLINFO 1
#define NO_MALLOC_STATS 1
#define USE_BUILTIN_FFS 1
#define USE_LOCKS 0

#pragma clang diagnostic ignored "-Weverything"

#include "malloc.c"

extern char __heap_base[];
extern char __heap_end[];

// Initialize dlmalloc to be able to use the memory between
// __heap_base and __heap_end.
static void init_allocator(void) {
  if (is_initialized(gm)) __builtin_trap();
  ensure_initialization();

  size_t heap_size = __heap_end - __heap_base;
  if (heap_size <= MIN_CHUNK_SIZE + TOP_FOOT_SIZE + MALLOC_ALIGNMENT) return;

  gm->least_addr = __heap_base;
  gm->seg.base = __heap_base;
  gm->seg.size = heap_size;
  gm->footprint = heap_size;
  gm->max_footprint = heap_size;
  gm->magic = mparams.magic;
  gm->release_checks = MAX_RELEASE_CHECK_RATE;

  init_bins(gm);
  init_top(gm, (mchunkptr)__heap_base, heap_size - TOP_FOOT_SIZE);
}

__attribute__((alias("memalign")))
void* aligned_alloc(size_t align, size_t size);
