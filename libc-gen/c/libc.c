#if __STDC_HOSTED__ == 1
#error "libc must be compiled with -ffreestanding"
#endif

#include "ctype.c"
#include "errno.c"
#include "fenv.c"
#include "math.c"
#include "stdio.c"
#include "stdlib.c"
#include "string.c"
#include "strings.c"
#include "time.c"
#include "unistd.c"
