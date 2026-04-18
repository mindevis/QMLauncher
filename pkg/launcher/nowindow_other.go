//go:build !windows

package launcher

import "os/exec"

func setCmdNoWindow(cmd *exec.Cmd) {}
