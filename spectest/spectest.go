package spectest

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/ncruces/wasm2go/util"
)

func Test(t *testing.T, modptr any, data []byte, name string) {
	if skipFloat(name) {
		t.SkipNow()
	}

	var test struct {
		Commands []struct {
			Type     string `json:"type"`
			Line     int    `json:"line"`
			Filename string `json:"filename"`
			Action   struct {
				Field string `json:"field"`
				Args  []struct {
					Type  string `json:"type"`
					Value string `json:"value"`
				} `json:"args"`
			} `json:"action"`
			Text     string `json:"text"`
			Expected []struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"expected"`
		} `json:"commands"`
	}

	mod := reflect.ValueOf(modptr)

	if err := json.Unmarshal(data, &test); err != nil {
		t.Fatal(err)
	}

	var file string
	for _, cmd := range test.Commands {
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

				method := mod.MethodByName(util.Mangle(cmd.Action.Field, util.IDExported))
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
							if got, want := res[i].Interface().(int32), int32(v); got != want {
								t.Errorf("got %d, want %d", got, want)
							}
						case "i64":
							v, err := strconv.ParseUint(exp.Value, 10, 64)
							if err != nil {
								t.Fatal(err)
							}
							if got, want := res[i].Interface().(int64), int64(v); got != want {
								t.Errorf("got %d, want %d", got, want)
							}
						case "f32":
							f := res[i].Interface().(float32)
							v := math.Float32bits(f)
							switch exp.Value {
							case "nan:canonical":
								if !testCanonical() && math.IsNaN(float64(f)) {
									t.Logf("got %x, want nan:canonical", v)
									break
								}
								if v != 0xffc00000 && v != 0x7fc00000 {
									t.Errorf("got %x, want nan:canonical", v)
								}
							case "nan:arithmetic":
								if !testCanonical() && math.IsNaN(float64(f)) {
									t.Logf("got %x, want nan:arithmetic", v)
									break
								}
								if v&0x7fc00000 != 0x7fc00000 {
									t.Errorf("got %x, want nan:arithmetic", v)
								}
							default:
								i, err := strconv.ParseUint(exp.Value, 10, 32)
								if err != nil {
									t.Fatal(err)
								}
								if v != uint32(i) {
									t.Errorf("got %d, want %d", v, uint32(i))
								}
							}
						case "f64":
							f := res[i].Interface().(float64)
							v := math.Float64bits(f)
							switch exp.Value {
							case "nan:canonical":
								if !testCanonical() && math.IsNaN(f) {
									t.Logf("got %x, want nan:canonical", v)
									break
								}
								if v != 0xfff8000000000000 && v != 0x7ff8000000000000 {
									t.Errorf("got %x, want nan:canonical", v)
								}
							case "nan:arithmetic":
								if !testCanonical() && math.IsNaN(f) {
									t.Logf("got %x, want nan:arithmetic", v)
									break
								}
								if v&0x7ff8000000000000 != 0x7ff8000000000000 {
									t.Errorf("got %x, want nan:arithmetic", v)
								}
							default:
								i, err := strconv.ParseUint(exp.Value, 10, 64)
								if err != nil {
									t.Fatal(err)
								}
								if v != uint64(i) {
									t.Errorf("got %d, want %d", v, uint64(i))
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

// We skip float tests on MIPS, due to inaccuracy.
func skipFloat(name string) bool {
	return strings.HasPrefix(runtime.GOARCH, "mips") &&
		(strings.Contains(name, "float") ||
			strings.Contains(name, "f32") ||
			strings.Contains(name, "f64"))
}

// We only check for canonical NaNs on amd64 and arm64.
func testCanonical() bool {
	switch runtime.GOARCH {
	case "amd64", "arm64":
		return true
	}
	return false
}
