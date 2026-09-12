package spectest

import (
	"encoding/json"
	"os"
)

type specTest struct {
	Commands []specCommand `json:"commands"`
}

type specCommand struct {
	Type     string `json:"type"`
	Line     int    `json:"line"`
	Filename string `json:"filename"`
	Action   struct {
		Field string    `json:"field"`
		Args  []specArg `json:"args"`
	} `json:"action"`
	Text     string    `json:"text"`
	Expected []specArg `json:"expected"`
}

type specArg struct {
	Type     string  `json:"type"`
	LaneType string  `json:"lane_type"`
	Value    specVal `json:"value"`
}

// specVal is a scalar value, or a list of lane values for v128.
type specVal []string

func (v *specVal) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '[' {
		return json.Unmarshal(b, (*[]string)(v))
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	*v = specVal{s}
	return nil
}

// one returns the single scalar value.
func (v specVal) one() string {
	if len(v) == 1 {
		return v[0]
	}
	return ""
}

func parseSpec(file string) (*specTest, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	spec := new(specTest)
	if err := json.NewDecoder(f).Decode(spec); err != nil {
		return nil, err
	}
	return spec, nil
}
