package libc

import (
	"bytes"
	"math/bits"
)

func memchr(s sptr_t, c int32, n sptr_t) sptr_t {
	b := memory[uptr_t(s):]
	if wptr_t(len(b)) > wptr_t(uptr_t(n)) {
		b = b[:uptr_t(n)]
	}
	if i := bytes.IndexByte(b, byte(c)); i >= 0 {
		return s + sptr_t(i)
	}
	return 0
}

func memmem(haystack, hn, needle, nn sptr_t) sptr_t {
	hn, nn = haystack+hn, needle+nn
	h := memory[uptr_t(haystack):uptr_t(hn)]
	n := memory[uptr_t(needle):uptr_t(nn)]
	i := bytes.Index(h, n)
	if i < 0 {
		return 0
	}
	return haystack + sptr_t(i)
}

func memcmp(s1, s2, n sptr_t) int32 {
	if s1 == s2 {
		return 0
	}
	e1, e2 := s1+n, s2+n
	b1 := memory[uptr_t(s1):uptr_t(e1)]
	b2 := memory[uptr_t(s2):uptr_t(e2)]
	return int32(bytes.Compare(b1, b2))
}

func bcmp(s1, s2, n sptr_t) int32 {
	if s1 == s2 {
		return 0
	}
	e1, e2 := s1+n, s2+n
	b1 := memory[uptr_t(s1):uptr_t(e1)]
	b2 := memory[uptr_t(s2):uptr_t(e2)]
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
	b := memory[uptr_t(s):]
	b = b[:bytes.IndexByte(b, 0)]
	sz := len(b)
	if c := byte(c); c != 0 {
		if i := bytes.IndexByte(b, c); i >= 0 {
			sz = i
		}
	}
	return s + sptr_t(sz)
}

func strrchr(s sptr_t, c int32) sptr_t {
	b := memory[uptr_t(s):]
	b = b[:bytes.IndexByte(b, 0)+1]
	if i := bytes.LastIndexByte(b, byte(c)); i >= 0 {
		return s + sptr_t(i)
	}
	return 0
}

func strstr(haystack, needle sptr_t) sptr_t {
	h := memory[uptr_t(haystack):]
	n := memory[uptr_t(needle):]
	h = h[:bytes.IndexByte(h, 0)]
	n = n[:bytes.IndexByte(n, 0)]
	i := bytes.Index(h, n)
	if i < 0 {
		return 0
	}
	return haystack + sptr_t(i)
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
	sz := int(min(wptr_t(len(b1)), wptr_t(len(b2)), wptr_t(uptr_t(n))))
	if i := bytes.IndexByte(b2[:sz], 0); i >= 0 {
		sz = i + 1
	}
	return int32(bytes.Compare(b1[:sz], b2[:sz]))
}

func strspn(s, accept sptr_t) sptr_t {
	b := memory[uptr_t(s):]
	a := memory[uptr_t(accept):]
	a = a[:bytes.IndexByte(a, 0)]

	set := makeByteSet(a)
	for i, c := range b {
		if set[c/bits.UintSize]&(1<<(c%bits.UintSize)) == 0 {
			return sptr_t(i)
		}
	}
	return sptr_t(len(b))
}

func strcspn(s, reject sptr_t) sptr_t {
	b := memory[uptr_t(s):]
	r := memory[uptr_t(reject):]
	r = r[:bytes.IndexByte(r, 0)+1]
	if len(r) == 2 || len(r) == 3 {
		c1, c2 := r[0], r[0]
		if len(r) == 3 {
			c2 = r[1]
		}
		n := min(len(b), 16)
		for i, c := range b[:n] {
			if c == 0 || c == c1 || c == c2 {
				return sptr_t(i)
			}
		}
		b = b[n:]
		end := bytes.IndexByte(b, 0)
		if end < 0 {
			end = len(b)
		}
		b = b[:end]
		if i := bytes.IndexByte(b, c1); i >= 0 {
			end = i
		}
		if c2 != c1 {
			if i := bytes.IndexByte(b[:end], c2); i >= 0 {
				end = i
			}
		}
		return sptr_t(n + end)
	}

	set := makeByteSet(r)
	for i, c := range b {
		if set[c/bits.UintSize]&(1<<(c%bits.UintSize)) != 0 {
			return sptr_t(i)
		}
	}
	return sptr_t(len(b))
}

func makeByteSet(chars []byte) (set [256 / bits.UintSize]uint) {
	for _, c := range chars {
		set[c/bits.UintSize] |= 1 << (c % bits.UintSize)
	}
	return set
}
