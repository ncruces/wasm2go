// A Two-Level Segregated Fit (TLSF) memory allocator for Wasm.
// Takes over the initial heap, then grows it as needed.
// Provides O(1) allocation and deallocation with bounded fragmentation.
// Assumes a single-threaded, freestanding environment.
// Keeps block metadata in boundary tags and segregates free blocks.

#include <assert.h>
#include <limits.h>
#include <stdalign.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#define BITS_PER_WORD (sizeof(size_t) * CHAR_BIT)

#define PAGESIZE 65536

// Alignment constants.
#define ALIGN_SIZE alignof(max_align_t)
#define ALIGN_LOG2 __builtin_ctz(ALIGN_SIZE)

// Second-level subdivisions.
#define SL_INDEX_LOG2 4
#define SL_INDEX_COUNT (1U << SL_INDEX_LOG2)  // 16

// Smallest power-of-two range to subdivide.
#define FL_INDEX_SHIFT (SL_INDEX_LOG2 + ALIGN_LOG2)

// Total first-level indices needed to cover size_t address space.
#define FL_INDEX_COUNT (BITS_PER_WORD - FL_INDEX_SHIFT + 1)

// Types for first and second-level bimaps.
typedef size_t fl_bitmap_t;
typedef uint32_t sl_bitmap_t;

static_assert(FL_INDEX_COUNT <= sizeof(fl_bitmap_t) * CHAR_BIT,
              "fl_bitmap_t overflow");
static_assert(SL_INDEX_COUNT <= sizeof(sl_bitmap_t) * CHAR_BIT,
              "sl_bitmap_t overflow");

// Block header, precedes every block (allocated or free).
typedef struct block_header {
  size_t size;  // Tags on low bits.
} block_header_t;

#define BLOCK_BIT_FREE (1U << 0)
#define BLOCK_BIT_PREV_FREE (1U << 1)
#define BLOCK_TAG_MASK (ALIGN_SIZE - 1)

static_assert(ALIGN_LOG2 >= 2, "needed to store 2 tag bits");

// Intrusive list overlays the payload for free blocks.
typedef struct free_block {
  block_header_t header;
  // Payload starts here.
  struct free_block* next_free;
  struct free_block* prev_free;
  // Lots of empty space...
  size_t __min_size_footer
      __attribute__((deprecated("use block_get_footer()")));
} free_block_t;

// Minimum block size required to safely store free block metadata and footer.
#define BLOCK_SIZE_MIN __builtin_align_up(sizeof(free_block_t), ALIGN_SIZE)

// Minimum usable user payload (at the minimum block size).
#define BLOCK_PAYLOAD_MIN (BLOCK_SIZE_MIN - sizeof(block_header_t))

// Minumum usable pool size (enought to hold one minimally sized block and an
// empty sentinel block).
#define POOL_SIZE_MIN (BLOCK_SIZE_MIN + sizeof(block_header_t))

// Global TLSF state.
static struct {
  fl_bitmap_t fl_bitmap;
  sl_bitmap_t sl_bitmap[FL_INDEX_COUNT];
  free_block_t* blocks[FL_INDEX_COUNT][SL_INDEX_COUNT];
  char* heap_end;
} g_tlsf;

// The zero-based index of the highest set bit.
static inline int tlsf_fls(size_t size) {
  return (int)BITS_PER_WORD - __builtin_clzg(size, (int)BITS_PER_WORD) - 1;
}

// The zero-based index of the lowest set bit.
static inline int tlsf_ffs(size_t size) { return __builtin_ctzg(size, -1); }

static inline void* tlsf_block_to_payload(const block_header_t* block) {
  return (void*)((char*)block + sizeof(block_header_t));
}

static inline block_header_t* tlsf_payload_to_block(const void* ptr) {
  return (block_header_t*)((char*)ptr - sizeof(block_header_t));
}

static inline size_t tlsf_block_get_size(const block_header_t* block) {
  return block->size & ~BLOCK_TAG_MASK;
}

static inline void tlsf_block_set_size(block_header_t* block, size_t size) {
  block->size = size | (block->size & BLOCK_TAG_MASK);
}

static inline size_t* tlsf_block_get_footer(const block_header_t* block) {
  return (size_t*)((char*)block + tlsf_block_get_size(block) - sizeof(size_t));
}

static inline bool tlsf_block_is_free(const block_header_t* block) {
  return (block->size & BLOCK_BIT_FREE) != 0;
}

