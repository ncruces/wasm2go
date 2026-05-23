package libc

import (
	"testing"
	"time"
)

func Test_gmtime_r(t *testing.T) {
	memory = make([]byte, 1024)
	timer := ptr(8)
	buf := ptr(16)

	// 1234567890 = 2009-02-13 23:31:30 UTC
	ts := int64(1234567890)
	store64(memory[uptr(timer):], uint64(ts))

	res := gmtime_r(timer, buf)
	if res != buf {
		t.Errorf("got %v, want %v for return pointer", res, buf)
	}

	wantFields := []uint32{
		30,  // tm_sec
		31,  // tm_min
		23,  // tm_hour
		13,  // tm_mday
		1,   // tm_mon (0-11, February is 1)
		109, // tm_year (years since 1900)
		5,   // tm_wday (0-6, Friday is 5)
		43,  // tm_yday (0-365, 44th day of the year - 1)
		0,   // tm_isdst
		0,   // tm_gmtoff
		0,   // tm_zone
	}

	for i, want := range wantFields {
		got := load32(memory[uptr(buf)+uptr(i)*4:])
		if got != want {
			t.Errorf("field %d: got %v, want %v", i, got, want)
		}
	}
}

func Test_localtime_r(t *testing.T) {
	memory = make([]byte, 1024)
	timer := ptr(8)
	buf := ptr(16)

	ts := int64(1234567890)
	store64(memory[uptr(timer):], uint64(ts))

	res := localtime_r(timer, buf)
	if res != buf {
		t.Errorf("got %v, want %v for return pointer", res, buf)
	}

	// Dynamically calculate expected values based on system's local timezone
	tm := time.Unix(ts, 0)
	var isdst uint32
	if tm.IsDST() {
		isdst = 1
	}
	_, zone := tm.Zone()

	wantFields := []uint32{
		uint32(tm.Second()),
		uint32(tm.Minute()),
		uint32(tm.Hour()),
		uint32(tm.Day()),
		uint32(tm.Month() - time.January),
		uint32(tm.Year() - 1900),
		uint32(tm.Weekday() - time.Sunday),
		uint32(tm.YearDay() - 1),
		isdst,
		uint32(zone),
		uint32(0),
	}

	for i, want := range wantFields {
		got := load32(memory[uptr(buf)+uptr(i)*4:])
		if got != want {
			t.Errorf("field %d: got %v, want %v", i, got, want)
		}
	}
}
