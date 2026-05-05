//go:build darwin

package brain

import (
	"os/exec"
	"syscall"
)

func detachServer(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
