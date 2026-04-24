#pragma once

#define FE_TONEAREST 0

int fegetround(void);
int fesetround(int);
int feclearexcept(int);
int feraiseexcept(int);
int fetestexcept(int);
