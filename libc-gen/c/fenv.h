#pragma once

#define FE_TONEAREST  0

int fegetround(void);
int fesetround(int);

#define FE_ALL_EXCEPT 0

int feclearexcept(int);
int feraiseexcept(int);
int fetestexcept(int);
