package command

import (
	"context"
	"strings"
)

type MockRunner struct {
	Results map[string]Result
	Calls   []string
}

func (m *MockRunner) Run(_ context.Context, path string, args ...string) Result {
	key := Render(path, args...)
	m.Calls = append(m.Calls, key)
	if m.Results != nil {
		if res, ok := m.Results[key]; ok {
			res.Path = path
			res.Args = append([]string(nil), args...)
			res.Command = key
			return res
		}
		if res, ok := m.Results[path+" "+strings.Join(args, " ")]; ok {
			res.Path = path
			res.Args = append([]string(nil), args...)
			res.Command = key
			return res
		}
	}
	return Result{Path: path, Args: append([]string(nil), args...), Command: key, ExitCode: 127, Missing: true, Error: "mock result not found"}
}
