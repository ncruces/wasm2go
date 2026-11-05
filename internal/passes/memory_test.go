package passes

import (
	"go/ast"
	"testing"

	"golang.org/x/tools/go/ast/astutil"
)

func TestSelectMemoryMode(t *testing.T) {
	memSlice := &ast.SelectorExpr{X: ast.NewIdent("m"), Sel: ast.NewIdent("memory")}
	impSlice := &ast.StarExpr{X: memSlice}

	tests := []struct {
		name  string
		slice ast.Expr
		src   string
		want  MemoryMode
	}{
		{
			name:  "address-of memory without reads",
			slice: memSlice,
			src: `func (m *Module) f(v0 int32) int32 {
				return int32(memory_grow(&m.memory, int64(v0), maxMem))
			}`,
			want: LoadOnRead,
		},
		{
			name:  "single read outside loop",
			slice: memSlice,
			src: `func (m *Module) f(v0 int32) int32 {
				return int32(load32(m.memory, uint32(v0)))
			}`,
			want: LoadOnRead,
		},
		{
			name:  "forward goto with single read",
			slice: impSlice,
			src: `func (m *Module) f(v0 int32) int32 {
				if v0 == 0 {
					goto l0
				}
				store32(*m.memory, 0, 1)
			l0:
				return 0
			}`,
			want: LoadOnRead,
		},
		{
			name:  "two reads without calls",
			slice: memSlice,
			src: `func (m *Module) f(v0 int32) int32 {
				store32(m.memory, uint32(v0), 1)
				return int32(load32(m.memory, uint32(v0)))
			}`,
			want: LoadOnWrite,
		},
		{
			name:  "single read in backward goto loop",
			slice: impSlice,
			src: `func (m *Module) f(v0, v1 int32) int32 {
				var v2, v3 int32
			l0:
				if v2 >= v1 {
					return v3
				}
				v3 += int32(load32(*m.memory, uint32(v0)))
				v2++
				goto l0
			}`,
			want: LoadOnWrite,
		},
		{
			name:  "calls outside active span ignored",
			slice: memSlice,
			src: `func (m *Module) f(v0 int32) int32 {
				t0 := (m._alloc)()
				store32(m.memory, uint32(v0), uint32(t0))
				v1 := int32(load32(m.memory, uint32(v0)))
				_ = memory_grow(&m.memory, 1, maxMem)
				return v1
			}`,
			want: LoadOnWrite,
		},
		{
			name:  "two reads with one call in span",
			slice: memSlice,
			src: `func (m *Module) f(v0 int32) int32 {
				store32(m.memory, 0, 1)
				(m._alloc)()
				return int32(load32(m.memory, 0))
			}`,
			want: LoadOnRead,
		},
		{
			name:  "three reads with one call in span",
			slice: memSlice,
			src: `func (m *Module) f(v0 int32) int32 {
				store32(m.memory, 0, 1)
				(m._alloc)()
				store32(m.memory, 4, 2)
				return int32(load32(m.memory, 0))
			}`,
			want: LoadOnWrite,
		},
		{
			name:  "two reads with call in loop stays LoadOnRead",
			slice: impSlice,
			src: `func (m *Module) f(v0 int32) int32 {
				store32(*m.memory, 0, 1)
				store32(*m.memory, 4, 2)
			l0:
				t0 := (m._step)()
				if t0 != 0 {
					goto l0
				}
				return 0
			}`,
			want: LoadOnRead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := parseFunc(t, tt.src)
			internMemSlice(fn, tt.slice)
			if got := SelectMemoryMode(fn, tt.slice, testMayGrow); got != tt.want {
				t.Errorf("SelectMemoryMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadMemoryOnWrite(t *testing.T) {
	memSlice := &ast.SelectorExpr{X: ast.NewIdent("m"), Sel: ast.NewIdent("memory")}
	impSlice := &ast.StarExpr{X: memSlice}
	memID := ast.NewIdent("mem")

	tests := []struct {
		name  string
		slice ast.Expr
		src   string
		want  string
	}{
		{
			name:  "leading guards and calls skipped and trailing grow unrefreshed",
			slice: memSlice,
			src: `func (m *Module) f(v0 int32) int32 {
				var v1 int32
				if v0 == 0 {
					return 0
				}
				t0 := (m._alloc)()
				store32(m.memory, uint32(v0), uint32(t0))
				v1 = int32(load32(m.memory, uint32(v0)))
				_ = memory_grow(&m.memory, 1, maxMem)
				return v1
			}`,
			want: `func (m *Module) f(v0 int32) int32 {
				var v1 int32
				if v0 == 0 {
					return 0
				}
				t0 := (m._alloc)()
				mem := m.memory
				store32(mem, uint32(v0), uint32(t0))
				v1 = int32(load32(mem, uint32(v0)))
				_ = memory_grow(&m.memory, 1, maxMem)
				return v1
			}`,
		},
		{
			name:  "loop hoist for imported memory",
			slice: impSlice,
			src: `func (m *Module) f(v0, v1 int32) int32 {
				var v2, v3 int32
				v2 = 0
			l0:
				if v2 >= v1 {
					return v3
				}
				v3 += int32(load32(*m.memory, uint32(v0)))
				v2++
				goto l0
			}`,
			want: `func (m *Module) f(v0, v1 int32) int32 {
				var v2, v3 int32
				v2 = 0
				mem := *m.memory
			l0:
				if v2 >= v1 {
					return v3
				}
				v3 += int32(load32(mem, uint32(v0)))
				v2++
				goto l0
			}`,
		},
		{
			name:  "grow calls reload mem while preserving address-of and pure helpers",
			slice: memSlice,
			src: `func (m *Module) f(v0 int32) int32 {
				t0 := (fn0)(v0)
				store32(m.memory, 0, uint32(t0))
				if v0 != 0 {
					_ = memory_grow(&m.memory, 1, maxMem)
				}
				(m._alloc)()
				m.t0[uint(v0)].(func())()
				return int32(load32(m.memory, 0))
			}`,
			want: `func (m *Module) f(v0 int32) int32 {
				t0 := (fn0)(v0)
				mem := m.memory
				store32(mem, 0, uint32(t0))
				if v0 != 0 {
					_ = memory_grow(&m.memory, 1, maxMem)
					mem = m.memory
				}
				(m._alloc)()
				mem = m.memory
				m.t0[uint(v0)].(func())()
				mem = m.memory
				return int32(load32(mem, 0))
			}`,
		},
		{
			name:  "switch cases and byte index and len rewrites",
			slice: memSlice,
			src: `func (m *Module) f(v0, v1 int32) int32 {
				m.memory[uint32(v0)] = byte(v1)
				t0 := int32(len(m.memory) >> 16)
				switch v0 {
				case 0:
					(m._alloc)()
					fallthrough
				case 1:
					m.memory[uint32(v0)+1] = byte(t0)
				}
				return int32(m.memory[uint32(v0)])
			}`,
			want: `func (m *Module) f(v0, v1 int32) int32 {
				mem := m.memory
				mem[uint32(v0)] = byte(v1)
				t0 := int32(len(mem) >> 16)
				switch v0 {
				case 0:
					(m._alloc)()
					mem = m.memory
					fallthrough
				case 1:
					mem[uint32(v0)+1] = byte(t0)
				}
				return int32(mem[uint32(v0)])
			}`,
		},
		{
			name:  "forward goto and labeled grow keep mem in scope",
			slice: impSlice,
			src: `func (m *Module) f(v0 int32) int32 {
				if v0 == 0 {
					goto l0
				}
				store32(*m.memory, 0, 1)
			l0:
				_ = m.memImp.Grow(1, maxMem)
				return int32(load32(*m.memory, 0))
			}`,
			want: `func (m *Module) f(v0 int32) int32 {
				mem := *m.memory
				if v0 == 0 {
					goto l0
				}
				store32(mem, 0, 1)
			l0:
				_ = m.memImp.Grow(1, maxMem)
				mem = *m.memory
				return int32(load32(mem, 0))
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := parseFunc(t, tt.src)
			internMemSlice(fn, tt.slice)
			LoadMemoryOnWrite(fn, tt.slice, memID, testMayGrow)
			got := formatFunc(t, fn)
			want := normalizeFunc(t, tt.want)
			if got != want {
				t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
			}
		})
	}
}

func testMayGrow(ce *ast.CallExpr) bool {
	switch fn := ce.Fun.(type) {
	case *ast.ParenExpr:
		_, ok := fn.X.(*ast.SelectorExpr)
		return ok
	case *ast.TypeAssertExpr:
		return true
	case *ast.Ident:
		return fn.Name == "memory_grow"
	case *ast.SelectorExpr:
		_, ok := fn.X.(*ast.SelectorExpr)
		return ok && fn.Sel.Name == "Grow"
	}
	return false
}

func internMemSlice(fn *ast.FuncDecl, slice ast.Expr) {
	isMemSel := func(n ast.Node) bool {
		s, ok := n.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		x, ok := s.X.(*ast.Ident)
		return ok && x.Name == "m" && s.Sel.Name == "memory"
	}
	astutil.Apply(fn.Body, func(c *astutil.Cursor) bool {
		switch slice.(type) {
		case *ast.StarExpr:
			if st, ok := c.Node().(*ast.StarExpr); ok && isMemSel(st.X) {
				c.Replace(slice)
				return false
			}
		case *ast.SelectorExpr:
			if isMemSel(c.Node()) {
				c.Replace(slice)
				return false
			}
		}
		return true
	}, nil)
}