static inline void tlsf_block_set_free(block_header_t* block, bool free) {
  if (free) {
    *tlsf_block_get_footer(block) = tlsf_block_get_size(block);
    block->size |= BLOCK_BIT_FREE;
    return;
  }
  block->size &= ~BLOCK_BIT_FREE;
}

static inline bool tlsf_block_is_prev_free(const block_header_t* block) {
  return (block->size & BLOCK_BIT_PREV_FREE) != 0;
}

static inline void tlsf_block_set_prev_free(block_header_t* block, bool free) {
  if (free) {
    block->size |= BLOCK_BIT_PREV_FREE;
    return;
  }
  block->size &= ~BLOCK_BIT_PREV_FREE;
}

static inline block_header_t* tlsf_block_next_phys(
    const block_header_t* block) {
  return (block_header_t*)((char*)block + tlsf_block_get_size(block));
}

static inline block_header_t* tlsf_block_prev_phys(
    const block_header_t* block) {
  if (!tlsf_block_is_prev_free(block)) __builtin_trap();
  size_t prev_size = *(const size_t*)((const char*)block - sizeof(size_t));
  return (block_header_t*)((char*)block - prev_size);
}

static void tlsf_mapping(size_t size, bool insert, int* restrict fl,
                         int* restrict sl) {
  if (size < (1U << FL_INDEX_SHIFT)) {
    // No rounding needed: size is aligned.
    *fl = 0;
    *sl = (int)(size >> ALIGN_LOG2);
    return;
  }

  int t = tlsf_fls(size);
  int shift = t - SL_INDEX_LOG2;
  *fl = t - (FL_INDEX_SHIFT - 1);
  *sl = (int)((size >> shift) & (SL_INDEX_COUNT - 1));

  // We rounded down, that's what we need to insert a free block.
  if (insert) return;

  // To search for a free block, get the next size up,
  // if we needed to round down.
  if (size & ((1ULL << shift) - 1)) {
    *sl += 1;
    if (*sl == SL_INDEX_COUNT) {
      *sl = 0;
      *fl += 1;
    }
  }
}

static inline void tlsf_mapping_insert(size_t size, int* restrict fl,
                                       int* restrict sl) {
  tlsf_mapping(size, /*insert=*/true, fl, sl);
}

static inline void tlsf_mapping_search(size_t size, int* restrict fl,
                                       int* restrict sl) {
  tlsf_mapping(size, /*insert=*/false, fl, sl);
}

// Insert a free block into the free lists and update the bitmaps.
static void tlsf_insert_free_block(free_block_t* block) {
  int fl, sl;
  tlsf_mapping_insert(tlsf_block_get_size(&block->header), &fl, &sl);

  free_block_t* head = g_tlsf.blocks[fl][sl];
  block->next_free = head;
  block->prev_free = NULL;
  g_tlsf.blocks[fl][sl] = block;
  if (head != NULL) head->prev_free = block;

  // Add the new block to the bitmaps.
  g_tlsf.fl_bitmap |= (fl_bitmap_t)1 << fl;
  g_tlsf.sl_bitmap[fl] |= (sl_bitmap_t)1 << sl;
}

// Remove a free block from the free lists and update the bitmaps.
static void tlsf_remove_free_block(free_block_t* block) {
  int fl, sl;
  tlsf_mapping_insert(tlsf_block_get_size(&block->header), &fl, &sl);

  free_block_t* prev = block->prev_free;
  free_block_t* next = block->next_free;

  if (next != NULL) next->prev_free = prev;
  if (prev != NULL) {
    prev->next_free = next;
  } else {
    g_tlsf.blocks[fl][sl] = next;
  }

  // If the list is now empty, remove it from the bitmaps.
  if (next == NULL && prev == NULL) {
    g_tlsf.sl_bitmap[fl] &= ~((sl_bitmap_t)1 << sl);
    if (g_tlsf.sl_bitmap[fl] == 0) {
      g_tlsf.fl_bitmap &= ~((fl_bitmap_t)1 << fl);
    }
  }
}

static free_block_t* tlsf_find_free_block(size_t size) {
  int fl, sl;
  tlsf_mapping_search(size, &fl, &sl);

  sl_bitmap_t sl_map = g_tlsf.sl_bitmap[fl] & (~(sl_bitmap_t)0 << sl);
  if (sl_map == 0) {
    fl_bitmap_t fl_map = g_tlsf.fl_bitmap & (~(fl_bitmap_t)0 << (fl + 1));
    if (fl_map == 0) return NULL;  // Out of memory in existing pools.
    fl = tlsf_ffs(fl_map);
    sl_map = g_tlsf.sl_bitmap[fl];
  }

  sl = tlsf_ffs(sl_map);
  return g_tlsf.blocks[fl][sl];
}

