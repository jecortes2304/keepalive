package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.keepalive.daemon.plist")
		if _, err := os.Stat(plistPath); err == nil {
			exec.Command("launchctl", "unload", plistPath).Run()
		}
	}

	client := NewClient(socketPath)
	if err := client.SendStop(); err != nil {
		pid, running := IsRunning(pidPath)
		if !running {
			return nil
		}
		process, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return process.Signal(syscall.SIGTERM)
	}
	return nil
}
