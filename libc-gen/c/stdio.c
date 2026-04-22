#include <stdio.h>

int putchar(int ch) { return fputc(ch, stdout); }

void rewind(FILE* stream) { fseek(stream, 0, SEEK_SET); }

int printf(const char* restrict format, ...) {
  va_list args;
  va_start(args, format);
  int ret = vfprintf(stdout, format, args);
  va_end(args);
  return ret;
}

int fprintf(FILE* restrict stream, const char* restrict format, ...) {
  va_list args;
  va_start(args, format);
  int ret = vfprintf(stream, format, args);
  va_end(args);
  return ret;
}

#ifdef SQLITE3_H

int vfprintf(FILE* restrict stream, const char* restrict format, va_list args) {
  char* str = sqlite3_vmprintf(format, args);
  if (str == NULL) {
    return -1;
  }
  size_t len = strlen(str);
  int ret = fwrite(str, 1, len, stream);
  sqlite3_free(str);
  return ret;
}

#endif
