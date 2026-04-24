#pragma once

#include <stddef.h>

typedef long long time_t;

struct tm {
  int tm_sec;
  int tm_min;
  int tm_hour;
  int tm_mday;
  int tm_mon;
  int tm_year;
  int tm_wday;
  int tm_yday;
  int tm_isdst;

  long tm_gmtoff;
  const char* tm_zone;
};

time_t time(time_t*);
struct tm* gmtime_r(const time_t*, struct tm*);
struct tm* localtime_r(const time_t*, struct tm*);
