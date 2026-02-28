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

func Test(t *testing.T, modptr any, data []byte, name ...string) {
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
		if len(name) > 0 && name[0] != file {
			continue
		}
		switch cmd.Type {
		case "module":
			file = cmd.Filename
		case "assert_return", "assert_trap":
			t.Run(fmt.Sprintf("line_%d", cmd.Line), func(t *testing.T) {
				if cmd.Type == "assert_trap" {
					defer func() {
						r := recover()
						if r == nil {
							t.Errorf("expected trap: %s", cmd.Text)
						} else if !strings.Contains(fmt.Sprint(r), cmd.Text) {
							t.Errorf("got trap %q, want %q", r, cmd.Text)
						}
					}()
				}

				method := mod.MethodByName(exported(cmd.Action.Field))
				args := make([]reflect.Value, len(cmd.Action.Args))
				for i, arg := range cmd.Action.Args {
					switch arg.Type {
					case "i32":
						v, _ := strconv.ParseUint(arg.Value, 10, 32)
						args[i] = reflect.ValueOf(int32(v))
					case "i64":
						v, _ := strconv.ParseUint(arg.Value, 10, 64)
						args[i] = reflect.ValueOf(int64(v))
					case "f32":
						v, _ := strconv.ParseUint(arg.Value, 10, 32)
						args[i] = reflect.ValueOf(math.Float32frombits(uint32(v)))
					case "f64":
						v, _ := strconv.ParseUint(arg.Value, 10, 64)
						args[i] = reflect.ValueOf(math.Float64frombits(uint64(v)))
					}
				}

				res := method.Call(args)
				if cmd.Type == "assert_return" {
					for i, exp := range cmd.Expected {
						switch exp.Type {
						case "i32":
							v, _ := strconv.ParseUint(exp.Value, 10, 32)
							if got, want := res[i].Interface().(int32), int32(v); got != want {
								t.Errorf("got %d, want %d", got, want)
							}
						case "i64":
							v, _ := strconv.ParseUint(exp.Value, 10, 64)
							if got, want := res[i].Interface().(int64), int64(v); got != want {
								t.Errorf("got %d, want %d", got, want)
							}
						case "f32":
							f := res[i].Interface().(float32)
							if strings.Contains(exp.Value, "nan") {
								if f == f {
									t.Errorf("got %f, want NaN", f)
								}
							} else {
								v, _ := strconv.ParseUint(exp.Value, 10, 32)
								if got, want := math.Float32bits(f), uint32(v); got != want {
									t.Errorf("got %d, want %d", got, want)
								}
							}
						case "f64":
							f := res[i].Interface().(float64)
							if strings.Contains(exp.Value, "nan") {
								if f == f {
									t.Errorf("got %f, want NaN", f)
								}
							} else {
								v, _ := strconv.ParseUint(exp.Value, 10, 64)
								if got, want := math.Float64bits(f), uint64(v); got != want {
									t.Errorf("got %d, want %d", got, want)
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
