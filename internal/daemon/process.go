package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

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
		return terminateProcess(process)
	}
	return nil
}

func startDetachedCmd(binary string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(binary, args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd, nil
}

func StartDetached(binary string, args []string) (int, error) {
	cmd, err := startDetachedCmd(binary, args)
	if err != nil {
		return 0, err
	}
	setProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("starting detached process: %w", err)
	}

	return cmd.Process.Pid, nil
}