// Split an allocated block into two allocated blocks.
// Return the second block, or NULL if the remaining space is too small.
static block_header_t* tlsf_block_split(block_header_t* block, size_t size) {
  size_t total_size = tlsf_block_get_size(block);
  size_t remaining_size = total_size - size;
  if (remaining_size < BLOCK_SIZE_MIN) {
    return NULL;
  }

  tlsf_block_set_size(block, size);
  block_header_t* remaining = tlsf_block_next_phys(block);

  // Initialize the new block.
  // Both blocks are allocated, so no tag bits are set.
  remaining->size = remaining_size;
  return remaining;
}

// Free an allocated block, coalesce it and update the free lists.
static void tlsf_block_free(block_header_t* block) {
  // Merge forward if next physical block is free.
  block_header_t* next = tlsf_block_next_phys(block);
  if (tlsf_block_is_free(next)) {
    tlsf_remove_free_block((free_block_t*)next);
    tlsf_block_set_size(block,
                        tlsf_block_get_size(block) + tlsf_block_get_size(next));
    next = tlsf_block_next_phys(block);
  }

  // Merge backward if previous physical block is free.
  if (tlsf_block_is_prev_free(block)) {
    block_header_t* prev = tlsf_block_prev_phys(block);
    tlsf_remove_free_block((free_block_t*)prev);
    tlsf_block_set_size(prev,
                        tlsf_block_get_size(prev) + tlsf_block_get_size(block));
    block = prev;
  }

  // Mark the consolidated block as free.
  tlsf_block_set_free(block, true);
  tlsf_block_set_prev_free(next, true);

  // Insert into the appropriate free list.
  tlsf_insert_free_block((free_block_t*)block);
}

static void tlsf_add_pool(char* pool_start, char* pool_end) {
  // Are we adding a contiguous pool?
  bool is_contiguous = pool_start == g_tlsf.heap_end;
  block_header_t* block;

  if (is_contiguous) {
    // We can use the previous sentinel block's header.
    block = (block_header_t*)(pool_start - sizeof(block_header_t));
  } else {
    // We need space for a header and to align our payload.
    char* payload_start =
        __builtin_align_up(pool_start + sizeof(block_header_t), ALIGN_SIZE);
    block = (block_header_t*)(payload_start - sizeof(block_header_t));
  }

  pool_end = __builtin_align_down(pool_end, ALIGN_SIZE);
  if (pool_end - (char*)block < (ptrdiff_t)POOL_SIZE_MIN) {
    return;  // Too small.
  }

  // Add a sentinel block to the end of the pool.
  block_header_t* sentinel =
      (block_header_t*)(pool_end - sizeof(block_header_t));
  sentinel->size = 0;

  // Initialize block, and add it to the free lists.
  if (!is_contiguous) block->size = 0;  // No previous block, clear tag bits.
  tlsf_block_set_size(block, (char*)sentinel - (char*)block);
  tlsf_block_free(block);

  g_tlsf.heap_end = pool_end;
}

// Takes a requested allocation size and rounds it up to
// an appropriate block size.
static inline size_t tlsf_request_to_block_size(size_t size) {
  size_t block_size =
      __builtin_align_up(sizeof(block_header_t) + size, ALIGN_SIZE);
  return block_size < BLOCK_SIZE_MIN ? BLOCK_SIZE_MIN : block_size;
}

void free(void* ptr) {
  if (ptr == NULL) return;
  tlsf_block_free(tlsf_payload_to_block(ptr));
}

void* malloc(size_t size) {
  if (size == 0 || size > PTRDIFF_MAX) return NULL;

  size_t block_size = tlsf_request_to_block_size(size);

  free_block_t* free_block = tlsf_find_free_block(block_size);
  if (free_block == NULL) {
    // Growth size needs to accommodate at least the block,
    // a sentinel header, and potential alignment overhead.
    size_t req_size = block_size + sizeof(block_header_t) + ALIGN_SIZE;
    size_t npages = __builtin_align_up(req_size, PAGESIZE) / PAGESIZE;

    size_t old = __builtin_wasm_memory_grow(0, npages);
    if (old == SIZE_MAX) return NULL;

    tlsf_add_pool((char*)(old * PAGESIZE), (char*)((old + npages) * PAGESIZE));

    free_block = tlsf_find_free_block(block_size);
    if (free_block == NULL) return NULL;
  }

  tlsf_remove_free_block(free_block);
  block_header_t* block = &free_block->header;

  tlsf_block_set_free(block, false);
  tlsf_block_set_prev_free(tlsf_block_next_phys(block), false);

  block_header_t* remaining = tlsf_block_split(block, block_size);
  if (remaining) tlsf_block_free(remaining);

  return tlsf_block_to_payload(block);
}

