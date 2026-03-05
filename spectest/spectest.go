package spectest

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

func Test(t *testing.T, modptr any, data []byte, name string) {
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
			t.Run(fmt.Sprintf("line_%d", cmd.Line), func(t *testing.T) {
				if cmd.Type == "assert_trap" {
					defer func() {
						want := cmd.Text
						switch want {
						case "out of bounds memory access", "out of bounds table access", "undefined element":
							want = "out of range"
						case "indirect call type mismatch":
							want = "interface conversion"
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
							f := math.Float32bits(res[i].Interface().(float32))
							switch exp.Value {
							case "nan:canonical":
								if f != 0xffc00000 && f != 0x7fc00000 {
									t.Errorf("got %x, want nan:canonical", f)
								}
							case "nan:arithmetic":
								if f&0x7fc00000 != 0x7fc00000 {
									t.Errorf("got %x, want nan:arithmetic", f)
								}
							default:
								v, err := strconv.ParseUint(exp.Value, 10, 32)
								if err != nil {
									t.Fatal(err)
								}
								if f != uint32(v) {
									t.Errorf("got %d, want %d", f, uint32(v))
								}
							}
						case "f64":
							f := math.Float64bits(res[i].Interface().(float64))
							switch exp.Value {
							case "nan:canonical":
								if f != 0xfff8000000000000 && f != 0x7ff8000000000000 {
									t.Errorf("got %x, want nan:canonical", f)
								}
							case "nan:arithmetic":
								if f&0x7ff8000000000000 != 0x7ff8000000000000 {
									t.Errorf("got %x, want nan:arithmetic", f)
								}
							default:
								v, err := strconv.ParseUint(exp.Value, 10, 64)
								if err != nil {
									t.Fatal(err)
								}
								if f != uint64(v) {
									t.Errorf("got %d, want %d", f, uint64(v))
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
