package recipe

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func LoadFile(path string) (Recipe, error) {
	var r Recipe
	data, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	err = json.Unmarshal(data, &r)
	return r, err
}

func LoadDir(dir string) ([]Recipe, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	var out []Recipe
	for _, path := range matches {
		r, err := LoadFile(path)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}
