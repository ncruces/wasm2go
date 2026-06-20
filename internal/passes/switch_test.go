package passes

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestInlineSwitchGotos(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "inline last case",
			src: `func f(x int) {
				switch x {
				default:
					return
				case 0:
					return
				case 1:
					goto lbl
				}
			lbl:
				;
				x++
				return
			}`,
			want: `func f(x int) {
				switch x {
				default:
					return
				case 0:
					return
				case 1:
					x++
					return
				}
			}`,
		},
		{
			name: "nested switch",
			src: `func f(x int) {
				{
					switch x {
					default:
						return
					case 0:
						return
					case 1:
						goto lbl
					}
				}
			lbl:
				;
				x++
				return
			}`,
			want: `func f(x int) {
				{
					switch x {
					default:
						return
					case 0:
						return
					case 1:
						x++
						return
					}
				}
			}`,
		},
		{
			name: "inline first case",
			src: `func f(x int) {
				switch x {
				case 0:
					goto lbl
				case 1:
					return
				default:
					return
				}
			lbl:
				;
				x++
				return
			}`,
			want: `func f(x int) {
				switch x {
				case 1:
					return
				default:
					return
				case 0:
					x++
					return
				}
			}`,
		},
		{
			name: "inline middle case",
			src: `func f(x int) {
				switch x {
				case 0:
					return
				case 1:
					goto lbl
				default:
					return
				}
			lbl:
				;
				x++
				return
			}`,
			want: `func f(x int) {
				switch x {
				case 0:
					return
				default:
					return
				case 1:
					x++
					return
				}
			}`,
		},
		{
			name: "no-op: label used more than once",
			src: `func f(x int) {
				switch x {
				case 0:
					goto lbl
				default:
					goto lbl
				}
			lbl:
				;
				return
			}`,
			want: `func f(x int) {
				switch x {
				case 0:
					goto lbl
				default:
					goto lbl
				}
			lbl:
				;
				return
			}`,
		},
		{
			name: "no-op: no default clause",
			src: `func f(x int) {
				switch x {
				case 0:
					goto lbl
				case 1:
					return
				}
			lbl:
				;
				return
			}`,
			want: `func f(x int) {
				switch x {
				case 0:
					goto lbl
				case 1:
					return
				}
			lbl:
				;
				return
			}`,
		},
		{
			name: "no-op: preceding case has fallthrough",
			src: `func f(x int) {
				switch x {
				case 0:
					fallthrough
				case 1:
					goto lbl
				default:
					return
				}
			lbl:
				;
				return
			}`,
			want: `func f(x int) {
				switch x {
				case 0:
					fallthrough
				case 1:
					goto lbl
				default:
					return
				}
			lbl:
				;
				return
			}`,
		},
		{
			name: "no-op: stmts contain break",
			src: `func f(x int) {
				switch {
				default:
					switch x {
					case 0:
						goto lbl
					default:
						return
					}
				lbl:
					;
					break
				}
			}`,
			want: `func f(x int) {
				switch {
				default:
					switch x {
					case 0:
						goto lbl
					default:
						return
					}
				lbl:
					;
					break
				}
			}`,
		},
		{
			name: "no-op: stmts end with fallthrough",
			src: `func f(x int) {
				switch x {
				case 0:
					switch x {
					case 0:
						goto lbl
					default:
						return
					}
				lbl:
					;
					fallthrough
				default:
				}
			}`,
			want: `func f(x int) {
				switch x {
				case 0:
					switch x {
					case 0:
						goto lbl
					default:
						return
					}
				lbl:
					;
					fallthrough
				default:
				}
			}`,
		},
		{
			name: "no-op: inlined label has external goto",
			src: `func f(x int) {
				goto exit
				switch x {
				case 0:
					goto lbl
				default:
					return
				}
			lbl:
				;
			exit:
				return
			}`,
			want: `func f(x int) {
				goto exit
				switch x {
				case 0:
					goto lbl
				default:
					return
				}
			lbl:
				;
			exit:
				return
			}`,
		},
		{
			name: "no-op: switch can't complete",
			src: `func f(x int) {
				switch x {
				case 0:
					goto lbl
				default:
					break
				case 1:
					x++
				}
			lbl:
				;
				x--
				return
			}`,
			want: `func f(x int) {
				switch x {
				case 0:
					goto lbl
				default:
					break
				case 1:
					x++
				}
			lbl:
				;
				x--
				return
			}`,
		},
		{
			name: "last case can complete: fallthrough added",
			src: `func f(x int) {
				switch x {
				case 0:
					goto lbl
				default:
					return
				case 1:
					x++
				}
			lbl:
				;
				x--
				return
			}`,
			want: `func f(x int) {
				switch x {
				default:
					return
				case 1:
					x++
					fallthrough
				case 0:
					x--
					return
				}
			}`,
		},
		{
			name: "empty inlined stmts",
			src: `func f(x int) {
				switch x {
				case 0:
					goto lbl
				default:
					return
				}
			lbl:
			}`,
			want: `func f(x int) {
				switch x {
				default:
					return
				case 0:
				}
			}`,
		},
		{
			name: "nested return",
			src: `func f(x int) {
				switch x {
				case 0:
					goto lbl
				default:
					{
						return
					}
				}
			lbl:
			}`,
			want: `func f(x int) {
				switch x {
				default:
					{
						return
					}
				case 0:
				}
			}`,
		},
		{
			name: "iterative: two patterns collapsed",
			src: `func f(x int) {
				switch x {
				case 0:
					goto lbl1
				default:
					return
				}
			lbl1:
				;
				switch x {
				case 1:
					goto lbl2
				default:
					return
				}
			lbl2:
				;
				return
			}`,
			want: `func f(x int) {
				switch x {
				default:
					return
				case 0:
					switch x {
					default:
						return
					case 1:
						return
					}
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := parseFunc(t, tt.src)
			InlineSwitchGotos(fn)
			got := formatFunc(t, fn)
			want := normalizeFunc(t, tt.want)
			if got != want {
				t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
			}
		})
	}
}

func parseFunc(t *testing.T, src string) *ast.FuncDecl {
	t.Helper()
	src = "package p\n" + src
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return f.Decls[0].(*ast.FuncDecl)
}

func formatFunc(t *testing.T, fn *ast.FuncDecl) string {
	t.Helper()
	fset := token.NewFileSet()
	file := &ast.File{
		Name:  ast.NewIdent("p"),
		Decls: []ast.Decl{fn},
	}
	var sb strings.Builder
	if err := format.Node(&sb, fset, file); err != nil {
		t.Fatalf("format error: %v", err)
	}
	out := sb.String()
	out = strings.TrimPrefix(out, "package p\n\n")
	return strings.TrimSpace(out)
}

func normalizeFunc(t *testing.T, src string) string {
	t.Helper()
	fn := parseFunc(t, src)
	return formatFunc(t, fn)
}
