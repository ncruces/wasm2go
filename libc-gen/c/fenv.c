#include <fenv.h>

int fegetround(void) { return FE_TONEAREST; }
int fesetround(int r) { return 0; }

int feclearexcept(int mask) { return mask; }
int feraiseexcept(int mask) { return mask; }
int fetestexcept(int mask) { return 0; }
