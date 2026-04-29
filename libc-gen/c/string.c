#include "string.h"
#include "stdlib.h"

__attribute__((always_inline))
void* (memset)(void* p, int v, size_t n) {
  return __builtin_memset(p, v, n);
}

__attribute__((always_inline))
void* (memcpy)(void* restrict d, const void* restrict s, size_t n) {
  return __builtin_memcpy(d, s, n);
}

__attribute__((always_inline))
void* (memmove)(void* restrict d, const void* restrict s, size_t n) {
  return __builtin_memmove(d, s, n);
}

void* mempcpy(void* restrict d, const void* restrict s, size_t n) {
  return (char*)memcpy(d, s, n) + n;
}

void* memccpy(void* restrict d, const void* restrict s, int c, size_t n) {
  const void* m = memchr(s, c, n);
  if (m != NULL) {
    n = (char*)m - (char*)s + 1;
    m = (char*)d + n;
  }
  memcpy(d, s, n);
  return (void*)m;
}

size_t strnlen(const char* s, size_t n) {
  const char* m = memchr(s, 0, n);
  return m ? m - s : n;
}

char* strpbrk(const char* s, const char* b) {
  s += strcspn(s, b);
  return *s ? (char*)s : 0;
}

char* strtok(char* restrict s, const char* restrict sep) {
  static char* p;
  if (!s && !(s = p)) return NULL;
  s += strspn(s, sep);
  if (!*s) return p = 0;
  p = s + strcspn(s, sep);
  if (*p) *p++ = 0;
  else p = 0;
  return s;
}

char* strcat(char* restrict d, const char* restrict s) {
  strcpy(d + strlen(d), s);
  return d;
}

char* strcpy(char* restrict d, const char* restrict s) {
  stpcpy(d, s);
  return d;
}

char* stpcpy(char* restrict d, const char* restrict s) {
  size_t slen = strlen(s);
  memcpy(d, s, slen + 1);
  return d + slen;
}

char* strncat(char* restrict d, const char* restrict s, size_t n) {
  size_t dlen = strlen(d);
  size_t slen = strnlen(s, n);
  memcpy(d + dlen, s, slen);
  d[dlen + slen] = 0;
  return d;
}

char* strncpy(char* restrict d, const char* restrict s, size_t n) {
  stpncpy(d, s, n);
  return d;
}

char* stpncpy(char* restrict d, const char* restrict s, size_t n) {
  size_t slen = strnlen(s, n);
  memcpy(d, s, slen);
  memset(d + slen, 0, n - slen);
  return d + slen;
}

char* strdup(const char* s) {
  size_t slen = strlen(s);
  char* d = malloc(slen + 1);
  if (d == NULL) return NULL;
  return memcpy(d, s, slen + 1);
}

char* strndup(const char* s, size_t n) {
  size_t slen = strnlen(s, n);
  char* d = malloc(slen + 1);
  if (d == NULL) return NULL;
  memcpy(d, s, slen);
  d[slen] = 0;
  return d;
}

size_t strlcat(char* d, const char* s, size_t n) {
  size_t dlen = strnlen(d, n);
  if (dlen == n) return dlen + strlen(s);
  return dlen + strlcpy(d + dlen, s, n - dlen);
}

size_t strlcpy(char* restrict d, const char* restrict s, size_t n) {
  size_t slen = strlen(s);
  if (n-- > 0) {
    if (n > slen) n = slen;
    memcpy(d, s, n);
    d[n] = 0;
  }
  return slen;
}
