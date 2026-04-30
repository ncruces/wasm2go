#pragma once

#include <stddef.h>

__attribute__((noreturn)) void abort(void);
__attribute__((noreturn)) void exit(int);

int system(const char*);

int abs(int);
int atoi(const char*);
long atol(const char*);
long long atoll(const char*);
double atof(const char*);

float strtof(const char* restrict, char** restrict);
double strtod(const char* restrict, char** restrict);
long strtol(const char* restrict, char** restrict, int);
long long strtoll(const char* restrict, char** restrict, int);
unsigned long strtoul(const char* restrict, char** restrict, int);
unsigned long long strtoull(const char* restrict, char** restrict, int);

#define abs(x) (__builtin_abs(x))
#define alloca(x) (__builtin_alloca(x))

void free(void*);
__attribute__((malloc)) void* malloc(size_t);
__attribute__((malloc)) void* calloc(size_t, size_t);
__attribute__((malloc)) void* aligned_alloc(size_t, size_t);
void* realloc(void*, size_t);

void qsort(void*, size_t, size_t, int (*)(const void*, const void*));
void* bsearch(const void *, const void *, size_t, size_t, int (*)(const void*, const void*));
