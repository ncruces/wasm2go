#pragma once

#include <stdarg.h>
#include <stddef.h>

typedef void FILE;

#define stdin (FILE*)(0)
#define stdout (FILE*)(1)
#define stderr (FILE*)(2)

#define SEEK_SET 0
#define SEEK_CUR 1
#define SEEK_END 2

int printf(const char* restrict, ...);
int fprintf(FILE* restrict, const char* restrict, ...);
int vfprintf(FILE* restrict, const char* restrict, va_list);

int putchar(int);
int puts(const char*);

FILE* fopen(const char*, const char*);
int fclose(FILE*);
int fputc(int, FILE*);
size_t fread(void* restrict, size_t, size_t, FILE* restrict);
size_t fwrite(const void* restrict, size_t, size_t, FILE* restrict);
int fseek(FILE*, long, int);
int fflush(FILE*);
long ftell(FILE*);
void rewind(FILE*);
