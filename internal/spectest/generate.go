//go:build ignore

package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	wg20 = "https://github.com/WebAssembly/spec/archive/refs/tags/wg-2.0.tar.gz"
	wg30 = "https://github.com/WebAssembly/spec/archive/refs/tags/wg-3.0.tar.gz"
	wabt = "https://github.com/WebAssembly/wabt/releases/download/1.0.41/wabt-1.0.41-linux-x64.tar.gz"
)

func main() {
	log.SetFlags(0)

	chdir()
	download(wg20, wg20Files)
	download(wg30, wg30Files)
	install()
	generate()
}

func chdir() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("failed to get current file path")
	}
	if err := os.Chdir(filepath.Dir(filename)); err != nil {
		log.Fatalf("failed to change directory: %v", err)
	}
}

func install() {
	log.Printf("downloading %s...", wabt)
	resp, err := http.Get(wabt)
	if err != nil {
		log.Fatalf("failed to download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("bad status: %s", resp.Status)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		log.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	log.Print("extracting wast2json...")
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("tar read error: %v", err)
		}

		if filepath.Base(hdr.Name) == "wast2json" {
			f, err := os.OpenFile("wast2json", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
			if err != nil {
				log.Fatalf("file open error: %v", err)
			}
			defer f.Close()
			if _, err := io.Copy(f, tr); err != nil {
				log.Fatalf("file copy error: %v", err)
			}
			return
		}
	}
	log.Fatal("failed to extract wast2json")
}

func download(spec string, files set[string]) {
	log.Printf("downloading %s...", spec)
	resp, err := http.Get(spec)
	if err != nil {
		log.Fatalf("failed to download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("bad status: %s", resp.Status)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		log.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	log.Print("extracting wast files...")
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("tar read error: %v", err)
		}
		_, name, ok := strings.Cut(hdr.Name, "/test/core/")
		if !ok {
			continue
		}
		if !files.has(name) {
			if strings.HasSuffix(name, ".wast") {
				log.Printf("Skipping %s", name)
			}
			continue
		}
		files.del(name)
		target := filepath.Join(strings.TrimSuffix(name, ".wast"), filepath.Base(name))
		f, err := os.Create(target)
		if err != nil {
			log.Fatalf("file open error: %v", err)
		}
		defer f.Close()
		if _, err := io.Copy(f, tr); err != nil {
			log.Fatalf("file copy error: %v", err)
		}
	}
	if len(files) > 0 {
		log.Fatalf("failed to extract wast files: %v", files)
	}
}

func generate() {
	exe, err := filepath.Abs("wast2json")
	if err != nil {
		log.Fatalf("failed to resolve wast2json: %v", err)
	}

	err = filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasSuffix(path, "skip") {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) == ".wast" {
			wast2json(exe, path)
			processJSON(strings.TrimSuffix(path, ".wast") + ".json")
		}
		return nil
	})
	if err != nil {
		log.Fatalf("failed to walk directory: %v", err)
	}
}

func wast2json(exe, path string) {
	log.Printf("running wast2json on %s...", path)

	cmd := exec.Command(exe, filepath.Base(path),
		"--enable-extended-const",
		"--enable-tail-call",
		"--enable-threads")
	cmd.Dir = filepath.Dir(path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("failed to run wast2json on %s: %v", path, err)
	}
}

func processJSON(path string) {
	spec := parseJSON(path)

	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		log.Fatalf("failed to open root %s: %v", filepath.Dir(path), err)
	}
	defer root.Close()

	for _, cmd := range spec.Commands {
		if cmd.Filename == "" {
			continue
		}

		switch cmd.Type {
		case "assert_invalid", "assert_malformed":
			if err := root.Remove(cmd.Filename); err != nil && !os.IsNotExist(err) {
				log.Fatalf("failed to remove %s: %v", cmd.Filename, err)
			}

		case "module", "assert_unlinkable", "assert_uninstantiable":
			ext := filepath.Ext(cmd.Filename)
			if ext != ".wasm" {
				log.Fatalf("unexpected extension: %s", ext)
			}
			module := strings.TrimSuffix(cmd.Filename, ext)
			if err := root.MkdirAll(module, 0755); err != nil {
				log.Fatalf("failed to create dir %s: %v", module, err)
			}
			if err := root.Rename(cmd.Filename, filepath.Join(module, cmd.Filename)); err != nil {
				log.Fatalf("failed to move %s: %v", cmd.Filename, err)
			}

		default:
			log.Fatalf("unknown command: %s", cmd.Type)
		}
	}
}

