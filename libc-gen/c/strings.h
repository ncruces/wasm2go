#pragma once

int bcmp(const void*, const void*, size_t);
void bcopy(const void*, void*, size_t);
void bzero(void*, size_t);
int ffs(int);

#define ffs(x) (__builtin_ffs(x))
