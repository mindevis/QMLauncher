//go:build windows

package launcher

import (
	"os/exec"
	"syscall"
)

// setCmdNoWindow hides the console window when spawning subprocesses on Windows.
func setCmdNoWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
