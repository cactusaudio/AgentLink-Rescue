package rules

import "testing"

func TestMatchRule(t *testing.T) {
	rule := AgentRule{Patterns: []string{"*clash*", "io.github.clashverge.helper"}}
	if !MatchRule(rule, "/Library/PrivilegedHelperTools/io.github.clashverge.helper") {
		t.Fatal("expected helper match")
	}
	if MatchRule(rule, "/Library/PrivilegedHelperTools/com.example.clean") {
		t.Fatal("unexpected match")
	}
}
