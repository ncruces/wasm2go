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
	Type  string `json:"type"`
	Value string `json:"value"`
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
