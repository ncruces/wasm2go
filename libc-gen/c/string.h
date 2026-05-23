#pragma once

#include <stddef.h>

void* memset(void*, int, size_t);
void* memcpy(void* restrict, const void* restrict, size_t);
void* memmove(void*, const void*, size_t);

#define memset(p, v, n) (__builtin_memset(p, v, n))
#define memcpy(d, s, n) (__builtin_memcpy(d, s, n))
#define memmove(d, s, n) (__builtin_memmove(d, s, n))

void* memchr(const void*, int, size_t);
void* memrchr(const void*, int, size_t);
int memcmp(const void*, const void*, size_t);
void* memmem(const void*, size_t, const void*, size_t);
void* mempcpy(void* restrict, const void* restrict, size_t n);
void* memccpy(void* restrict, const void* restrict, int, size_t);

size_t strlen(const char*);
size_t strnlen(const char*, size_t);
size_t strspn(const char*, const char*);
size_t strcspn(const char*, const char*);
char* strpbrk(const char*, const char*);
char* strtok(char* restrict, const char* restrict);
char* strchr(const char*, int);
char* strrchr(const char*, int);
char* strchrnul(const char*, int);
int strcmp(const char*, const char*);
int strncmp(const char*, const char*, size_t);
char* strstr(const char*, const char*);
char* strcat(char* restrict, const char* restrict);
char* strcpy(char* restrict, const char* restrict);
char* stpcpy(char* restrict, const char* restrict);
char* strncat(char* restrict, const char* restrict, size_t);
char* strncpy(char* restrict, const char* restrict, size_t);
char* stpncpy(char* restrict, const char* restrict, size_t);
char* strdup(const char*);
char* strndup(const char*, size_t);
size_t strlcat(char* restrict, const char* restrict, size_t);
size_t strlcpy(char* restrict, const char* restrict, size_t);
