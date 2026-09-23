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
	want := set[int]{}
	for i, line := range bytes.Split(src, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if bytes.HasPrefix(trimmed, []byte("extern")) {
			continue
		}
		if bytes.Contains(line, []byte("_sink(")) ||
			bytes.Contains(line, []byte("_source(")) {
			want.add(i + 1)
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
	got := set[int]{}
	var curFile string
	lineRe := regexp.MustCompile(`^/\*line ([^:]*):(\d+)(:\d+)?\*/`)
	for line := range bytes.Lines(out.Bytes()) {
		m := lineRe.FindSubmatch(line)
		if m == nil {
			continue
		}
		if len(m[1]) != 0 {
			curFile = string(m[1])
		}
		if !strings.HasSuffix(curFile, "dwarfline.c") {
			continue
		}
		code := line[len(m[0]):]
		if bytes.Contains(code, []byte("_sink(")) ||
			bytes.Contains(code, []byte("_source(")) {
			n, _ := strconv.Atoi(string(m[2]))
			got.add(n)
		}
	}

	for line := range want {
		if !got.has(line) {
			t.Errorf("c line %d not found in -dwarfline output", line)
		}
	}
	for line := range got {
		if !want.has(line) {
			t.Errorf("unexpected c line %d in -dwarfline output", line)
		}
	}
}
