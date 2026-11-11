package main

import (
	"bytes"
	"embed"
	"encoding/binary"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"maps"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"

	"golang.org/x/tools/go/ast/astutil"

	"github.com/ncruces/wasm2go/internal/mangle"
)

//go:embed c go
var src embed.FS

var (
	output  = flag.String("o", "", "output file (default stdout)")
	wasm    = flag.String("wasm", "", "input.wasm file")
	pkg     = flag.String("pkg", "", "package name (default wasm2go)")
	m64     = flag.Bool("m64", false, "use 64-bit pointers (int64)")
	deref   = flag.Bool("deref-mem", false, "dereference memory (*m.memory instead of m.memory)")
	cout    = flag.String("c-out", "", "extract libc C source and header files to directory")
	version = flag.Bool("version", false, "print version and exit")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [option]... [func]...\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	if *version {
		fmt.Fprintln(flag.CommandLine.Output(), filepath.Base(os.Args[0]), getVersion())
		os.Exit(0)
	}

	if *cout != "" {
		s, err := fs.Sub(src, "c")
		if err != nil {
			log.Fatal(err)
		}
		err = os.RemoveAll(*cout)
		if err != nil {
			log.Fatal(err)
		}
		err = os.CopyFS(*cout, s)
		if err != nil {
			log.Fatal(err)
		}
	}

	funcs := flag.Args()
	if *wasm != "" {
		funcs = readWasm()
	}
	if *pkg == "" {
		*pkg = "wasm2go"
	}
	if len(funcs) == 0 {
		return
	}

	sptrType := "int32"
	uptrType := "uint32"
	invokeTypes['p'] = "int32"
	if *m64 {
		sptrType = "int64"
		uptrType = "uint64"
		invokeTypes['p'] = "int64"
	}

	fset := token.NewFileSet()
	entries, err := src.ReadDir("go")
	if err != nil {
		log.Fatal(err)
	}

	allImports := make(map[string]string)
	available := make(map[string]*ast.FuncDecl)

	// Map out the functions and track module paths for automatic import resolution
	for _, entry := range entries {
		name := entry.Name()
		if name == "libc.go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := src.ReadFile("go/" + name)
		if err != nil {
			log.Fatal(err)
		}
		f, err := parser.ParseFile(fset, name, data, 0)
		if err != nil {
			log.Fatal(err)
		}

		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			name := path
			if idx := strings.LastIndex(name, "/"); idx >= 0 {
				name = name[idx+1:]
			}
			if imp.Name != nil {
				name = imp.Name.Name
			}
			allImports[name] = path
		}

		for _, decl := range f.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok {
				available[fd.Name.Name] = fd
			}
		}
	}

	invokeSigs := set[string]{}
	for i, name := range funcs {
		if sig, ok := parseInvokeSig(name); ok {
			invokeSigs.add(sig)
			funcs[i] = "_catch_longjmp"
		}
	}

	// Tree-shake the AST
	var keep []*ast.FuncDecl
	visited := make(set[string])
	for _, name := range funcs {
		if available[name] == nil && *wasm == "" {
			log.Fatalf("function %s not found in templates", name)
		}
		keep = appendDeps(keep, name, available, visited)
	}

	usedImports := make(set[string])

	// Execute the rewriting permutations
	for _, fd := range keep {
		fd.Name = mangle.ID(fd.Name.Name, mangle.Internal)

		astutil.Apply(fd, nil, func(c *astutil.Cursor) bool {
			if call, ok := c.Node().(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "cptr_t" {
					if *m64 {
						id.Name = "p64"
					} else {
						id.Name = uptrType
					}
				}
			}
			return true
		})

		var recv bool
		ast.Inspect(fd, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SliceExpr:
				if id, ok := x.X.(*ast.Ident); ok && id.Name == "memory" {
					x.X = makeMemoryExpr(*deref)
				}
			case *ast.IndexExpr:
				if id, ok := x.X.(*ast.Ident); ok && id.Name == "memory" {
					x.X = makeMemoryExpr(*deref)
				}
			case *ast.CallExpr:
				if id, ok := x.Fun.(*ast.Ident); ok && available[id.Name] != nil {
					callee := available[id.Name]
					if callee.Recv != nil {
						x.Fun = &ast.SelectorExpr{X: ast.NewIdent("m"), Sel: callee.Name}
					} else {
						x.Fun = callee.Name
					}
				}
				for i, arg := range x.Args {
					if id, ok := arg.(*ast.Ident); ok && id.Name == "memory" {
						x.Args[i] = makeMemoryExpr(*deref)
					}
				}
			case *ast.Ident:
				switch x.Name {
				case "sptr_t":
					x.Name = sptrType
				case "uptr_t", "cptr_t":
					x.Name = uptrType
				}
			case *ast.SelectorExpr:
				if id, ok := x.X.(*ast.Ident); ok {
					if id.Name == "m" {
						recv = true
					}
					if path, exists := allImports[id.Name]; exists {
						usedImports.add(path)
					}
				}
			}
			return true
		})

		if recv {
			fd.Recv = &ast.FieldList{
				List: []*ast.Field{{
					Names: []*ast.Ident{ast.NewIdent("m")},
					Type:  &ast.StarExpr{X: ast.NewIdent("Module")},
				}},
			}
		}
	}

	// Write the unified file
	outAST := &ast.File{
		Name: ast.NewIdent(*pkg),
	}

	if len(usedImports) > 0 {
		var specs []ast.Spec
		for _, imp := range slices.Sorted(maps.Keys(usedImports)) {
			specs = append(specs, &ast.ImportSpec{
				Path: &ast.BasicLit{Kind: token.STRING, Value: `"` + imp + `"`},
			})
		}
		outAST.Decls = append(outAST.Decls, &ast.GenDecl{Tok: token.IMPORT, Specs: specs})
	}

	for _, fd := range keep {
		outAST.Decls = append(outAST.Decls, fd)
	}

	for _, sig := range slices.Sorted(maps.Keys(invokeSigs)) {
		outAST.Decls = append(outAST.Decls, generateInvoke(fset, sig))
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated by libc-gen %s. DO NOT EDIT.\n\n", getVersion())
	if err := format.Node(&buf, fset, outAST); err != nil {
		log.Fatal(err)
	}

	outData := buf.Bytes()
	if *output == "" {
		os.Stdout.Write(outData)
	} else {
		if err := os.WriteFile(*output, outData, 0644); err != nil {
			log.Fatal(err)
		}
	}
}

