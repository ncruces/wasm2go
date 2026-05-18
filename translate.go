package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/ncruces/wasm2go/internal/util"
)

var (
	//go:embed helpers/helpers.go
	helpersSrc string
	//go:embed helpers/helpers_unsafe.go
	helpersUnsafeSrc string
	//go:embed helpers/atomics_unsafe.go
	helpersAtomicsSrc string
	//go:embed helpers/cpuarch_unsafe.go
	helpersCpuArchSrc string
)

// These helpers can never trap.
var pureHelpers = set[string]{
	"i32_shl": {}, "i32_shr_s": {}, "i32_shr_u": {}, "i32_rotl": {}, "i32_rotr": {},
	"i64_shl": {}, "i64_shr_s": {}, "i64_shr_u": {}, "i64_rotl": {}, "i64_rotr": {},
	"f32_abs": {}, "f32_copysign": {}, "f32_min": {}, "f32_max": {},
	"f64_min": {}, "f64_max": {}, "min": {}, "max": {},
	"i32_trunc_sat_f32_s": {}, "i32_trunc_sat_f32_u": {},
	"i32_trunc_sat_f64_s": {}, "i32_trunc_sat_f64_u": {},
	"i64_trunc_sat_f32_s": {}, "i64_trunc_sat_f32_u": {},
	"i64_trunc_sat_f64_s": {}, "i64_trunc_sat_f64_u": {},
}

// Standard library packages used by generated code.
var stdlib = map[string]string{
	"list":    "container/list",
	"binary":  "encoding/binary",
	"math":    "math",
	"bits":    "math/bits",
	"runtime": "runtime",
	"sync":    "sync",
	"atomic":  "sync/atomic",
	"time":    "time",
	"unsafe":  "unsafe",
}

type translator struct {
	in  *bufio.Reader
	out ast.File
	// Dependencies.
	packages set[string]
	provided set[string]
	helpers  set[string]
	// Sections.
	types     []funcType
	imports   []importDef
	functions []funcCompiler
	memory    *memoryDef
	tables    []tableDef
	globals   []globalDef
	exports   map[string]export
	elements  []elemSegment
	start     uint64
	data      []dataSegment
	dylink    *dylinkDef
}

