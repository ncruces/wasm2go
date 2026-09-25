#include <setjmp.h>

int test(void) {
  static jmp_buf env;
  int v = setjmp(env);
  if (v != 0) return v;
  longjmp(env, 42);
  return -1;
}
