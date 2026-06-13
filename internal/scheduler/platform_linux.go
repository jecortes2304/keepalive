//go:build linux

package scheduler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const systemdUnit = "keepalive.service"

const unitTemplate = `[Unit]
Description=Keepalive daemon - keep machine active
After=graphical-session.target

[Service]
Type=simple
ExecStart={{.BinaryPath}} daemon
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`

type unitData struct {
	BinaryPath string
}

func systemdUnitPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", systemdUnit)
}

func Install(binaryPath string) error {
	data := unitData{BinaryPath: binaryPath}

	tmpl, err := template.New("unit").Parse(unitTemplate)
	if err != nil {
		return err
	}

	unitPath := systemdUnitPath()
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		return err
	}

	f, err := os.Create(unitPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return err
	}

	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("reloading systemd: %w", err)
	}
	return exec.Command("systemctl", "--user", "enable", "--now", systemdUnit).Run()
}

func Uninstall() error {
	exec.Command("systemctl", "--user", "disable", "--now", systemdUnit).Run()
	return os.Remove(systemdUnitPath())
}
