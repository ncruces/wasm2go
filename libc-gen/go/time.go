package libc

import "time"

func localtime_r(timer, buf ptr) ptr {
	t := load64(memory[uptr(timer):])
	storetime_r(memory[uptr(buf):], time.Unix(int64(t), 0))
	return buf
}

func gmtime_r(timer, buf ptr) ptr {
	t := load64(memory[uptr(timer):])
	storetime_r(memory[uptr(buf):], time.Unix(int64(t), 0).UTC())
	return buf
}

func storetime_r(buf []byte, t time.Time) {
	const size = 32 / 8
	var isdst uint32
	if t.IsDST() {
		isdst = 1
	}
	_, zone := t.Zone()

	// https://pubs.opengroup.org/onlinepubs/9799919799/basedefs/time.h.html
	store32(buf[0*size:], uint32(t.Second()))
	store32(buf[1*size:], uint32(t.Minute()))
	store32(buf[2*size:], uint32(t.Hour()))
	store32(buf[3*size:], uint32(t.Day()))
	store32(buf[4*size:], uint32(t.Month()-time.January))
	store32(buf[5*size:], uint32(t.Year()-1900))
	store32(buf[6*size:], uint32(t.Weekday()-time.Sunday))
	store32(buf[7*size:], uint32(t.YearDay()-1))
	store32(buf[8*size:], isdst)
	store32(buf[9*size:], uint32(zone))
	store32(buf[10*size:], 0)
}

func gettimeofday(arg, _ ptr) int32 {
	if arg != 0 {
		now := time.Now()
		store64(memory[uptr(arg):], uint64(now.Unix()))
		store32(memory[uptr(arg)+8:], uint32(now.Nanosecond()/1000))
	}
	return 0
}