func translate(r io.Reader, w io.Writer) error {
	var t translator

	t.in = bufio.NewReader(r)
	err := readHeader(t.in)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	t.packages = set[string]{}
	t.provided = set[string]{}
	t.helpers = set[string]{}

	for _, file := range provided {
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range f.Decls {
			// Check if the receiver type is *Module.
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv != nil {
				if star, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
					if id, ok := star.X.(*ast.Ident); ok && id.Name == "Module" {
						t.provided.add(fn.Name.Name)
					}
				}
			}
		}
	}

	// Load Wasm.
	for {
		if err := t.readSection(); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	exported := false
	for _, exp := range t.exports {
		if exp.kind == externMemory {
			exported = true
			break
		}
	}
	if t.memory != nil && (exported || t.memory.imported) {
		t.out.Decls = append(t.createMemoryTypes(), t.out.Decls...)
	}
	if !*nohost && len(t.imports) > 0 {
		t.out.Decls = append(t.createHostInterfaces(), t.out.Decls...)
	}
	if t.dylink != nil {
		t.out.Decls = append(t.out.Decls, t.createDylinkConstants())
	}

	t.out.Decls = append([]ast.Decl{
		t.createModuleStruct(),
		t.createNewFunc()},
		t.out.Decls...)

	// Fill in missing names.
	if t.out.Name == nil {
		t.out.Name = newID("wasm2go")
	}
	for i, fn := range t.functions {
		if fn.decl != nil && fn.decl.Name.Name == "" {
			fn.decl.Name.Name = "fn" + strconv.Itoa(i)
		}
	}
	if t.memory != nil && t.memory.id.Name == "" {
		t.memory.id.Name = "memory"
	}
	for i, tb := range t.tables {
		if tb.id.Name == "" {
			tb.id.Name = "t" + strconv.Itoa(i)
		}
	}
	for i, gv := range t.globals {
		if gv.id.Name == "" {
			gv.id.Name = "g" + strconv.Itoa(i)
		}
	}

	t.out.Decls = append(t.out.Decls, t.createExportMethods()...)

	// Add helpers.
	if len(t.helpers) > 0 {
		if *unsafe {
			if err := t.addHelpers(fset, "cpuarch_unsafe.go", helpersCpuArchSrc); err != nil {
				return err
			}
			if err := t.resolveHelpers(fset, "helpers_unsafe.go", helpersUnsafeSrc); err != nil {
				return err
			}
			if err := t.resolveHelpers(fset, "atomics_unsafe.go", helpersAtomicsSrc); err != nil {
				return err
			}
		}
		if err := t.resolveHelpers(fset, "helpers.go", helpersSrc); err != nil {
			return err
		}
		for name := range t.helpers {
			return fmt.Errorf("missing helper: %s", name)
		}
	}

	// Set imports.
	if len(t.packages) > 0 {
		specs := make([]ast.Spec, 0, len(t.data))
		for _, pkg := range slices.Sorted(maps.Keys(t.packages)) {
			spec := ast.ImportSpec{
				Path: &ast.BasicLit{Kind: token.STRING, Value: `"` + pkg + `"`},
			}
			if pkg == "embed" {
				spec.Name = newID("_")
			}
			specs = append(specs, &spec)
		}
		t.out.Decls = append([]ast.Decl{
			&ast.GenDecl{Tok: token.IMPORT, Specs: specs}},
			t.out.Decls...)
	}

	// Add data segments.
	if len(t.data) > 0 {
		if *embed {
			embedDecl := &ast.GenDecl{
				Tok: token.VAR,
				Doc: &ast.CommentGroup{
					List: []*ast.Comment{{Text: "//go:embed " + filepath.Base(embedFile)}},
				},
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{newID("data")},
						Type:  newID("string"),
					},
				},
			}
			t.out.Decls = append(t.out.Decls, embedDecl)
		} else {
			var specs []ast.Spec
			for i, seg := range t.data {
				if seg.merged {
					continue
				}
				specs = append(specs, &ast.ValueSpec{
					Names: []*ast.Ident{dataID(i)},
					Values: []ast.Expr{&ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(string(seg.init)),
					}},
				})
			}
			t.out.Decls = append(t.out.Decls, &ast.GenDecl{Tok: token.CONST, Specs: specs})
		}
	}

	util.RemoveParens(&t.out)

	// Print Go.
	out := bufio.NewWriter(w)
	out.WriteString("// Code generated by wasm2go. DO NOT EDIT.\n\n")
	if *tags != "" {
		out.WriteString("//go:build " + *tags + "\n\n")
	}
	err = format.Node(out, fset, &t.out)
	if err != nil {
		return err
	}
	return out.Flush()
}

type sectionID byte

const (
	sectionCustom sectionID = iota
	sectionType
	sectionImport
	sectionFunction
	sectionTable
	sectionMemory
	sectionGlobal
	sectionExport
	sectionStart
	sectionElement
	sectionCode
	sectionData
	sectionDataCount
)

func readHeader(r io.Reader) error {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return err
	}
	if magic := string(header[:4]); magic != "\x00asm" {
		return fmt.Errorf("invalid magic number: %q", magic)
	}
	if version := binary.LittleEndian.Uint32(header[4:]); version != 1 {
		return fmt.Errorf("invalid version: %d", version)
	}
	return nil
}

func (t *translator) readSection() error {
	id, err := t.in.ReadByte()
	if err != nil {
		return err
	}

	size, err := readLEB128(t.in)
	if err != nil {
		return err
	}

	switch sectionID(id) {
	case sectionType:
		return t.readTypeSection()
	case sectionImport:
		return t.readImportSection()
	case sectionFunction:
		return t.readFunctionSection()
	case sectionTable:
		return t.readTableSection()
	case sectionMemory:
		return t.readMemorySection()
	case sectionElement:
		return t.readElementSection()
	case sectionGlobal:
		return t.readGlobalSection()
	case sectionExport:
		return t.readExportSection()
	case sectionStart:
		return t.readStartSection()
	case sectionCode:
		return t.readCodeSection()
	case sectionData:
		return t.readDataSection()
	case sectionDataCount:
		return t.readDataCountSection()
	case sectionCustom:
		return t.readCustomSection(int(size))
	default:
		return fmt.Errorf("skipped section: %d", id)
	}
}

