package configfile

import (
	"fmt"
	"os"
	"strings"
)

func TOMLBasicSyntaxOK(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return TOMLStringBasicSyntaxOK(string(data))
}

func TOMLStringBasicSyntaxOK(text string) error {
	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") || line == "[]" {
				return fmt.Errorf("line %d: malformed section", i+1)
			}
			continue
		}
		if !strings.Contains(line, "=") {
			return fmt.Errorf("line %d: expected key=value", i+1)
		}
		parts := strings.SplitN(line, "=", 2)
		if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return fmt.Errorf("line %d: malformed key=value", i+1)
		}
	}
	return nil
}
