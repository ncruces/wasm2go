#include <fenv.h>

int fegetround(void) { return 0; }
int fesetround(int r) { return 0; }
int feclearexcept(int mask) { return 0; }
int feraiseexcept(int mask) { return 0; }
int fetestexcept(int mask) { return 0; }
