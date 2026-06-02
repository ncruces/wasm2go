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

const wabt = "https://github.com/WebAssembly/wabt/releases/download/1.0.41/wabt-1.0.41-linux-x64.tar.gz"

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
