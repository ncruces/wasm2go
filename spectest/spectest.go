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
	"unicode"
)

func Test(t *testing.T, modptr any, data []byte, name string) {
	if strings.HasPrefix(runtime.GOARCH, "mips") && isfloat(name) {
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
					defer func() {
						want := cmd.Text
						switch {
						case strings.Contains(want, "out of bounds") || want == "undefined element":
							want = "out of range"
						case strings.Contains(want, "type mismatch"):
							want = "interface conversion"
						case strings.Contains(want, "uninitialized element"):
							want = "is nil"
						}
						if r := recover(); r == nil {
							t.Errorf("expected trap: %s", cmd.Text)
						} else if !strings.Contains(fmt.Sprint(r), want) {
							t.Errorf("got trap %q, want %q", r, cmd.Text)
						}
					}()
				}

				method := mod.MethodByName(exported(cmd.Action.Field))
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
								if !canonical() && math.IsNaN(float64(f)) {
									t.Logf("got %x, want nan:canonical", v)
									break
								}
								if v != 0xffc00000 && v != 0x7fc00000 {
									t.Errorf("got %x, want nan:canonical", v)
								}
							case "nan:arithmetic":
								if !canonical() && math.IsNaN(float64(f)) {
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
								if !canonical() && math.IsNaN(f) {
									t.Logf("got %x, want nan:canonical", v)
									break
								}
								if v != 0xfff8000000000000 && v != 0x7ff8000000000000 {
									t.Errorf("got %x, want nan:canonical", v)
								}
							case "nan:arithmetic":
								if !canonical() && math.IsNaN(f) {
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

func exported(name string) string {
	var buf strings.Builder
	buf.WriteByte('X')
	mangle(&buf, name)
	return buf.String()
}

func mangle(buf *strings.Builder, name string) {
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			r = '_'
		}
		buf.WriteRune(r)
	}
}

func isfloat(name string) bool {
	return strings.Contains(name, "float") ||
		strings.Contains(name, "f32") ||
		strings.Contains(name, "f64")
}

func canonical() bool {
	switch runtime.GOARCH {
	case "amd64", "arm64":
		return true
	}
	return false
}
