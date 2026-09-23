package main

import (
	"bytes"
	"cmp"
	"debug/dwarf"
	"fmt"
	"go/ast"
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

func (t *translator) readDebugSection(name string, r *bytes.Reader) error {
	section, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if t.debugSections == nil {
		t.debugSections = map[string][]byte{}
	}
	t.debugSections[name] = section
	return nil
}

type dwarfLine struct {
	addr uint64
	file string
	line int
	col  int
}

const (
	dwarfPCStart  = "_dwarfStart" // start of instruction stream for a function
	dwarfPCEnd    = "_dwarfEnd"   // end of instruction stream for a function
	dwarfPCPrefix = "_dwarfPC_"   // prefix for per-instruction offset markers
)

func (fn *funcCompiler) markDebugStart() {
	body := fn.decl.Body
	body.List = append(body.List, &ast.ExprStmt{X: ast.NewIdent(dwarfPCStart)})
}

func (fn *funcCompiler) markDebugEnd() {
	body := fn.decl.Body
	body.List = append(body.List, &ast.ExprStmt{X: ast.NewIdent(dwarfPCEnd)})
}

func (fn *funcCompiler) markDebugOffset(offset uint64) {
	body := fn.blocks.top().body
	body.List = append(body.List, &ast.ExprStmt{X: ast.NewIdent(dwarfPCPrefix + strconv.FormatUint(offset, 10))})
}

func loadDwarfLines(sections map[string][]byte) ([]dwarfLine, error) {
	dbg, err := dwarf.New(
		sections["abbrev"], sections["aranges"], sections["frame"],
		sections["info"], sections["line"], sections["pubnames"],
		sections["ranges"], sections["str"])
	if err != nil {
		return nil, err
	}

	var lines []dwarfLine
	r := dbg.Reader()
	for {
		e, err := r.Next()
		if err != nil {
			return nil, err
		}
		if e == nil {
			break
		}
		if e.Tag != dwarf.TagCompileUnit {
			continue
		}
		lr, err := dbg.LineReader(e)
		if err != nil {
			return nil, err
		}
		for {
			var le dwarf.LineEntry
			if err := lr.Next(&le); err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			if !le.EndSequence {
				lines = append(lines, dwarfLine{
					addr: le.Address,
					file: le.File.Name,
					line: le.Line,
					col:  le.Column,
				})
			}
		}
	}

	slices.SortFunc(lines, func(a, b dwarfLine) int {
		return cmp.Compare(a.addr, b.addr)
	})
	return lines, nil
}

func injectDwarfLines(buf []byte, sections map[string][]byte) ([]byte, error) {
	lines, err := loadDwarfLines(sections)
	if err != nil {
		return nil, err
	}

	abs, err := filepath.Abs(*output)
	if err != nil {
		return nil, err
	}

	// Report paths relative to output.
	dir, file := filepath.Split(abs)
	for i := range lines {
		a, err := filepath.Abs(lines[i].file)
		if err != nil {
			continue
		}
		p, err := filepath.Rel(dir, a)
		if err != nil {
			continue
		}
		lines[i].file = p
	}

	var (
		out    bytes.Buffer
		cur    *dwarfLine
		prev   dwarfLine
		lineNo int
		inCode bool
	)
	for raw := range bytes.Lines(buf) {
		trimmed := string(bytes.TrimRight(bytes.TrimLeft(raw, "\t"), "\n"))

		switch {
		case trimmed == dwarfPCStart:
			inCode = true
			continue
		case trimmed == dwarfPCEnd:
			inCode = false
			cur = nil
			continue
		case inCode && strings.HasPrefix(trimmed, dwarfPCPrefix):
			offset, err := strconv.ParseUint(trimmed[len(dwarfPCPrefix):], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid wasm PC marker: %s", trimmed)
			}
			// Find the last dwarf line entry with addr <= offset (the one that covers this pc).
			i, ok := slices.BinarySearchFunc(lines, offset, func(e dwarfLine, t uint64) int {
				return cmp.Compare(e.addr, t)
			})
			if ok {
				cur = &lines[i]
			} else if i > 0 {
				cur = &lines[i-1]
			} else {
				cur = nil
			}
			continue
		}

		lineNo++
		if !inCode {
			cur = &dwarfLine{file: file, line: lineNo, col: 1}
		}

		if cur != nil && len(trimmed) > 0 &&
			!strings.HasPrefix(trimmed, "/*") &&
			!strings.HasPrefix(trimmed, "//") {
			var file string
			if cur.file != "" && cur.file != prev.file {
				prev.file = cur.file
				file = cur.file
			}
			line, col := cur.line, cur.col
			if line > 0 {
				prev.line = line
			} else {
				line = prev.line
			}
			if line > 0 {
				out.WriteString("/*line ")
				if file != "" {
					out.WriteString(file)
				}
				out.WriteByte(':')
				out.WriteString(strconv.Itoa(line))
				if col > 0 {
					out.WriteByte(':')
					out.WriteString(strconv.Itoa(col))
				}
				out.WriteString("*/")
			}
		}
		out.Write(raw)
	}
	return out.Bytes(), nil
}
