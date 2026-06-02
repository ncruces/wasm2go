package spectest

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/ncruces/wasm2go/internal/mangle"
)

func TestModule(t *testing.T, ctor func() any, jsonPath, name string) {
	t.Helper()

	spec, err := parseSpec(jsonPath)
	if err != nil {
		t.Fatal(err)
	}

	exp := classifyModule(spec, name)
	switch exp.mode {
	case moduleTestUnlinkable:
		if exp.text == "" {
			t.Skip("module is assert_unlinkable")
		} else {
			t.Skipf("module is assert_unlinkable: %s", exp.text)
		}
	case moduleTestUninstantiable:
		defer RecoverTrap(t, exp.text)
		_ = ctor()
	case moduleTestRuntime:
		runAssertions(t, reflect.ValueOf(ctor()), spec, name)
	default:
		_ = ctor()
	}
}

func runAssertions(t *testing.T, mod reflect.Value, spec *specTest, name string) {
	var file string
	for _, cmd := range spec.Commands {
		if cmd.Type == "module" {
			file = cmd.Filename
			continue
		} else if file != name {
			continue
		}

		switch cmd.Type {
		case "action", "assert_return", "assert_trap":
			t.Run(fmt.Sprintf("%s/line_%d", name, cmd.Line), func(t *testing.T) {
				if cmd.Type == "assert_trap" {
					defer RecoverTrap(t, cmd.Text)
				}

				method := mod.MethodByName(mangle.Name(cmd.Action.Field, mangle.Exported))
				args := make([]reflect.Value, len(cmd.Action.Args))
				for i, arg := range cmd.Action.Args {
					switch arg.Type {
					case "i32":
						v, err := strconv.ParseUint(arg.Value, 10, 32)
						if err != nil {
							t.Fatal(err)
						}
						args[i] = reflect.ValueOf(int32(v))
					case "i64":
						v, err := strconv.ParseUint(arg.Value, 10, 64)
						if err != nil {
							t.Fatal(err)
						}
						args[i] = reflect.ValueOf(int64(v))
					case "f32":
						v, err := strconv.ParseUint(arg.Value, 10, 32)
						if err != nil {
							t.Fatal(err)
						}
						args[i] = reflect.ValueOf(math.Float32frombits(uint32(v)))
					case "f64":
						v, err := strconv.ParseUint(arg.Value, 10, 64)
						if err != nil {
							t.Fatal(err)
						}
						args[i] = reflect.ValueOf(math.Float64frombits(uint64(v)))
					case "funcref", "externref":
						if arg.Value == "null" {
							var ptr *any
							args[i] = reflect.Zero(reflect.TypeOf(ptr).Elem())
						} else {
							args[i] = reflect.ValueOf(arg)
						}
					}
				}

				res := method.Call(args)
				for i := range res {
					if res[i].Kind() == reflect.Pointer {
						res[i] = res[i].Elem()
					}
				}
				if cmd.Type == "assert_return" {
					for i, exp := range cmd.Expected {
						switch exp.Type {
						case "i32":
							v, err := strconv.ParseUint(exp.Value, 10, 32)
							if err != nil {
								t.Fatal(err)
							}
							if i := res[i].Interface().(int32); i != int32(v) {
								if skipFloatBits(name) && isInfOrNaN32(uint32(v)) && isInfOrNaN32(uint32(i)) {
									t.Logf("got %d, want %d", i, int32(v))
								} else {
									t.Errorf("got %d, want %d", i, int32(v))
								}
							}
						case "i64":
							v, err := strconv.ParseUint(exp.Value, 10, 64)
							if err != nil {
								t.Fatal(err)
							}
							if i := res[i].Interface().(int64); i != int64(v) {
								if skipFloatBits(name) && isInfOrNaN64(uint64(v)) && isInfOrNaN64(uint64(i)) {
									t.Logf("got %d, want %d", i, int64(v))
								} else {
									t.Errorf("got %d, want %d", i, int64(v))
								}
							}
						case "f32":
							f := res[i].Interface().(float32)
							v := math.Float32bits(f)
							switch exp.Value {
							case "nan:canonical":
								if v != 0xffc00000 && v != 0x7fc00000 {
									if skipCanonical() && isNaN32(v) {
										t.Logf("got %x, want nan:canonical", v)
									} else {
										t.Errorf("got %x, want nan:canonical", v)
									}
								}
							case "nan:arithmetic":
								if v&0x7fc00000 != 0x7fc00000 {
									if skipCanonical() && isNaN32(v) {
										t.Logf("got %x, want nan:arithmetic", v)
									} else {
										t.Errorf("got %x, want nan:arithmetic", v)
									}
								}
							default:
								i, err := strconv.ParseUint(exp.Value, 10, 32)
								if err != nil {
									t.Fatal(err)
								}
								if v != uint32(i) {
									if skipFloatBits(name) && isNaN32(v) && isNaN32(uint32(i)) {
										t.Logf("got %d, want %d", v, uint32(i))
									} else {
										t.Errorf("got %d, want %d", v, uint32(i))
									}
								}
							}
						case "f64":
							f := res[i].Interface().(float64)
							v := math.Float64bits(f)
							switch exp.Value {
							case "nan:canonical":
								if v != 0xfff8000000000000 && v != 0x7ff8000000000000 {
									if skipCanonical() && isNaN64(v) {
										t.Logf("got %x, want nan:canonical", v)
									} else {
										t.Errorf("got %x, want nan:canonical", v)
									}
								}
							case "nan:arithmetic":
								if v&0x7ff8000000000000 != 0x7ff8000000000000 {
									if skipCanonical() && isNaN64(v) {
										t.Logf("got %x, want nan:arithmetic", v)
									} else {
										t.Errorf("got %x, want nan:arithmetic", v)
									}
								}
							default:
								i, err := strconv.ParseUint(exp.Value, 10, 64)
								if err != nil {
									t.Fatal(err)
								}
								if v != uint64(i) {
									if skipFloatBits(name) && isNaN64(v) && isNaN64(i) {
										t.Logf("got %d, want %d", v, uint64(i))
									} else {
										t.Errorf("got %d, want %d", v, uint64(i))
									}
								}
							}
						}
					}
				}
			})
		}
	}
}

