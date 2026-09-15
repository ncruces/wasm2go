package spectest

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unsafe"

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
						v, err := parseInt[int32](arg.Value.one())
						if err != nil {
							t.Fatal(err)
						}
						args[i] = reflect.ValueOf(v)
					case "i64":
						v, err := parseInt[int64](arg.Value.one())
						if err != nil {
							t.Fatal(err)
						}
						args[i] = reflect.ValueOf(v)
					case "f32":
						v, err := strconv.ParseUint(arg.Value.one(), 10, 32)
						if err != nil {
							t.Fatal(err)
						}
						args[i] = reflect.ValueOf(math.Float32frombits(uint32(v)))
					case "f64":
						v, err := strconv.ParseUint(arg.Value.one(), 10, 64)
						if err != nil {
							t.Fatal(err)
						}
						args[i] = reflect.ValueOf(math.Float64frombits(uint64(v)))
					case "funcref", "externref":
						if arg.Value.one() == "null" {
							var ptr *any
							args[i] = reflect.Zero(reflect.TypeOf(ptr).Elem())
						} else {
							args[i] = reflect.ValueOf(arg)
						}
					case "v128":
						lanes, err := laneBytes(arg)
						if err != nil {
							t.Fatal(err)
						}
						args[i] = newV128(method.Type().In(i), lanes)
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
							v, err := parseInt[int32](exp.Value.one())
							if err != nil {
								t.Fatal(err)
							}
							if i := res[i].Interface().(int32); i != v {
								if skipFloatBits(name) && isInfOrNaN32(v) && isInfOrNaN32(i) {
									t.Logf("got %d, want %d", i, v)
								} else {
									t.Errorf("got %d, want %d", i, v)
								}
							}
						case "i64":
							v, err := parseInt[int64](exp.Value.one())
							if err != nil {
								t.Fatal(err)
							}
							if i := res[i].Interface().(int64); i != v {
								if skipFloatBits(name) && isInfOrNaN64(v) && isInfOrNaN64(i) {
									t.Logf("got %d, want %d", i, v)
								} else {
									t.Errorf("got %d, want %d", i, v)
								}
							}
						case "f32":
							f := res[i].Interface().(float32)
							v := math.Float32bits(f)
							switch exp.Value.one() {
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
								i, err := strconv.ParseUint(exp.Value.one(), 10, 32)
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
							switch exp.Value.one() {
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
								i, err := strconv.ParseUint(exp.Value.one(), 10, 64)
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
						case "v128":
							compareV128(t, v128Bytes(res[i]), exp, name)
						}
					}
				}
			})
		}
	}
}

// newV128 builds a value of the generated module's v128 type, a struct of
// two little-endian uint64 words, from 16 lane bytes. The fields are
// unexported, so they are set through their addresses.
func newV128(typ reflect.Type, lanes [16]byte) reflect.Value {
	v := reflect.New(typ).Elem()
	for j := 0; j < 2; j++ {
		f := v.Field(j)
		w := binary.LittleEndian.Uint64(lanes[8*j:])
		reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem().SetUint(w)
	}
	return v
}

// v128Bytes returns the 16 lane bytes of a generated v128 value.
func v128Bytes(v reflect.Value) (r [16]byte) {
	binary.LittleEndian.PutUint64(r[0:], v.Field(0).Uint())
	binary.LittleEndian.PutUint64(r[8:], v.Field(1).Uint())
	return
}

// laneWidth returns the byte width of a v128 lane type.
func laneWidth(laneType string) (int, error) {
	switch laneType {
	case "i8":
		return 1, nil
	case "i16":
		return 2, nil
	case "i32", "f32":
		return 4, nil
	case "i64", "f64":
		return 8, nil
	}
	return 0, fmt.Errorf("unsupported lane type: %q", laneType)
}

