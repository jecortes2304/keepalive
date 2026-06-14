//go:build !windows

package daemon

import (
	"os"
	"os/exec"
	"syscall"
)

func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}

func terminateProcess(process *os.Process) error {
	return process.Signal(syscall.SIGTERM)
}