func (t *translator) readTypeSection() error {
	numTypes, err := readLEB128(t.in)
	if err != nil {
		return err
	}

	var buf strings.Builder
	t.types = make([]funcType, numTypes)
	for i := range t.types {
		form, err := t.in.ReadByte()
		if err != nil {
			return err
		}

		if form != 0x60 {
			return fmt.Errorf("unsupported form: %x", form)
		}

		// Parse parameter types.
		n, err := readLEB128(t.in)
		if err != nil {
			return err
		}

		_, err = io.CopyN(&buf, t.in, int64(n))
		if err != nil {
			return err
		}
		t.types[i].params = buf.String()
		buf.Reset()

		// Parse result types.
		n, err = readLEB128(t.in)
		if err != nil {
			return err
		}

		_, err = io.CopyN(&buf, t.in, int64(n))
		if err != nil {
			return err
		}
		t.types[i].results = buf.String()
		buf.Reset()
	}
	return nil
}

func (t *translator) readImportSection() error {
	count, err := readLEB128(t.in)
	if err != nil {
		return err
	}

	var buf strings.Builder
	for range count {
		n, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		_, err = io.CopyN(&buf, t.in, int64(n))
		if err != nil {
			return err
		}
		mod := buf.String()
		buf.Reset()

		n, err = readLEB128(t.in)
		if err != nil {
			return err
		}
		_, err = io.CopyN(&buf, t.in, int64(n))
		if err != nil {
			return err
		}
		name := buf.String()
		buf.Reset()

		kind, err := t.in.ReadByte()
		if err != nil {
			return err
		}

		switch externKind(kind) {
		case externFunction:
			index, err := readLEB128(t.in)
			if err != nil {
				return err
			}
			typ := t.types[index]

			if n := util.Mangle(name, util.IDInternal); t.provided.has(n) {
				id := ast.NewIdent(n)
				fn := funcCompiler{
					typ:      typ,
					decl:     &ast.FuncDecl{Name: id},
					call:     &ast.ParenExpr{X: &ast.SelectorExpr{X: newID("m"), Sel: id}},
					provided: true}
				t.functions = append(t.functions, fn)
				continue
			}

			t.imports = append(t.imports, importDef{
				module: mod,
				name:   name,
				kind:   externFunction,
				fnType: typ,
			})

			args := make([]ast.Expr, len(typ.params))
			for i := range typ.params {
				args[i] = localVar(i)
			}

			call := &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.SelectorExpr{X: newID("m"), Sel: util.MangleID(mod, util.IDInternal)},
					Sel: util.MangleID(name, util.IDExported)},
				Args: args,
			}

			var stmt ast.Stmt
			if len(typ.results) == 0 {
				stmt = &ast.ExprStmt{X: call}
			} else {
				stmt = &ast.ReturnStmt{Results: []ast.Expr{call}}
			}

			id := &ast.Ident{}
			fn := funcCompiler{
				typ: typ,
				decl: &ast.FuncDecl{
					Name: id,
					Recv: modRecvList,
					Type: typ.toAST(true),
					Body: &ast.BlockStmt{List: []ast.Stmt{stmt}}},
				call: &ast.ParenExpr{X: &ast.SelectorExpr{X: newID("m"), Sel: id}}}
			t.functions = append(t.functions, fn)
			t.out.Decls = append(t.out.Decls, fn.decl)

		case externMemory:
			if t.memory != nil {
				return errors.New("multiple memories not supported")
			}
			min, max, shared, is64, err := t.readLimits(65536) // 4 GiB
			if err != nil {
				return err
			}
			id := &ast.Ident{}
			t.memory = &memoryDef{
				id:       id,
				min:      int64(min),
				max:      int64(max),
				imported: true,
				shared:   shared,
				is64:     is64,
				selector: &ast.StarExpr{X: &ast.SelectorExpr{X: newID("m"), Sel: id}}}
			t.imports = append(t.imports, importDef{
				module: mod,
				name:   name,
				kind:   externMemory,
			})

		case externGlobal:
			typ, err := t.in.ReadByte()
			if err != nil {
				return err
			}
			_, err = t.in.ReadByte() // mutable
			if err != nil {
				return err
			}
			idx := len(t.globals)
			t.globals = append(t.globals, globalDef{
				id:       &ast.Ident{},
				typ:      wasmType(typ),
				imported: true,
			})
			t.imports = append(t.imports, importDef{
				module: mod,
				name:   name,
				kind:   externGlobal,
				typ:    wasmType(typ),
				index:  idx,
			})

		case externTable:
			typ, err := t.in.ReadByte()
			if err != nil {
				return err
			}
			if !wasmType(typ).ref() {
				return fmt.Errorf("unsupported table type: %x", typ)
			}

			min, max, _, is64, err := t.readLimits(65536) // 1 MiB
			if err != nil {
				return err
			}
			idx := len(t.tables)
			t.tables = append(t.tables, tableDef{
				id:       &ast.Ident{},
				min:      int(min),
				max:      int(max),
				is64:     is64,
				imported: true,
			})
			t.imports = append(t.imports, importDef{
				module: mod,
				name:   name,
				kind:   externTable,
				index:  idx,
			})

		default:
			return fmt.Errorf("unsupported import kind: %x", kind)
		}
	}
	return nil
}

