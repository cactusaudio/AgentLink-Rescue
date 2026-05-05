//go:build !darwin

package brain

import "os/exec"

func detachServer(cmd *exec.Cmd) {
	_ = cmd
}
