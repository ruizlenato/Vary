//go:build windows

package executil

import (
	"os/exec"
	"syscall"
)

func ConfigureCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
