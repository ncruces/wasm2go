package libc

import (
	"strings"
	"testing"
)

var strcspnBenchResult sptr_t

func BenchmarkStrcspn(b *testing.B) {
	for _, tc := range []struct {
		name, input, reject string
		want                sptr_t
	}{
		{"one/short-absent", strings.Repeat("a", 12), "z", 12},
		{"one/long-absent", strings.Repeat("a", 256), "z", 256},
		{"two/long-absent", strings.Repeat("a", 256), "zZ", 256},
		{"one/long-early", "z" + strings.Repeat("a", 255), "z", 0},
		{"two/long-late", strings.Repeat("a", 255) + "Z", "zZ", 255},
		{"three/long-absent", strings.Repeat("a", 256), "zZx", 256},
	} {
		b.Run(tc.name, func(b *testing.B) {
			memory = make([]byte, 1024)
			writeString(10, tc.input)
			writeString(400, tc.reject)
			b.ReportAllocs()
			b.ResetTimer()
			var got sptr_t
			for i := 0; i < b.N; i++ {
				got = strcspn(10, 400)
			}
			strcspnBenchResult = got
			if got != tc.want {
				b.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}
