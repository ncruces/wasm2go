package libc

import "testing"

func writeString(p ptr, s string) {
	copy(memory[uptr(p):], s)
	memory[uptr(p)+uptr(len(s))] = 0
}

func Test_memchr(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello world")

	if got := memchr(10, 'w', ptr(len("hello world"))); got != 16 {
		t.Errorf("got %v, want 16", got)
	}
	if got := memchr(10, 'z', ptr(len("hello world"))); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	if got := memchr(10, 'w', ptr(len("hello"))); got != 0 {
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

func Test_strlen(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello")
	writeString(20, "")

	if got, want := strlen(10), len("hello"); got != ptr(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	if got := strlen(20); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func Test_strchr(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello")

	if got, want := strchr(10, 'l'), 10+2; got != ptr(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	if got := strchr(10, 'z'); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	if got, want := strchr(10, 0), 10+len("hello"); got != ptr(want) {
		t.Errorf("got %v, want %d", got, want)
	}
}

func Test_strchrnul(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello")

	if got, want := strchrnul(10, 'l'), 10+2; got != ptr(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	if got, want := strchrnul(10, 'z'), 10+len("hello"); got != ptr(want) {
		t.Errorf("got %v, want %d", got, want)
	}
}

func Test_strrchr(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello")

	if got, want := strrchr(10, 'l'), 10+3; got != ptr(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	if got := strrchr(10, 'z'); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
	if got, want := strrchr(10, 0), 10+len("hello"); got != ptr(want) {
		t.Errorf("got %v, want %d", got, want)
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

	if got, want := strspn(10, 30), len("hello "); got != ptr(want) {
		t.Errorf("got %v, want %d", got, want)
	}
}

func Test_strcspn(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello world")
	writeString(30, " ")
	writeString(40, "xyz")

	if got, want := strcspn(10, 30), len("hello"); got != ptr(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	if got, want := strcspn(10, 40), len("hello world"); got != ptr(want) {
		t.Errorf("got %v, want %d", got, want)
	}
}

func Test_strstr(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello world")
	writeString(30, "world")
	writeString(40, "z")
	writeString(50, "")

	if got, want := strstr(10, 30), 10+len("hello "); got != ptr(want) {
		t.Errorf("got %v, want %d", got, want)
	}
	if got := strstr(10, 50); got != 10 {
		t.Errorf("got %v, want 10", got)
	}
	if got := strstr(10, 40); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func Test_strcpy(t *testing.T) {
	memory = make([]byte, 1024)
	writeString(10, "hello")

	if got := strcpy(20, 10); got != 20 {
		t.Errorf("got %v, want 20", got)
	}
	if got := strcmp(10, 20); got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}
