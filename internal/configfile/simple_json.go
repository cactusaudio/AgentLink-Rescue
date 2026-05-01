package configfile

import (
	"encoding/json"
	"os"
)

func JSONBasicParseOK(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var v any
	return json.Unmarshal(data, &v)
}
