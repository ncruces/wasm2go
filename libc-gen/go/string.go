package libc

import (
	"bytes"
	"math/bits"
)

func memchr(s sptr_t, c int32, n sptr_t) sptr_t {
	b := memory[uptr_t(s):]
	if uint64(uptr_t(n)) < uint64(len(b)) {
		b = b[:uptr_t(n)]
	}
	if i := bytes.IndexByte(b, byte(c)); i >= 0 {
		return s + sptr_t(i)
	}
	return 0
}

func memmem(haystack, hn, needle, nn sptr_t) sptr_t {
	hn, nn = haystack+hn, needle+nn
	h := memory[uptr_t(haystack):uptr_t(hn):len(memory)]
	n := memory[uptr_t(needle):uptr_t(nn):len(memory)]
	if i := bytes.Index(h, n); i >= 0 {
		return haystack + sptr_t(i)
	}
	return 0
}

func memcmp(s1, s2, n sptr_t) int32 {
	if s1 == s2 {
		return 0
	}
	e1, e2 := s1+n, s2+n
	b1 := memory[uptr_t(s1):uptr_t(e1):len(memory)]
	b2 := memory[uptr_t(s2):uptr_t(e2):len(memory)]
	return int32(bytes.Compare(b1, b2))
}

func bcmp(s1, s2, n sptr_t) int32 {
	if s1 == s2 {
		return 0
	}
	e1, e2 := s1+n, s2+n
	b1 := memory[uptr_t(s1):uptr_t(e1):len(memory)]
	b2 := memory[uptr_t(s2):uptr_t(e2):len(memory)]
	if bytes.Equal(b1, b2) {
		return 0
	}
	return 1
}

func strlen(s sptr_t) sptr_t {
	return sptr_t(bytes.IndexByte(memory[uptr_t(s):], 0))
}

func strchr(s sptr_t, c int32) sptr_t {
	s = strchrnul(s, c)
	if memory[uptr_t(s)] == byte(c) {
		return s
	}
	return 0
}

func strchrnul(s sptr_t, c int32) sptr_t {
	for i, d := range memory[uptr_t(s):] {
		if d == 0 || d == byte(c) {
			return s + sptr_t(i)
		}
	}
	return -1
}

func strrchr(s sptr_t, c int32) sptr_t {
	b := memory[uptr_t(s):]
	b = b[: bytes.IndexByte(b, 0)+1 : len(b)]
	if i := bytes.LastIndexByte(b, byte(c)); i >= 0 {
		return s + sptr_t(i)
	}
	return 0
}

func strstr(haystack, needle sptr_t) sptr_t {
	h := memory[uptr_t(haystack):]
	n := memory[uptr_t(needle):]
	h = h[:bytes.IndexByte(h, 0):len(h)]
	n = n[:bytes.IndexByte(n, 0):len(n)]
	if i := bytes.Index(h, n); i >= 0 {
		return haystack + sptr_t(i)
	}
	return 0
}

func strcmp(s1, s2 sptr_t) int32 {
	if s1 == s2 {
		return 0
	}
	b1 := memory[uptr_t(s1):]
	b2 := memory[uptr_t(s2):]
	sz := min(len(b1), len(b2))
	if i := bytes.IndexByte(b2[:sz], 0); i >= 0 {
		sz = i + 1
	}
	return int32(bytes.Compare(b1[:sz], b2[:sz]))
}

func strncmp(s1, s2, n sptr_t) int32 {
	if s1 == s2 {
		return 0
	}
	b1 := memory[uptr_t(s1):]
	b2 := memory[uptr_t(s2):]
	sz := int(min(uint64(len(b1)), uint64(len(b2)), uint64(uptr_t(n))))
	if i := bytes.IndexByte(b2[:sz], 0); i >= 0 {
		sz = i + 1
	}
	return int32(bytes.Compare(b1[:sz], b2[:sz]))
}

func strspn(s, accept sptr_t) sptr_t {
	b := memory[uptr_t(s):]
	a := memory[uptr_t(accept):]

	set := makeByteSet(a)
	for i, c := range b {
		if set[c/bits.UintSize]&(1<<(c%bits.UintSize)) == 0 {
			return sptr_t(i)
		}
	}
	return -1
}

func strcspn(s, reject sptr_t) sptr_t {
	b := memory[uptr_t(s):]
	r := memory[uptr_t(reject):]

	set := makeByteSet(r)
	set[0] |= 1
	for i, c := range b {
		if set[c/bits.UintSize]&(1<<(c%bits.UintSize)) != 0 {
			return sptr_t(i)
		}
	}
	return -1
}

func makeByteSet(chars []byte) (set [256 / bits.UintSize]uint) {
	for _, c := range chars {
		if c == 0 {
			break
		}
		set[c/bits.UintSize] |= 1 << (c % bits.UintSize)
	}
	return set
}
