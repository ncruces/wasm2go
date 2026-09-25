#pragma once

#include <stdint.h>
#include <sys/types.h>

#undef SEEK_SET
#undef SEEK_CUR
#undef SEEK_END
#define SEEK_SET 0
#define SEEK_CUR 1
#define SEEK_END 2

void* sbrk(intptr_t);