func RecoverTrap(t testing.TB, want string) {
	t.Helper()

	var got string
	if r := recover(); r != nil {
		got = fmt.Sprint(r)
	} else {
		t.Fatalf("want trap: %s", want)
	}

	switch {
	case strings.Contains(got, want):
		return
	case strings.Contains(want, "out of bounds"):
		if strings.Contains(got, "out of range") || strings.Contains(got, "cannot convert slice with length") {
			return
		}
	case strings.Contains(want, "undefined"):
		if strings.Contains(got, "out of range") {
			return
		}
	case strings.Contains(want, "type mismatch") || strings.Contains(want, "indirect call"):
		if strings.Contains(got, "interface conversion") {
			return
		}
	case strings.Contains(want, "uninitialized"):
		if strings.Contains(got, "is nil") {
			return
		}
	}

	t.Fatalf("got trap %q, want %q", got, want)
}

// We only check for canonical NaNs on amd64 and arm64.
func skipCanonical() bool {
	switch runtime.GOARCH {
	case "amd64", "arm64":
		return false
	}
	return true
}

// We skip specific float bit pattern checks (infinities, NaNs) on s390x and MIPS.
func skipFloatBits(name string) bool {
	if runtime.GOARCH == "s390x" || strings.HasPrefix(runtime.GOARCH, "mips") {
		return (strings.Contains(name, "float") ||
			strings.Contains(name, "f32") ||
			strings.Contains(name, "f64"))
	}
	return false
}

func isNaN32(bits uint32) bool {
	return (bits & 0x7FFFFFFF) > 0x7F800000
}

func isNaN64(bits uint64) bool {
	return (bits & 0x7FFFFFFFFFFFFFFF) > 0x7FF0000000000000
}

func isInfOrNaN32(bits uint32) bool {
	return (bits & 0x7FFFFFFF) >= 0x7F800000
}

func isInfOrNaN64(bits uint64) bool {
	return (bits & 0x7FFFFFFFFFFFFFFF) >= 0x7FF0000000000000
}