func parseJSON(path string) *specTest {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("failed to open %s: %v", path, err)
	}
	defer f.Close()

	spec := new(specTest)
	if err := json.NewDecoder(f).Decode(spec); err != nil {
		log.Fatalf("failed to parse %s: %v", path, err)
	}
	return spec
}

type specTest struct {
	Commands []struct {
		Type     string `json:"type"`
		Filename string `json:"filename"`
	} `json:"commands"`
}

var wg20Files = set[string]{
	"align.wast":           {},
	"br_if.wast":           {},
	"br_table.wast":        {},
	"elem.wast":            {},
	"func.wast":            {},
	"global.wast":          {},
	"linking.wast":         {},
	"local_tee.wast":       {},
	"memory.wast":          {},
	"ref_is_null.wast":     {},
	"ref_null.wast":        {},
	"select.wast":          {},
	"table.wast":           {},
	"unreached-valid.wast": {},
}

var wg30Files = set[string]{
	"address.wast":              {},
	"block.wast":                {},
	"br.wast":                   {},
	"call.wast":                 {},
	"call_indirect.wast":        {},
	"conversions.wast":          {},
	"endianness.wast":           {},
	"f32.wast":                  {},
	"f32_cmp.wast":              {},
	"f32_bitwise.wast":          {},
	"f64.wast":                  {},
	"f64_cmp.wast":              {},
	"f64_bitwise.wast":          {},
	"fac.wast":                  {},
	"float_exprs.wast":          {},
	"float_literals.wast":       {},
	"float_memory.wast":         {},
	"float_misc.wast":           {},
	"forward.wast":              {},
	"func_ptrs.wast":            {},
	"i32.wast":                  {},
	"i64.wast":                  {},
	"if.wast":                   {},
	"int_exprs.wast":            {},
	"int_literals.wast":         {},
	"labels.wast":               {},
	"left-to-right.wast":        {},
	"load.wast":                 {},
	"local_get.wast":            {},
	"local_set.wast":            {},
	"loop.wast":                 {},
	"memory_grow.wast":          {},
	"memory_redundancy.wast":    {},
	"memory_size.wast":          {},
	"memory_trap.wast":          {},
	"names.wast":                {},
	"nop.wast":                  {},
	"ref_func.wast":             {},
	"return.wast":               {},
	"return_call.wast":          {},
	"return_call_indirect.wast": {},
	"stack.wast":                {},
	"start.wast":                {},
	"store.wast":                {},
	"switch.wast":               {},
	"table_get.wast":            {},
	"table_grow.wast":           {},
	"table_set.wast":            {},
	"table_size.wast":           {},
	"traps.wast":                {},
	"unreachable.wast":          {},
	"unwind.wast":               {},

	"bulk-memory/bulk.wast":        {},
	"bulk-memory/memory_copy.wast": {},
	"bulk-memory/memory_fill.wast": {},
	"bulk-memory/memory_init.wast": {},
	"bulk-memory/table_copy.wast":  {},
	"bulk-memory/table_fill.wast":  {},
	"bulk-memory/table_init.wast":  {},

	"memory64/address64.wast":           {},
	"memory64/bulk64.wast":              {},
	"memory64/call_indirect64.wast":     {},
	"memory64/endianness64.wast":        {},
	"memory64/float_memory64.wast":      {},
	"memory64/load64.wast":              {},
	"memory64/memory_copy64.wast":       {},
	"memory64/memory_fill64.wast":       {},
	"memory64/memory_grow64.wast":       {},
	"memory64/memory_init64.wast":       {},
	"memory64/memory_redundancy64.wast": {},
	"memory64/memory_trap64.wast":       {},
	"memory64/table_copy_mixed.wast":    {},
	"memory64/table_copy64.wast":        {},
	"memory64/table_fill64.wast":        {},
	"memory64/table_get64.wast":         {},
	"memory64/table_grow64.wast":        {},
	"memory64/table_init64.wast":        {},
	"memory64/table_set64.wast":         {},
	"memory64/table_size64.wast":        {},
}

type set[T comparable] map[T]struct{}

func (s set[T]) add(t T) bool {
	if _, ok := s[t]; ok {
		return false
	}
	s[t] = struct{}{}
	return true
}

func (s set[T]) del(t T) bool {
	if _, ok := s[t]; ok {
		delete(s, t)
		return true
	}
	return false
}

func (s set[T]) has(t T) bool {
	_, ok := s[t]
	return ok
}
