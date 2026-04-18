//go:build !windows

package updater

import "os/exec"

func setCmdNoWindow(cmd *exec.Cmd) {}
