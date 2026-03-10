//go:build !windows

package adb

import "os/exec"

func configureCommand(cmd *exec.Cmd) {}
