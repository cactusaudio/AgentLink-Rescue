package verifier

import (
	"context"
	"os"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/configfile"
)

func shellLoginPathContains(ctx Context, args map[string]string) Result {
	target := args["path"]
	if target == "" {
		return fail("path argument missing")
	}
	res := runZsh(ctx, "print -r -- $PATH")
	if res.ExitCode != 0 {
		return fail(commandString(res))
	}
	for _, part := range strings.Split(strings.TrimSpace(res.Stdout), ":") {
		if part == target {
			return pass("login PATH contains " + target)
		}
	}
	return fail("login PATH does not contain " + target)
}

func commandExistsInLoginShell(ctx Context, args map[string]string) Result {
	name := args["command"]
	if name == "" {
		return fail("command argument missing")
	}
	res := runZsh(ctx, "command -v -- "+quoteShell(name))
	return resultFromBool(res.ExitCode == 0, commandString(res))
}

func shellConfigNoParseError(ctx Context, args map[string]string) Result {
	path := expand(ctx, args["path"])
	if path == "" {
		path = expand(ctx, "~/.zshrc")
	}
	if _, err := os.Stat(path); err != nil {
		return warn("shell config missing: " + path)
	}
	res := runZsh(ctx, "zcompile -t "+quoteShell(path))
	if res.ExitCode == 0 {
		return pass("shell config parse OK")
	}
	return fail(commandString(res))
}

func managedBlockCount(ctx Context, args map[string]string) Result {
	path := expand(ctx, args["path"])
	marker := args["marker"]
	want := args["count"]
	got := configfile.CountManagedBlocks(path, marker)
	if want == "" || want == "1" {
		if got == 1 {
			return pass("managed block count is 1")
		}
		return fail("managed block count is not 1")
	}
	return pass("managed block count checked")
}

func runZsh(ctx Context, script string) command.Result {
	runner := ctx.Runner
	if runner == nil {
		runner = command.NewExecRunner()
	}
	callCtx, cancel := context.WithTimeout(ctx.Context, 5*time.Second)
	defer cancel()
	prefix := `if [ -r "$HOME/.zshrc" ]; then . "$HOME/.zshrc"; fi; `
	return runner.Run(callCtx, "/bin/zsh", "-lc", prefix+script)
}
