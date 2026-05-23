#include <stdio.h>
#include <string.h>

int putchar(int ch) { return fputc(ch, stdout); }

void rewind(FILE* stream) { fseek(stream, 0, SEEK_SET); }

int printf(const char* restrict fmt, ...) {
  va_list args;
  va_start(args, fmt);
  int ret = vprintf(fmt, args);
  va_end(args);
  return ret;
}

int fprintf(FILE* restrict stream, const char* restrict fmt, ...) {
  va_list args;
  va_start(args, fmt);
  int ret = vfprintf(stream, fmt, args);
  va_end(args);
  return ret;
}

int sprintf(char* restrict buf, const char* restrict fmt, ...) {
  int ret;
  va_list va;
  va_start(va, fmt);
  ret = vsprintf(buf, fmt, va);
  va_end(va);
  return ret;
}

int snprintf(char* restrict buf, size_t count, const char* restrict fmt, ...) {
  int ret;
  va_list va;
  va_start(va, fmt);
  ret = vsnprintf(buf, (int)count, fmt, va);
  va_end(va);
  return ret;
}

int vprintf(const char* restrict fmt, va_list va) {
  return vfprintf(stdout, fmt, va);
}

#ifdef SQLITE3_H

#define SQLITE_MAX_LENGTH 1000000000

int vfprintf(FILE* restrict stream, const char* restrict fmt, va_list va) {
  char* str = sqlite3_vmprintf(fmt, va);
  if (str == NULL) {
    return -1;
  }
  size_t len = strlen(str);
  int ret = fwrite(str, 1, len, stream);
  sqlite3_free(str);
  return ret;
}

int vsprintf(char* restrict buf, const char* restrict fmt, va_list va) {
  sqlite3_vsnprintf(SQLITE_MAX_LENGTH, buf, fmt, va);
  return (int)strlen(buf);
}

int vsnprintf(char* restrict buf, size_t count, const char* restrict fmt,
              va_list va) {
  int n = count > SQLITE_MAX_LENGTH ? SQLITE_MAX_LENGTH : (int)count;
  if (n > 0) {
    va_list va2;
    va_copy(va2, va);
    sqlite3_vsnprintf(n, buf, fmt, va2);
    va_end(va2);

    size_t len = strlen(buf);
    if (len < (size_t)(n - 1)) {
      return (int)len;
    }
  }

  char* str = sqlite3_vmprintf(fmt, va);
  if (str == NULL) {
    return -1;
  }
  size_t len = strlen(str);
  if (count > (size_t)n) {
    size_t cpy = len < count - 1 ? len : count - 1;
    memcpy(buf, str, cpy);
    buf[cpy] = '\0';
  }
  sqlite3_free(str);
  return (int)len;
}

#elif defined(STB_SPRINTF_H_INCLUDE)

static char* vfprintf_cb(const char* buf, void* user, int len) {
  fwrite(buf, 1, len, (FILE*)user);
  return (char*)buf;
}

int vfprintf(FILE* restrict stream, const char* restrict fmt, va_list va) {
  char buf[STB_SPRINTF_MIN];
  return STB_SPRINTF_DECORATE(vsprintfcb)(vfprintf_cb, stream, buf, fmt, va);
}

int vsprintf(char* restrict buf, const char* restrict fmt, va_list va) {
  return STB_SPRINTF_DECORATE(vsprintf)(buf, fmt, va);
}

int vsnprintf(char* restrict buf, size_t count, const char* restrict fmt,
              va_list va) {
  return STB_SPRINTF_DECORATE(vsnprintf)(buf, (int)count, fmt, va);
}

#endif