func (t *translator) readFunctionSection() error {
	numFuncs, err := readLEB128(t.in)
	if err != nil {
		return err
	}

	start := len(t.functions)
	t.functions = append(t.functions, make([]funcCompiler, numFuncs)...)
	for i := range numFuncs {
		i += uint64(start)
		index, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		fn := &t.functions[i]
		fn.typ = t.types[index]
		fn.decl = &ast.FuncDecl{
			Name: &ast.Ident{},
			Recv: modRecvList,
			Type: fn.typ.toAST(true),
		}
		fn.call = &ast.ParenExpr{X: &ast.SelectorExpr{X: newID("m"), Sel: fn.decl.Name}}
		t.out.Decls = append(t.out.Decls, fn.decl)
	}
	return nil
}

func (t *translator) readTableSection() error {
	numTabs, err := readLEB128(t.in)
	if err != nil {
		return err
	}

	start := len(t.tables)
	t.tables = append(t.tables, make([]tableDef, numTabs)...)
	for i := range numTabs {
		i += uint64(start)
		typ, err := t.in.ReadByte()
		if err != nil {
			return err
		}
		if !wasmType(typ).ref() {
			return fmt.Errorf("unsupported table type: %x", typ)
		}

		min, max, _, is64, err := t.readLimits(65536) // 1 MiB
		if err != nil {
			return err
		}

		t.tables[i] = tableDef{
			id:   &ast.Ident{},
			min:  int(min),
			max:  int(max),
			is64: is64,
		}
	}
	return nil
}

func (t *translator) readMemorySection() error {
	numMems, err := readLEB128(t.in)
	if err != nil {
		return err
	}
	if numMems == 0 {
		return nil
	}
	if numMems > 1 {
		return errors.New("multiple memories not supported")
	}

	id := &ast.Ident{}
	min, max, shared, is64, err := t.readLimits(65536) // 4 GiB
	t.memory = &memoryDef{
		id:       id,
		min:      int64(min),
		max:      int64(max),
		shared:   shared,
		is64:     is64,
		selector: &ast.SelectorExpr{X: newID("m"), Sel: id}}
	return err
}

func (t *translator) readLimits(def uint64) (min, max uint64, shared, is64 bool, err error) {
	flags, err := readLEB128(t.in)
	if err != nil {
		return
	}
	shared = (flags & 2) != 0
	is64 = (flags & 4) != 0
	min, err = readLEB128(t.in)
	if err != nil {
		return
	}
	max = def
	if is64 && def == 65536 {
		max = 1 << 48
	}
	if flags&1 == 1 {
		max, err = readLEB128(t.in)
	}
	return
}

