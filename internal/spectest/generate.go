//go:build generator

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
	"slices"
	"strings"
)

const wabt = "https://github.com/WebAssembly/wabt/releases/download/1.0.41/wabt-1.0.41-linux-x64.tar.gz"

var skipModules = []string{
	"bulk.5", // data.drop not supported
	"elem.58",
	"elem.60",
	"elem.61",
	"elem.67",
	"func_ptrs.0",
	"global.0",
	"linking.1",
	"linking.16",
	"linking.17",
	"linking.20",
	"linking.30",
	"linking.31",
	"linking.34",
	"linking.38", // needs linking.39
	"linking.6",
	"memory_grow.6",
	"memory_grow.7",
	"names.3",
	"ref_func.1",
	"return_call.0",
	"return_call_indirect.0",
	"table_copy.1",
	"table_copy.10",
	"table_copy.11",
	"table_copy.12",
	"table_copy.13",
	"table_copy.14",
	"table_copy.15",
	"table_copy.16",
	"table_copy.17",
	"table_copy.18",
	"table_copy.2",
	"table_copy.3",
	"table_copy.4",
	"table_copy.5",
	"table_copy.6",
	"table_copy.7",
	"table_copy.8",
	"table_copy.9",
	"table_grow.6",
	"table_grow.7",
	"table_init.1",
	"table_init.2",
	"table_init.3",
	"table_init.4",
	"table_init.5",
	"table_init.6",
}

func main() {
	log.SetFlags(0)

	chdir()
	download()
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

func download() {
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

	log.Println("extracting wast2json...")
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("tar read error: %v", err)
		}

		if filepath.Base(hdr.Name) == "wast2json" && hdr.Typeflag == tar.TypeReg {
			f, err := os.OpenFile("wast2json", os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(hdr.Mode))
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
	log.Fatalf("failed to extract wast2json")
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
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read %s: %v", path, err)
	}

	var spec struct {
		Commands []struct {
			Type     string `json:"type"`
			Filename string `json:"filename"`
		} `json:"commands"`
	}

	if err := json.Unmarshal(data, &spec); err != nil {
		log.Fatalf("failed to parse %s: %v", path, err)
	}

	dir := filepath.Dir(path)

	var currentFolder, currentBase string
	copied := make(set[string])

	for _, cmd := range spec.Commands {
		switch cmd.Type {
		case "assert_invalid", "assert_malformed", "assert_unlinkable", "assert_uninstantiable":
			if cmd.Filename == "" {
				continue
			}
			path := filepath.Join(dir, cmd.Filename)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				log.Fatalf("failed to remove %s: %v", path, err)
			}
		case "module":
			if cmd.Filename == "" {
				continue
			}
			currentBase = strings.TrimSuffix(cmd.Filename, filepath.Ext(cmd.Filename))
			currentFolder = filepath.Join(dir, currentBase)
			if err := os.MkdirAll(currentFolder, 0755); err != nil {
				log.Fatalf("failed to create dir %s: %v", currentFolder, err)
			}
			path := filepath.Join(dir, cmd.Filename)
			if err := os.Rename(path, filepath.Join(currentFolder, cmd.Filename)); err != nil {
				log.Fatalf("failed to move %s: %v", path, err)
			}
		case "action", "assert_return", "assert_trap":
			if currentFolder == "" || copied.has(currentFolder) {
				continue
			}
			if _, skip := slices.BinarySearch(skipModules, currentBase); skip {
				continue
			}
			copied.add(currentFolder)

			dest := filepath.Join(currentFolder, currentBase+".json")
			os.Remove(dest)
			if err := os.Link(path, dest); err != nil {
				log.Fatalf("failed to link json to %s: %v", dest, err)
			}
		}
	}

	if err := os.Remove(path); err != nil {
		log.Fatalf("failed to remove original json %s: %v", path, err)
	}
}

type set[T comparable] map[T]struct{}

func (s set[T]) add(t T) bool {
	if _, ok := s[t]; ok {
		return false
	}
	s[t] = struct{}{}
	return true
}

func (s set[T]) has(t T) bool {
	_, ok := s[t]
	return ok
}
