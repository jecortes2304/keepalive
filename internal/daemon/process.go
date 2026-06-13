package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func StartDetached(binary string, args []string) (int, error) {
	cmd := exec.Command(binary, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("starting detached process: %w", err)
	}

	return cmd.Process.Pid, nil
}

func StopRunning(pidPath, socketPath string) error {
	client := NewClient(socketPath)
	if err := client.SendStop(); err != nil {
		pid, running := IsRunning(pidPath)
		if !running {
			return fmt.Errorf("no running instance found")
		}
		process, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return process.Signal(syscall.SIGTERM)
	}
	return nil
}