func (t *translator) readGlobalSection() error {
	numGlobals, err := readLEB128(t.in)
	if err != nil {
		return err
	}

	start := len(t.globals)
	t.globals = append(t.globals, make([]globalDef, numGlobals)...)
	for i := range numGlobals {
		i += uint64(start)
		g := &t.globals[i]
		g.id = &ast.Ident{}

		typ, err := t.in.ReadByte()
		if err != nil {
			return err
		}
		g.typ = wasmType(typ)

		_, err = t.in.ReadByte() // mutable
		if err != nil {
			return err
		}

		g.init, err = t.readConstExpr()
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *translator) readElementSection() error {
	count, err := readLEB128(t.in)
	if err != nil {
		return err
	}

	t.elements = make([]elemSegment, count)
	for i := range t.elements {
		tag, err := readLEB128(t.in)
		if err != nil {
			return err
		}

		if tag > 7 {
			return fmt.Errorf("unsupported element segment tag: %d", tag)
		}

		isPassive := tag&1 != 0
		hasIndex := tag&3 == 2
		hasType := tag&3 != 0
		hasExpr := tag&4 != 0

		t.elements[i].passive = isPassive

		if hasIndex {
			idx, err := readLEB128(t.in)
			if err != nil {
				return err
			}
			t.elements[i].index = uint32(idx)
		}

		if !isPassive {
			expr, err := t.readConstExpr()
			if err != nil {
				return err
			}
			t.elements[i].offset = expr
		}

		if hasType {
			typ, err := t.in.ReadByte()
			if err != nil {
				return err
			}
			if hasExpr && !wasmType(typ).ref() || !hasExpr && typ != 0x00 {
				return fmt.Errorf("unsupported element type: %x", typ)
			}
		}

		numElems, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		t.elements[i].init = make([]ast.Expr, numElems)
		for j := range t.elements[i].init {
			if hasExpr {
				expr, err := t.readConstExpr()
				if err != nil {
					return err
				}
				t.elements[i].init[j] = expr
			} else {
				idx, err := readLEB128(t.in)
				if err != nil {
					return err
				}
				t.elements[i].init[j] = t.functions[idx].call
			}
		}
	}
	return nil
}

func (t *translator) readExportSection() error {
	numExports, err := readLEB128(t.in)
	if err != nil {
		return err
	}

	var buf strings.Builder
	t.exports = make(map[string]export, numExports)
	for range numExports {
		n, err := readLEB128(t.in)
		if err != nil {
			return err
		}

		_, err = io.CopyN(&buf, t.in, int64(n))
		if err != nil {
			return err
		}
		name := buf.String()
		buf.Reset()

		kind, err := t.in.ReadByte()
		if err != nil {
			return err
		}
		index, err := readLEB128(t.in)
		if err != nil {
			return err
		}

		t.exports[name] = export{
			kind:  externKind(kind),
			index: int(index),
		}

		switch externKind(kind) {
		case externFunction:
			if !t.functions[index].provided {
				decl := t.functions[index].decl
				decl.Name.Name = util.Mangle(name, util.IDExported)
			}
		}
	}
	return nil
}

func (t *translator) readStartSection() error {
	index, err := readLEB128(t.in)
	if err != nil {
		return err
	}
	// Bitwise not makes the zero value useful (no start function).
	t.start = ^index
	return nil
}

func (t *translator) readConstExpr() (ast.Expr, error) {
	var stack stack[ast.Expr]

	for {
		opcode, err := t.in.ReadByte()
		if err != nil {
			return nil, err
		}

		switch opcode {
		case 0x41: // i32.const
			expr, err := t.constI32()
			if err != nil {
				return nil, err
			}
			stack.append(expr)
		case 0x42: // i64.const
			expr, err := t.constI64()
			if err != nil {
				return nil, err
			}
			stack.append(expr)
		case 0x43: // f32.const
			expr, err := t.constF32()
			if err != nil {
				return nil, err
			}
			stack.append(expr)
		case 0x44: // f64.const
			expr, err := t.constF64()
			if err != nil {
				return nil, err
			}
			stack.append(expr)
		case 0x23: // global.get
			expr, err := t.globalGet()
			if err != nil {
				return nil, err
			}
			stack.append(expr)
		case 0xd0: // ref.null
			_, err := t.in.ReadByte()
			if err != nil {
				return nil, err
			}
			stack.append(newID("nil"))
		case 0xd2: // ref.func
			index, err := readLEB128(t.in)
			if err != nil {
				return nil, err
			}
			stack.append(t.functions[index].call)

		case 0x6a, 0x7c: // i32.add, i64.add
			stack.append(&ast.BinaryExpr{Y: stack.pop(), X: stack.pop(), Op: token.ADD})
		case 0x6b, 0x7d: // i32.sub, i64.sub
			stack.append(&ast.BinaryExpr{Y: stack.pop(), X: stack.pop(), Op: token.SUB})
		case 0x6c, 0x7e: // i32.mul, i64.mul
			stack.append(&ast.BinaryExpr{Y: stack.pop(), X: stack.pop(), Op: token.MUL})

		case 0x0b: // end
			return stack[0], nil

		default:
			return nil, fmt.Errorf("unsupported opcode in constant expression: 0x%02X", opcode)
		}
	}
}

func (t *translator) readBlockType() (typ funcType, err error) {
	i, err := readSignedLEB128(t.in)
	if err != nil {
		return
	}
	switch {
	case i >= 0:
		return t.types[i], nil
	case i >= -4 || i == -16 || i == -17:
		typ.results = string([]wasmType{wasmType(i + 128)})
	case i != -64:
		err = fmt.Errorf("unsupported block type: %d", i)
	}
	return
}

func (t *translator) readDataCountSection() error {
	_, err := readLEB128(t.in)
	return err
}

func (t *translator) readDataSection() error {
	count, err := readLEB128(t.in)
	if err != nil {
		return err
	}

	if int(count) >= len(t.data) {
		t.data = append(t.data, make([]dataSegment, int(count)-len(t.data))...)
	}

	var (
		threshold  int64 = 64
		lastActive int   = -1
		lastDest   int64
		lastSize   int64
		fileOffset int64
	)

	var f *os.File
	if *embed {
		f, err = os.Create(embedFile)
		if err != nil {
			return err
		}
		defer f.Close()
		threshold = 4096
		t.packages.add("embed")
	}

	for i := range t.data {
		if tag, err := readLEB128(t.in); err != nil {
			return err
		} else if tag == 1 {
			t.data[i].passive = true
		} else if tag != 0 {
			return fmt.Errorf("unsupported data segment tag: %d", tag)
		}

		var dest int64 = -1

		if !t.data[i].passive {
			expr, err := t.readConstExpr()
			if err != nil {
				return err
			}
			t.data[i].offset = expr
			if v, ok := islit(expr, "i32"); ok {
				dest = v
			} else if v, ok := islit(expr, "i64"); ok {
				dest = v
			}
		}

		numElems, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		size := int64(numElems)

		if dest >= 0 && lastActive >= 0 {
			gap := dest - (lastDest + lastSize)
			if 0 <= gap && gap < threshold {
				if f == nil {
					data := &t.data[lastActive]
					data.init = append(data.init, make([]byte, gap)...)
					buf := make([]byte, size)
					if _, err := io.ReadFull(t.in, buf); err != nil {
						return err
					}
					data.init = append(data.init, buf...)
				} else {
					if _, err := f.Write(make([]byte, gap)); err != nil {
						return err
					}
					fileOffset += gap
					if _, err := io.CopyN(f, t.in, size); err != nil {
						return err
					}
					fileOffset += size
					t.dataExpr(lastActive).X.(*ast.SliceExpr).High.(*ast.BasicLit).Value = formatInt(fileOffset)
				}
				t.data[i].merged = true
				lastSize += gap + size
				continue
			}
		}

		if f == nil {
			data := &t.data[i]
			data.init = make([]byte, size)
			if _, err := io.ReadFull(t.in, data.init); err != nil {
				return err
			}
		} else {
			if _, err := io.CopyN(f, t.in, size); err != nil {
				return err
			}
			t.dataExpr(i).X = &ast.SliceExpr{
				X:    newID("data"),
				Low:  &ast.BasicLit{Kind: token.INT, Value: formatInt(fileOffset)},
				High: &ast.BasicLit{Kind: token.INT, Value: formatInt(fileOffset + size)},
			}
			fileOffset += size
		}

		if dest >= 0 {
			lastDest = dest
			lastSize = size
			lastActive = i
		} else {
			lastActive = -1
		}
	}

	if f != nil {
		return f.Close()
	}
	return nil
}

func (t *translator) readCustomSection(size int) error {
	data := make([]byte, size)
	if _, err := io.ReadFull(t.in, data); err != nil {
		return err
	}

	r := bytes.NewReader(data)
	n, err := readLEB128(r)
	if err != nil {
		return err
	}
	var buf strings.Builder
	if _, err := io.CopyN(&buf, r, int64(n)); err != nil {
		return err
	}
	if buf.String() == "name" {
		return t.readNameSection(r)
	} else if buf.String() == "dylink.0" {
		return t.readDylink0Section(r)
	}
	return nil
}

func (t *translator) readNameSection(r *bytes.Reader) error {
	seen := set[string]{}
	for r.Len() > 0 {
		kind, err := r.ReadByte()
		if err != nil {
			return err
		}
		size, err := readLEB128(r)
		if err != nil {
			return err
		}

		switch nameSubsection(kind) {
		case nameModule:
			n, err := readLEB128(r)
			if err != nil {
				return err
			}
			var buf strings.Builder
			if _, err := io.CopyN(&buf, r, int64(n)); err != nil {
				return err
			}
			name := buf.String()
			t.out.Name = util.MangleID(name, util.IDLocal)

		case nameFunction, nameGlobal, nameTable:
			count, err := readLEB128(r)
			if err != nil {
				return err
			}
			for range count {
				index, err := readLEB128(r)
				if err != nil {
					return err
				}
				n, err := readLEB128(r)
				if err != nil {
					return err
				}
				var buf strings.Builder
				if _, err := io.CopyN(&buf, r, int64(n)); err != nil {
					return err
				}

				var id *ast.Ident
				switch nameSubsection(kind) {
				case nameFunction:
					if int(index) < len(t.functions) {
						id = t.functions[index].decl.Name
					}
				case nameTable:
					if int(index) < len(t.tables) {
						id = t.tables[index].id
					}
				case nameGlobal:
					if int(index) < len(t.globals) {
						id = t.globals[index].id
					}
				}
				if id.Name == "" && seen.add(buf.String()) {
					id.Name = util.Mangle(buf.String(), util.IDInternal)
				}
			}

		default:
			_, err := r.Seek(int64(size), io.SeekCurrent)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *translator) readDylink0Section(r *bytes.Reader) error {
	t.dylink = &dylinkDef{}
	for r.Len() > 0 {
		kind, err := r.ReadByte()
		if err != nil {
			return err
		}
		size, err := readLEB128(r)
		if err != nil {
			return err
		}

		if dylinkKind(kind) == dylinkMemInfo {
			memSize, err := readLEB128(r)
			if err != nil {
				return err
			}
			t.dylink.memorySize = int64(memSize)

			memAlign, err := readLEB128(r)
			if err != nil {
				return err
			}
			t.dylink.memoryAlignment = int64(memAlign)

			tableSize, err := readLEB128(r)
			if err != nil {
				return err
			}
			t.dylink.tableSize = int64(tableSize)

			tableAlign, err := readLEB128(r)
			if err != nil {
				return err
			}
			t.dylink.tableAlignment = int64(tableAlign)
		} else {
			_, err := r.Seek(int64(size), io.SeekCurrent)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *translator) addHelpers(fset *token.FileSet, filename, src string) error {
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, decl := range f.Decls {
		if d, ok := decl.(*ast.GenDecl); ok && d.Tok == token.IMPORT {
			continue
		}
		t.out.Decls = append(t.out.Decls, decl)
		ast.Inspect(decl, t.resolveImports)
	}
	return nil
}

func (t *translator) resolveHelpers(fset *token.FileSet, filename, src string) error {
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if t.helpers.has(d.Name.Name) {
				t.out.Decls = append(t.out.Decls, d)
				ast.Inspect(d, t.resolveImports)
				delete(t.helpers, d.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if s, ok := spec.(*ast.TypeSpec); ok && t.helpers.has(s.Name.Name) {
					t.out.Decls = append(t.out.Decls, d)
					ast.Inspect(d, t.resolveImports)
					delete(t.helpers, s.Name.Name)
					break
				}
			}
		}
	}
	return nil
}

func (t *translator) resolveImports(n ast.Node) bool {
	if sel, ok := n.(*ast.SelectorExpr); ok {
		if id, ok := sel.X.(*ast.Ident); ok {
			if path, ok := stdlib[id.Name]; ok {
				t.packages.add(path)
			}
		}
	}
	return true
}

func (t *translator) dataExpr(i int) *ast.ParenExpr {
	if i >= len(t.data) {
		t.data = append(t.data, make([]dataSegment, i+1-len(t.data))...)
	}
	if t.data[i].embed == nil {
		t.data[i].embed = &ast.ParenExpr{X: dataID(i)}
	}
	return t.data[i].embed
}
