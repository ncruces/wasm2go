package libc

import "testing"

func writeString(p sptr_t, s string) {
	copy(memory[uptr_t(p):], s)
	memory[uptr_t(p)+uptr_t(len(s))] = 0
}

func Test_memchr(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello world")

	if got := memchr(10, 'w', sptr_t(len("hello world"))); got != 16 {
		t.Errorf("got %v, want 16", got)
	}
	if got := memchr(10, 'z', sptr_t(len("hello world"))); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	if got := memchr(10, 'w', sptr_t(len("hello"))); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func Test_memmem(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello world")
	writeString(30, "world")
	writeString(40, "z")
	writeString(50, "")

	if got, want := memmem(10, sptr_t(len("hello world")), 30, sptr_t(len("world"))), 10+len("hello "); got != sptr_t(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	if got := memmem(10, sptr_t(len("hello world")), 50, 0); got != 10 {
		t.Errorf("got %v, want 10", got)
	}
	if got := memmem(10, sptr_t(len("hello world")), 40, 1); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func Test_memcmp(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "abc")
	writeString(20, "abd")

	if got := memcmp(10, 20, 2); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	if got := memcmp(10, 20, 3); got >= 0 {
		t.Errorf("got %v, want < 0", got)
	}
	if got := memcmp(20, 10, 3); got <= 0 {
		t.Errorf("got %v, want > 0", got)
	}
}

func Test_bcmp(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "abc")
	writeString(20, "abd")

	if got := bcmp(10, 20, 2); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	if got := bcmp(10, 20, 3); got != 1 {
		t.Errorf("got %v, want 1", got)
	}
	if got := bcmp(20, 10, 3); got != 1 {
		t.Errorf("got %v, want 1", got)
	}
}

func Test_strlen(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello")
	writeString(20, "")

	if got, want := strlen(10), len("hello"); got != sptr_t(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	if got := strlen(20); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func Test_strchr(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello")

	if got, want := strchr(10, 'l'), 10+2; got != sptr_t(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	if got := strchr(10, 'z'); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	if got, want := strchr(10, 0), 10+len("hello"); got != sptr_t(want) {
		t.Errorf("got %v, want %d", got, want)
	}
}

func Test_strchrnul(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello")

	if got, want := strchrnul(10, 'l'), 10+2; got != sptr_t(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	if got, want := strchrnul(10, 'z'), 10+len("hello"); got != sptr_t(want) {
		t.Errorf("got %v, want %d", got, want)
	}
}

func Test_strrchr(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello")

	if got, want := strrchr(10, 'l'), 10+3; got != sptr_t(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	if got := strrchr(10, 'z'); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	if got, want := strrchr(10, 0), 10+len("hello"); got != sptr_t(want) {
		t.Errorf("got %v, want %d", got, want)
	}
}

func Test_strstr(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello world")
	writeString(30, "world")
	writeString(40, "z")
	writeString(50, "")

	if got, want := strstr(10, 30), 10+len("hello "); got != sptr_t(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	if got := strstr(10, 50); got != 10 {
		t.Errorf("got %v, want 10", got)
	}
	if got := strstr(10, 40); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func Test_strcmp(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "abc")
	writeString(20, "abc")
	writeString(30, "abd")

	if got := strcmp(10, 20); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	if got := strcmp(10, 30); got >= 0 {
		t.Errorf("got %v, want < 0", got)
	}
	if got := strcmp(30, 10); got <= 0 {
		t.Errorf("got %v, want > 0", got)
	}
}

func Test_strncmp(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "abc")
	writeString(20, "abd")

	if got := strncmp(10, 20, 2); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	if got := strncmp(10, 20, 3); got >= 0 {
		t.Errorf("got %v, want < 0", got)
	}
}

func Test_strspn(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello world")
	writeString(30, "helo ")

	if got, want := strspn(10, 30), len("hello "); got != sptr_t(want) {
		t.Errorf("got %v, want %d", got, want)
	}
}

func Test_strcspn(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello world")
	writeString(30, " ")
	writeString(40, "xyz")

	if got, want := strcspn(10, 30), len("hello"); got != sptr_t(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	if got, want := strcspn(10, 40), len("hello world"); got != sptr_t(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	for _, tt := range []struct {
		name, s, reject string
		want            sptr_t
	}{
		{"empty reject", "abc", "", 3},
		{"one reject", "abc", "b", 1},
		{"two reject first", "abc", "bx", 1},
		{"two reject second", "abc", "xb", 1},
		{"two reject absent", "abc", "xy", 3},
		{"duplicate reject", "abc", "bb", 1},
		{"high byte", "ab\xffc", "\xffx", 2},
		{"input NUL", "ab\x00c", "cx", 2},
		{"past scalar prefix", "abcdefghijklmnopq", "qz", 16},
		{"absent past scalar prefix", "abcdefghijklmnopq", "xz", 17},
		{"long reject", "abc", "xyzb", 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			memory = make([]byte, 1024)
			writeString(10, tt.s)
			writeString(400, tt.reject)
			if got := strcspn(10, 400); got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