func appendDeps(keep []*ast.FuncDecl, name string, available map[string]*ast.FuncDecl, visited set[string]) []*ast.FuncDecl {
	fd := available[name]
	if fd == nil || !visited.add(name) {
		return keep
	}
	ast.Inspect(fd, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok {
				keep = appendDeps(keep, id.Name, available, visited)
			}
		}
		return true
	})
	return append(keep, fd)
}

func makeMemoryExpr(deref bool) ast.Expr {
	sel := &ast.SelectorExpr{X: ast.NewIdent("m"), Sel: ast.NewIdent("memory")}
	if deref {
		return &ast.ParenExpr{X: &ast.StarExpr{X: sel}}
	}
	return sel
}

func readWasm() (funcs []string) {
	b, err := os.ReadFile(*wasm)
	if err != nil {
		log.Fatal(err)
	}

	b = readHeader(b)
	for len(b) > 0 {
		id := b[0]
		size, rest := readLEB128(b[1:])
		content := rest[:size]
		b = rest[size:]

		switch id {
		case 0: // custom
			if *pkg != "" {
				break
			}
			name, sec := readString(content)
			if name == "name" {
				for len(sec) > 0 {
					subId := sec[0]
					var subSize uint64
					subSize, sec = readLEB128(sec[1:])
					subContent := sec[:subSize]
					sec = sec[subSize:]
					if subId == 0 {
						name, _ := readString(subContent)
						*pkg = mangle.Name(name, mangle.Local)
					}
				}
			}
		case 2: // import
			count, content := readLEB128(content)
			for i := 0; i < int(count); i++ {
				var mod, name string
				mod, content = readString(content)
				name, content = readString(content)
				kind := content[0]
				content = content[1:]
				switch kind {
				case 0: // func
					_, content = readLEB128(content)
					if mod == "env" {
						funcs = append(funcs, name)
					}
				case 1: // table
					var flags uint64
					_ = content[0]
					flags, content = readLEB128(content[1:])
					_, content = readLEB128(content)
					if flags&1 != 0 {
						_, content = readLEB128(content)
					}
				case 2: // memory
					var flags uint64
					flags, content = readLEB128(content)
					_, content = readLEB128(content)
					if flags&1 != 0 {
						_, content = readLEB128(content)
					}
					*deref = true
					if flags&4 != 0 {
						*m64 = true
					}
				case 3: // global
					content = content[2:]
				}
			}
		case 5: // memory
			count, content := readLEB128(content)
			for i := 0; i < int(count); i++ {
				var flags uint64
				flags, content = readLEB128(content)
				_, content = readLEB128(content)
				if flags&1 != 0 {
					_, content = readLEB128(content)
				}
				if flags&2 != 0 {
					*deref = true
				}
				if flags&4 != 0 {
					*m64 = true
				}
			}
		}
	}

	slices.Sort(funcs)
	return slices.Compact(funcs)
}

