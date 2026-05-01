package diagnose

import "testing"

func TestPingArgsDoNotUseTTLFlag(t *testing.T) {
	args := pingArgs("1.1.1.1")
	for _, arg := range args {
		if arg == "-t" {
			t.Fatalf("ping args must not include BSD TTL flag -t: %v", args)
		}
	}
	want := []string{"-c", "1", "1.1.1.1"}
	if len(args) != len(want) {
		t.Fatalf("args=%v", args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args=%v want=%v", args, want)
		}
	}
}
