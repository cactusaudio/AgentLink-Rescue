package verifier

import (
	"os"
	"strings"
)

func envKeyPresent(ctx Context, args map[string]string) Result {
	_ = ctx
	key := args["env"]
	if key == "" {
		key = args["key"]
	}
	if key == "" {
		key = "DEEPSEEK_API_KEY"
	}
	val := os.Getenv(key)
	if val == "" {
		if strings.EqualFold(args["required"], "true") {
			return fail(key + " missing")
		}
		return warn(key + " missing")
	}
	return pass(key + " present: REDACTED")
}

func noSecretLeak(ctx Context, args map[string]string) Result {
	text := args["text"]
	for _, key := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "DEEPSEEK_API_KEY", "OPENROUTER_API_KEY"} {
		val := os.Getenv(key)
		if val != "" && strings.Contains(text, val) {
			return fail("report leaks " + key)
		}
	}
	return pass("no secret leak detected")
}

func fileNotContainsEnvValue(ctx Context, args map[string]string) Result {
	path, text, err := readFile(ctx, args)
	if err != nil {
		return fail(err.Error())
	}
	for _, key := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "DEEPSEEK_API_KEY", "OPENROUTER_API_KEY"} {
		val := os.Getenv(key)
		if val != "" && strings.Contains(text, val) {
			return fail(path + " contains " + key + " value")
		}
	}
	return pass(path + " does not contain key values")
}
