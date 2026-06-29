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
	wg10 = "https://github.com/WebAssembly/spec/archive/refs/tags/wg-1.0.tar.gz"
	wg20 = "https://github.com/WebAssembly/spec/archive/refs/tags/wg-2.0.tar.gz"
	wg30 = "https://github.com/WebAssembly/spec/archive/refs/tags/wg-3.0.tar.gz"
	wabt = "https://github.com/WebAssembly/wabt/releases/download/1.0.41/wabt-1.0.41-linux-x64.tar.gz"
	wsmt = "https://github.com/bytecodealliance/wasm-tools/releases/download/v1.252.0/wasm-tools-1.252.0-x86_64-linux.tar.gz"
)

func main() {
	log.SetFlags(0)

	chdir()
	download(wg10, 10)
	download(wg20, 20)
	download(wg30, 30)
	for f, i := range files {
		if i > 0 {
			log.Fatalf("missing wast file: %s", f)
		}
	}
	install("wast2json", wabt)
	install("wasm-tools", wsmt)
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

func install(file, url string) {
	log.Printf("downloading %s...", url)
	resp, err := http.Get(url)
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

	log.Printf("extracting %s...", file)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("tar read error: %v", err)
		}

		if filepath.Base(hdr.Name) == file {
			f, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
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
	log.Fatalf("failed to extract %s", file)
}

func download(spec string, version byte) {
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
		if v, ok := files[name]; v != version {
			if !ok && strings.HasSuffix(name, ".wast") {
				log.Printf("skipping %s", name)
			}
			continue
		}
		files[name] = 0
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
}

func generate() {
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
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
			json := strings.TrimSuffix(path, ".wast") + ".json"
			// wasm_tools(path, json)
			wast2json(path)
			processJSON(json)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("failed to walk directory: %v", err)
	}
}

func wast2json(path string) {
	exe, err := filepath.Abs("wast2json")
	if err != nil {
		log.Fatalf("failed to resolve wast2json: %v", err)
	}

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

func wasm_tools(wast, json string) {
	exe, err := filepath.Abs("wasm-tools")
	if err != nil {
		log.Fatalf("failed to resolve wasm-tools: %v", err)
	}

	log.Printf("running wasm-tools on %s...", wast)

	cmd := exec.Command(exe, "wast2json",
		"-o", filepath.Base(json), filepath.Base(wast))
	cmd.Dir = filepath.Dir(wast)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("failed to run wasm-tools on %s: %v", wast, err)
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
			if cmd.Binary == "" {
				break
			}
			if err := root.Remove(cmd.Binary); err != nil && !os.IsNotExist(err) {
				log.Fatalf("failed to remove %s: %v", cmd.Filename, err)
			}

		case "module", "assert_unlinkable", "assert_uninstantiable":
			ext := filepath.Ext(cmd.Filename)
			if ext == ".wat" && cmd.Binary != "" {
				if err := root.Remove(cmd.Filename); err != nil && !os.IsNotExist(err) {
					log.Fatalf("failed to remove %s: %v", cmd.Filename, err)
				}
				ext = filepath.Ext(cmd.Binary)
				cmd.Filename = cmd.Binary
			}
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
		Binary   string `json:"binary_filename"`
	} `json:"commands"`
}

var files = map[string]byte{
	"break-drop.wast": 10,
	"globals.wast":    10,

	"align.wast":             20,
	"br_if.wast":             20,
	"br_table.wast":          20,
	"elem.wast":              20,
	"data.wast":              20,
	"func.wast":              20,
	"global.wast":            20,
	"linking.wast":           20,
	"local_tee.wast":         20,
	"memory.wast":            20,
	"ref_is_null.wast":       20,
	"ref_null.wast":          20,
	"select.wast":            20,
	"table.wast":             20,
	"unreached-valid.wast":   20,
	"unreached-invalid.wast": 20,

	"address.wast":              30,
	"block.wast":                30,
	"br.wast":                   30,
	"call.wast":                 30,
	"call_indirect.wast":        30,
	"comments.wast":             30,
	"conversions.wast":          30,
	"const.wast":                30,
	"custom.wast":               30,
	"endianness.wast":           30,
	"f32.wast":                  30,
	"f32_cmp.wast":              30,
	"f32_bitwise.wast":          30,
	"f64.wast":                  30,
	"f64_cmp.wast":              30,
	"f64_bitwise.wast":          30,
	"fac.wast":                  30,
	"float_exprs.wast":          30,
	"float_literals.wast":       30,
	"float_memory.wast":         30,
	"float_misc.wast":           30,
	"forward.wast":              30,
	"func_ptrs.wast":            30,
	"i32.wast":                  30,
	"i64.wast":                  30,
	"if.wast":                   30,
	"int_exprs.wast":            30,
	"int_literals.wast":         30,
	"labels.wast":               30,
	"left-to-right.wast":        30,
	"load.wast":                 30,
	"local_get.wast":            30,
	"local_set.wast":            30,
	"loop.wast":                 30,
	"memory_grow.wast":          30,
	"memory_redundancy.wast":    30,
	"memory_size.wast":          30,
	"memory_trap.wast":          30,
	"names.wast":                30,
	"nop.wast":                  30,
	"ref_func.wast":             30,
	"return.wast":               30,
	"return_call.wast":          30,
	"return_call_indirect.wast": 30,
	"stack.wast":                30,
	"start.wast":                30,
	"store.wast":                30,
	"switch.wast":               30,
	"table_get.wast":            30,
	"table_grow.wast":           30,
	"table_set.wast":            30,
	"table_size.wast":           30,
	"token.wast":                30,
	"traps.wast":                30,
	"type.wast":                 30,
	"unreachable.wast":          30,
	"unwind.wast":               30,

	"bulk-memory/bulk.wast":        30,
	"bulk-memory/memory_copy.wast": 30,
	"bulk-memory/memory_fill.wast": 30,
	"bulk-memory/memory_init.wast": 30,
	"bulk-memory/table_copy.wast":  30,
	"bulk-memory/table_fill.wast":  30,
	"bulk-memory/table_init.wast":  30,

	"memory64/address64.wast":           30,
	"memory64/bulk64.wast":              30,
	"memory64/call_indirect64.wast":     30,
	"memory64/endianness64.wast":        30,
	"memory64/float_memory64.wast":      30,
	"memory64/load64.wast":              30,
	"memory64/memory_copy64.wast":       30,
	"memory64/memory_fill64.wast":       30,
	"memory64/memory_grow64.wast":       30,
	"memory64/memory_init64.wast":       30,
	"memory64/memory_redundancy64.wast": 30,
	"memory64/memory_trap64.wast":       30,
	"memory64/table_copy_mixed.wast":    30,
	"memory64/table_copy64.wast":        30,
	"memory64/table_fill64.wast":        30,
	"memory64/table_get64.wast":         30,
	"memory64/table_grow64.wast":        30,
	"memory64/table_init64.wast":        30,
	"memory64/table_set64.wast":         30,
	"memory64/table_size64.wast":        30,
}
