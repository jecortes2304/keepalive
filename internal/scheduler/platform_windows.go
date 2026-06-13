//go:build windows

package scheduler

import (
	"fmt"
	"os/exec"
)

func Install(binaryPath string) error {
	cmd := exec.Command("schtasks", "/Create",
		"/SC", "ONLOGON",
		"/TN", "Keepalive",
		"/TR", fmt.Sprintf(`"%s" daemon`, binaryPath),
		"/F",
	)
	return cmd.Run()
}

func Uninstall() error {
	cmd := exec.Command("schtasks", "/Delete", "/TN", "Keepalive", "/F")
	return cmd.Run()
}
