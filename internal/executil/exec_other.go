//go:build !windows

package executil

import "os/exec"

func ConfigureCommand(cmd *exec.Cmd) {}