func readHeader(b []byte) []byte {
	if len(b) < 8 {
		log.Fatal("invalid wasm file")
	}
	if magic := string(b[:4]); magic != "\x00asm" {
		log.Fatalf("invalid magic number: %q", magic)
	}
	if version := binary.LittleEndian.Uint32(b[4:]); version != 1 {
		log.Fatalf("invalid version: %d", version)
	}
	return b[8:]
}

func readLEB128(b []byte) (uint64, []byte) {
	var x uint64
	var s uint
	for i, v := range b {
		x |= uint64(v&0x7f) << s
		s += 7
		if v <= 0x7f {
			return x, b[i+1:]
		}
	}
	return 0, nil
}

func readString(b []byte) (string, []byte) {
	l, b := readLEB128(b)
	return string(b[:l]), b[l:]
}

var invokeTypes = map[byte]string{
	'v': "",
	'i': "int32",
	'j': "int64",
	'f': "float32",
	'd': "float64",
	'p': "ptr",
	'e': "any",
}

func parseInvokeSig(name string) (sig string, ok bool) {
	sig, ok = strings.CutPrefix(name, "invoke_")
	if !ok {
		return "", false
	}
	for _, c := range sig {
		_, ok = invokeTypes[byte(c)]
		if !ok {
			return "", false
		}
	}
	return sig, true
}

func generateInvoke(fset *token.FileSet, sig string) *ast.FuncDecl {
	var retKeyword string
	retType := invokeTypes[sig[0]]
	if retType != "" {
		retKeyword = "return"
	}

	var callArgs, tableParams []string
	params := []string{"idx " + invokeTypes['p']}

	for i := 1; i < len(sig); i++ {
		t := invokeTypes[sig[i]]
		name := fmt.Sprintf("v%d", i)
		params = append(params, name+" "+t)
		callArgs = append(callArgs, name)
		tableParams = append(tableParams, t)
	}

	src := fmt.Sprintf(`
		package p
		func (m *Module) _invoke_%s(%s)%s {
			defer m.__catch_longjmp(*m.X__stack_pointer())
			%s (*m.X__indirect_function_table())[idx].(func(%s)%s)(%s)
		}`,
		sig, strings.Join(params, ","), retType,
		retKeyword, strings.Join(tableParams, ","), retType, strings.Join(callArgs, ","))

	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		log.Fatal(err)
	}
	return f.Decls[0].(*ast.FuncDecl)
}

func getVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}
	return "(unknown)"
}