// laneBytes assembles v128 lane values into little-endian bytes.
func laneBytes(arg specArg) (r [16]byte, err error) {
	w, err := laneWidth(arg.LaneType)
	if err != nil {
		return r, err
	}
	if len(arg.Value)*w != len(r) {
		return r, fmt.Errorf("bad v128 lane count: %d x %s", len(arg.Value), arg.LaneType)
	}
	for i, s := range arg.Value {
		v, err := parseInt[int64](s)
		if err != nil {
			return r, err
		}
		for b := 0; b < w; b++ {
			r[i*w+b] = byte(uint64(v) >> (8 * b))
		}
	}
	return r, nil
}

// compareV128 checks a v128 result lane by lane.
func compareV128(t *testing.T, got [16]byte, exp specArg, name string) {
	t.Helper()

	w, err := laneWidth(exp.LaneType)
	if err != nil {
		t.Fatal(err)
	}
	if len(exp.Value)*w != len(got) {
		t.Fatalf("bad v128 lane count: %d x %s", len(exp.Value), exp.LaneType)
	}
	for i, s := range exp.Value {
		var lane uint64
		for b := 0; b < w; b++ {
			lane |= uint64(got[i*w+b]) << (8 * b)
		}
		switch {
		case exp.LaneType == "f32" && s == "nan:canonical":
			if v := uint32(lane); v != 0xffc00000 && v != 0x7fc00000 {
				if skipCanonical() && isNaN32(v) {
					t.Logf("lane %d: got %x, want nan:canonical", i, v)
				} else {
					t.Errorf("lane %d: got %x, want nan:canonical", i, v)
				}
			}
		case exp.LaneType == "f32" && s == "nan:arithmetic":
			if v := uint32(lane); v&0x7fc00000 != 0x7fc00000 {
				if skipCanonical() && isNaN32(v) {
					t.Logf("lane %d: got %x, want nan:arithmetic", i, v)
				} else {
					t.Errorf("lane %d: got %x, want nan:arithmetic", i, v)
				}
			}
		case exp.LaneType == "f64" && s == "nan:canonical":
			if lane != 0xfff8000000000000 && lane != 0x7ff8000000000000 {
				if skipCanonical() && isNaN64(lane) {
					t.Logf("lane %d: got %x, want nan:canonical", i, lane)
				} else {
					t.Errorf("lane %d: got %x, want nan:canonical", i, lane)
				}
			}
		case exp.LaneType == "f64" && s == "nan:arithmetic":
			if lane&0x7ff8000000000000 != 0x7ff8000000000000 {
				if skipCanonical() && isNaN64(lane) {
					t.Logf("lane %d: got %x, want nan:arithmetic", i, lane)
				} else {
					t.Errorf("lane %d: got %x, want nan:arithmetic", i, lane)
				}
			}
		default:
			v, err := parseInt[int64](s)
			if err != nil {
				t.Fatal(err)
			}
			mask := uint64(1)<<(8*w-1)<<1 - 1
			want := uint64(v) & mask
			if lane != want {
				floats := exp.LaneType == "f32" && isNaN32(uint32(lane)) && isNaN32(uint32(want)) ||
					exp.LaneType == "f64" && isNaN64(lane) && isNaN64(want)
				if skipFloatBits(name) && floats {
					t.Logf("lane %d: got %d, want %d", i, lane, want)
				} else {
					t.Errorf("lane %d: got %d, want %d", i, lane, want)
				}
			}
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

func parseInt[T int32 | int64](s string) (T, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		return T(v), nil
	}
	if u, err := strconv.ParseUint(s, 10, 64); err == nil {
		return T(u), nil
	}
	return 0, err
}

func isNaN32[T int32 | uint32](bits T) bool {
	return uint32(bits&0x7FFFFFFF) > 0x7F800000
}

func isNaN64[T int64 | uint64](bits T) bool {
	return uint64(bits&0x7FFFFFFFFFFFFFFF) > 0x7FF0000000000000
}

func isInfOrNaN32[T int32 | uint32](bits T) bool {
	return uint32(bits&0x7FFFFFFF) >= 0x7F800000
}

func isInfOrNaN64[T int64 | uint64](bits T) bool {
	return uint64(bits&0x7FFFFFFFFFFFFFFF) >= 0x7FF0000000000000
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
