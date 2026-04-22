#pragma once

#include <stddef.h>

__attribute__((noreturn)) void abort(void);
__attribute__((noreturn)) void exit(int);

int system(const char*);

int abs(int);
int atoi(const char*);
long strtol(const char* restrict, char** restrict, int);
double strtod(const char* restrict, char** restrict);

#define abs(x) (__builtin_abs(x))

void free(void*);
__attribute__((malloc)) void* malloc(size_t);
__attribute__((malloc)) void* calloc(size_t, size_t);
__attribute__((malloc)) void* aligned_alloc(size_t, size_t);
void* realloc(void*, size_t);

void qsort(void*, size_t, size_t, int (*)(const void*, const void*));
