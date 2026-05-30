package main

import (
	"bytes"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func Test_dwarfline(t *testing.T) {
	src, err := os.ReadFile("testdata/dwarfline/dwarfline.c")
	if err != nil {
		t.Fatal(err)
	}

	// Find expected line numbers.
	want := map[int]bool{}
	for i, line := range strings.Split(string(src), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "extern") {
			continue
		}
		if strings.Contains(line, "_sink(") || strings.Contains(line, "_source(") {
			want[i+1] = true
		}
	}
	if len(want) == 0 {
		t.Fatal("no import call lines found in dwarfline.c")
	}

	*dwarfline = true
	t.Cleanup(func() { *dwarfline = false })

	in, err := os.Open("testdata/dwarfline/dwarfline.wasm")
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()

	var out bytes.Buffer
	if err := translate(in, &out); err != nil {
		t.Fatal(err)
	}

	// Scan the output for /*line FILE:N:M*/ annotations on matching lines.
	got := map[int]bool{}
	curFile := ""
	lineRe := regexp.MustCompile(`^/\*line ([^:]*):(\d+):\d+\*/`)
	for line := range strings.Lines(out.String()) {
		m := lineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if m[1] != "" {
			curFile = m[1]
		}
		if !strings.HasSuffix(curFile, "dwarfline.c") {
			continue
		}
		code := line[len(m[0]):]
		if strings.Contains(code, "_sink(") || strings.Contains(code, "_source(") {
			n, _ := strconv.Atoi(m[2])
			got[n] = true
		}
	}

	for line := range want {
		if !got[line] {
			t.Errorf("c line %d not found in -dwarfline output", line)
		}
	}
	for line := range got {
		if !want[line] {
			t.Errorf("unexpected c line %d in -dwarfline output", line)
		}
	}
}