void* realloc(void* ptr, size_t size) {
  if (size == 0) {
    free(ptr);
    return NULL;
  }
  if (size > PTRDIFF_MAX) return NULL;
  if (ptr == NULL) return malloc(size);

  block_header_t* block = tlsf_payload_to_block(ptr);
  size_t old_size = tlsf_block_get_size(block);
  size_t block_size = tlsf_request_to_block_size(size);

  // Shrink or stay the same in-place.
  if (block_size <= old_size) {
    block_header_t* remaining = tlsf_block_split(block, block_size);
    if (remaining) tlsf_block_free(remaining);
    return ptr;
  }

  // Expand in-place if the next physical block is free and large enough.
  block_header_t* next = tlsf_block_next_phys(block);
  if (tlsf_block_is_free(next) &&
      old_size + tlsf_block_get_size(next) >= block_size) {
    tlsf_remove_free_block((free_block_t*)next);
    tlsf_block_set_size(block, old_size + tlsf_block_get_size(next));
    tlsf_block_set_prev_free(tlsf_block_next_phys(block), false);

    block_header_t* remaining = tlsf_block_split(block, block_size);
    if (remaining) tlsf_block_free(remaining);
    return ptr;
  }

  // Fallback to allocating new space and copying.
  void* new_ptr = malloc(size);
  if (new_ptr != NULL) {
    size_t old_payload_size = old_size - sizeof(block_header_t);
    __builtin_memcpy(new_ptr, ptr, old_payload_size);
    free(ptr);
  }
  return new_ptr;
}

void* memalign(size_t align, size_t size) {
  if (size == 0 || size > PTRDIFF_MAX) return NULL;
  if (align <= 0 || (align & (align - 1))) return NULL;
  if (align <= ALIGN_SIZE) return malloc(size);

  // Request enough space to guarantee finding an aligned boundary,
  // and to ensure any front padding can become a valid free block.
  size_t need;
  if (__builtin_add_overflow(size, align + BLOCK_SIZE_MIN - ALIGN_SIZE,
                             &need)) {
    return NULL;
  }

  char* raw_payload = (char*)malloc(need);
  if (raw_payload == NULL) return NULL;

  char* aligned_payload = __builtin_align_up(raw_payload, align);

  // Ensure the front padding is either zero or at least BLOCK_SIZE_MIN.
  if (aligned_payload > raw_payload &&
      aligned_payload - raw_payload < (ptrdiff_t)BLOCK_SIZE_MIN) {
    aligned_payload += align;
  }

  block_header_t* block = tlsf_payload_to_block(raw_payload);

  if (aligned_payload > raw_payload) {
    size_t front_padding = aligned_payload - raw_payload;
    block_header_t* aligned_block = tlsf_block_split(block, front_padding);

    tlsf_block_free(block);
    block = aligned_block;
  }

  size_t block_size = tlsf_request_to_block_size(size);
  block_header_t* remaining = tlsf_block_split(block, block_size);
  if (remaining) tlsf_block_free(remaining);

  return tlsf_block_to_payload(block);
}

void* calloc(size_t nelem, size_t elsize) {
  size_t need;
  if (__builtin_mul_overflow(nelem, elsize, &need)) return NULL;
  void* ptr = malloc(need);
  if (ptr) __builtin_memset(ptr, 0, need);
  return ptr;
}

void* aligned_alloc(size_t align, size_t size) {
  if (align <= 0 || ((align | size) & (align - 1))) return NULL;
  return memalign(align, size);
}

size_t malloc_usable_size(void* ptr) {
  if (ptr == NULL) return 0;
  block_header_t* block = tlsf_payload_to_block(ptr);
  return tlsf_block_get_size(block) - sizeof(block_header_t);
}

size_t malloc_good_size(size_t size) {
  if (size == 0 || size > PTRDIFF_MAX) return size;
  return tlsf_request_to_block_size(size) - sizeof(block_header_t);
}

extern char __heap_base[];
extern char __heap_end[];

static void init_allocator(void) { tlsf_add_pool(__heap_base, __heap_end); }
