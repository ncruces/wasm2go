#pragma once

#include <stdint.h>

typedef struct jmp_buf_impl {
  void* func_invocation_id;
  uint32_t label;
} jmp_buf[1];

__attribute__((returns_twice)) int setjmp(jmp_buf);
__attribute__((noreturn)) void longjmp(jmp_buf, int);
