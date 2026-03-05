package main

import (
	"bytes"
	"go/format"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_translate(t *testing.T) {
	tests := []string{"fib", "memory", "primes", "recursion", "stack", "table", "trig"}
	*nanbox = false
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			path := "testdata/" + name + "/" + name

			in, err := os.Open(path + ".wasm")
			if err != nil {
				t.Fatal(err)
			}
			defer in.Close()

			var out bytes.Buffer
			err = translate(in, &out)
			if err != nil {
				t.Fatal(err)
			}

			err = os.WriteFile(path+".go", out.Bytes(), 0644)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func Test_translateSpecTest(t *testing.T) {
	*nanbox = true
	filepath.WalkDir("spectest/", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if ext := filepath.Ext(path); ext == ".wasm" {
			in, err := os.Open(path)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				return nil
			}
			defer in.Close()

			var out bytes.Buffer
			err = translate(in, &out)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				return nil
			}

			err = os.WriteFile(strings.TrimRight(path, ext)+".go", out.Bytes(), 0644)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				return nil
			}

			err = generateSpecTest(path)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				return nil
			}
		}
		return nil
	})
}

func generateSpecTest(path string) error {
	dir := filepath.Dir(path)
	baseDir := filepath.Base(dir)
	wasmFile := filepath.Base(path)
	jsonFile := baseDir + ".json"
	testFile := filepath.Join(dir, baseDir+"_test.go")

	type specFileInfo struct {
		ImportPath string
		JSONFile   string
		WasmFile   string
	}

	info := specFileInfo{
		ImportPath: "github.com/ncruces/wasm2go/" + dir,
		JSONFile:   jsonFile,
		WasmFile:   wasmFile,
	}

	tmpl, err := template.New("spectest").Parse(specTestTemplate)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, info); err != nil {
		return err
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	return os.WriteFile(testFile, src, 0644)
}

const specTestTemplate = `package wasm2go

import (
	_ "embed"
	"testing"

	"github.com/ncruces/wasm2go/spectest"
)

//go:embed {{.JSONFile}}
var data []byte

func Test(t *testing.T) {
	spectest.Test(t, New(), data, "{{.WasmFile}}")
}
`
