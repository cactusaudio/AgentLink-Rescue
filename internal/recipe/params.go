package recipe

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveParams(r Recipe, supplied map[string]string) (map[string]string, error) {
	out := map[string]string{}
	allowed := map[string]ParamSpec{}
	for _, p := range r.Params {
		allowed[p.Name] = p
		if p.Default != "" {
			out[p.Name] = p.Default
		}
	}
	for k, v := range supplied {
		spec, ok := allowed[k]
		if !ok {
			return nil, fmt.Errorf("unknown param %s", k)
		}
		if len(spec.Allowed) > 0 && !stringIn(v, spec.Allowed) {
			return nil, fmt.Errorf("invalid value for param %s", k)
		}
		out[k] = v
	}
	return out, nil
}

func Interpolate(s string, home string, params map[string]string) string {
	if strings.HasPrefix(s, "~/") {
		s = filepath.Join(home, strings.TrimPrefix(s, "~/"))
	}
	if s == "~" {
		s = home
	}
	out := strings.ReplaceAll(s, "{{home}}", home)
	out = strings.ReplaceAll(out, "${HOME}", home)
	out = strings.ReplaceAll(out, "$HOME", home)
	for k, v := range params {
		out = strings.ReplaceAll(out, "{{param."+k+"}}", v)
		out = strings.ReplaceAll(out, "${"+k+"}", v)
	}
	return out
}

func ParamPairs(args []string) (map[string]string, error) {
	out := map[string]string{}
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("invalid param %q, expected key=value", arg)
		}
		out[parts[0]] = parts[1]
	}
	return out, nil
}

func HomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

func stringIn(v string, values []string) bool {
	for _, x := range values {
		if x == v {
			return true
		}
	}
	return false
}
