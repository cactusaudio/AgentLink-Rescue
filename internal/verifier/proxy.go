package verifier

import (
	"context"
	"net"
	"os"
	"strings"
	"time"

	"cactus-agentlink-rescue/internal/command"
	"cactus-agentlink-rescue/internal/system"
)

var proxyEnvKeys = []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "all_proxy", "no_proxy"}

func envProxyAbsent(ctx Context, args map[string]string) Result {
	res := runZsh(ctx, "env")
	if res.ExitCode != 0 {
		return fail(commandString(res))
	}
	for _, line := range strings.Split(res.Stdout, "\n") {
		for _, key := range proxyEnvKeys {
			if strings.HasPrefix(line, key+"=") && strings.TrimSpace(strings.TrimPrefix(line, key+"=")) != "" {
				return fail("proxy env still present: " + key)
			}
		}
	}
	return pass("proxy env absent")
}

func envProxyMatches(ctx Context, args map[string]string) Result {
	want := args["value"]
	if want == "" {
		want = "http://127.0.0.1:7897"
	}
	res := runZsh(ctx, "print -r -- ${HTTP_PROXY:-}")
	if res.ExitCode != 0 {
		return fail(commandString(res))
	}
	return resultFromBool(strings.TrimSpace(res.Stdout) == want, "HTTP_PROXY matches")
}

func localPortListening(ctx Context, args map[string]string) Result {
	port := args["port"]
	if port == "" {
		port = "7897"
	}
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, time.Second)
	if err != nil {
		return fail(err.Error())
	}
	_ = conn.Close()
	return pass("127.0.0.1:" + port + " listening")
}

func gitProxyAbsent(ctx Context, args map[string]string) Result {
	return configAbsent(ctx, verifierConfigToolPath("git"), []string{"config", "--global", "--get", args["key"]}, args["key"])
}

func gitProxyMatches(ctx Context, args map[string]string) Result {
	key := args["key"]
	if key == "" {
		key = "http.proxy"
	}
	return configMatches(ctx, verifierConfigToolPath("git"), []string{"config", "--global", "--get", key}, args["value"])
}

func npmProxyAbsent(ctx Context, args map[string]string) Result {
	key := args["key"]
	if key == "" {
		key = "proxy"
	}
	return configAbsent(ctx, verifierConfigToolPath("npm"), []string{"config", "get", key}, key)
}

func npmProxyMatches(ctx Context, args map[string]string) Result {
	key := args["key"]
	if key == "" {
		key = "proxy"
	}
	return configMatches(ctx, verifierConfigToolPath("npm"), []string{"config", "get", key}, args["value"])
}

func curlHTTPSOptional(ctx Context, args map[string]string) Result {
	target := args["url"]
	if target == "" {
		target = "https://www.apple.com/"
	}
	runner := ctx.Runner
	if runner == nil {
		runner = command.NewExecRunner()
	}
	callCtx, cancel := context.WithTimeout(ctx.Context, 8*time.Second)
	defer cancel()
	res := runner.Run(callCtx, "/usr/bin/curl", "-I", "-L", "--connect-timeout", "5", "--max-time", "8", "-sS", "-o", "/dev/null", "-w", "%{http_code}", target)
	if res.ExitCode == 0 {
		return pass("HTTPS reachable")
	}
	return warn(commandString(res))
}

func configAbsent(ctx Context, path string, args []string, key string) Result {
	if key == "" {
		key = "http.proxy"
		args[len(args)-1] = key
	}
	runner := ctx.Runner
	if runner == nil {
		runner = command.NewExecRunner()
	}
	res := runner.Run(ctx.Context, path, args...)
	val := strings.TrimSpace(res.Stdout)
	if res.ExitCode != 0 || val == "" || val == "null" || val == "undefined" {
		return pass(key + " absent")
	}
	return fail(key + " present")
}

func configMatches(ctx Context, path string, args []string, want string) Result {
	if want == "" {
		return skipped("no expected value")
	}
	runner := ctx.Runner
	if runner == nil {
		runner = command.NewExecRunner()
	}
	res := runner.Run(ctx.Context, path, args...)
	return resultFromBool(strings.TrimSpace(res.Stdout) == want, path+" config matches")
}

func currentEnv(key string) string {
	return os.Getenv(key)
}

func verifierConfigToolPath(tool string) string {
	switch tool {
	case "git":
		if path, ok := system.FindFirstExisting("/usr/bin/git", "/opt/homebrew/bin/git", "/usr/local/bin/git"); ok {
			return path
		}
		return "/usr/bin/git"
	case "npm":
		if path, ok := system.FindFirstExisting("/usr/bin/npm", "/opt/homebrew/bin/npm", "/usr/local/bin/npm"); ok {
			return path
		}
		return "/usr/bin/npm"
	default:
		return tool
	}
}
