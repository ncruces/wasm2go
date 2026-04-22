package libc

import "bytes"

func memchr(s, c, n ptr) ptr {
	b := memory[uptr(s):]
	if uint(len(b)) > uint(uptr(n)) {
		b = b[:uptr(n)]
	}
	if i := bytes.IndexByte(b, byte(c)); i >= 0 {
		return s + ptr(i)
	}
	return 0
}

func memcmp(s1, s2, n ptr) int32 {
	e1, e2 := s1+n, s2+n
	b1 := memory[uptr(s1):uptr(e1)]
	b2 := memory[uptr(s2):uptr(e2)]
	return int32(bytes.Compare(b1, b2))
}

func strlen(s ptr) ptr {
	return ptr(bytes.IndexByte(memory[uptr(s):], 0))
}

func strchr(s, c ptr) ptr {
	s = strchrnul(s, c)
	if memory[uptr(s)] == byte(c) {
		return s
	}
	return 0
}

func strchrnul(s, c ptr) ptr {
	b := memory[uptr(s):]
	b = b[:bytes.IndexByte(b, 0)]
	sz := len(b)
	if c := byte(c); c != 0 {
		if i := bytes.IndexByte(b, c); i >= 0 {
			sz = i
		}
	}
	return s + ptr(sz)
}

func strrchr(s, c ptr) ptr {
	b := memory[uptr(s):]
	b = b[:bytes.IndexByte(b, 0)+1]
	if i := bytes.LastIndexByte(b, byte(c)); i >= 0 {
		return s + ptr(i)
	}
	return 0
}

func strcmp(s1, s2 ptr) int32 {
	b1 := memory[uptr(s1):]
	b2 := memory[uptr(s2):]
	sz := min(len(b1), len(b2))
	if i := bytes.IndexByte(b1[:sz], 0); i >= 0 {
		sz = i + 1
	}
	return int32(bytes.Compare(b1[:sz], b2[:sz]))
}

func strncmp(s1, s2, n ptr) int32 {
	b1 := memory[uptr(s1):]
	b2 := memory[uptr(s2):]
	sz := int(min(uint(len(b1)), uint(len(b2)), uint(uptr(n))))
	if i := bytes.IndexByte(b1[:sz], 0); i >= 0 {
		sz = i + 1
	}
	return int32(bytes.Compare(b1[:sz], b2[:sz]))
}

func strspn(s, accept ptr) ptr {
	b := memory[uptr(s):]
	a := memory[uptr(accept):]
	a = a[:bytes.IndexByte(a, 0)]

	for i, b := range b {
		if bytes.IndexByte(a, b) < 0 {
			return ptr(i)
		}
	}
	return ptr(len(b))
}

func strcspn(s, reject ptr) ptr {
	b := memory[uptr(s):]
	r := memory[uptr(reject):]
	r = r[:bytes.IndexByte(r, 0)+1]

	for i, b := range b {
		if bytes.IndexByte(r, b) >= 0 {
			return ptr(i)
		}
	}
	return ptr(len(b))
}

func strstr(haystack, needle ptr) ptr {
	h := memory[uptr(haystack):]
	n := memory[uptr(needle):]
	h = h[:bytes.IndexByte(h, 0)]
	n = n[:bytes.IndexByte(n, 0)]
	i := bytes.Index(h, n)
	if i < 0 {
		return 0
	}
	return haystack + ptr(i)
}

func strcpy(d, s ptr) ptr {
	b := memory[uptr(s):]
	b = b[:bytes.IndexByte(b, 0)+1]
	copy(memory[uptr(d):], b)
	return d
}
