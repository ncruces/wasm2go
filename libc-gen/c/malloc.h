#pragma once

#include <stdlib.h>

size_t malloc_usable_size(void*);
size_t malloc_good_size(size_t);

void *memalign(size_t alignment, size_t size);
