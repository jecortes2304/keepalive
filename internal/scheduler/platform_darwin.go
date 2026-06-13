//go:build darwin

package scheduler

import (
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const launchdLabel = "com.keepalive.daemon"

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}/keepalive.stdout.log</string>
    <key>StandardErrorPath</key>
    <string>{{.LogPath}}/keepalive.stderr.log</string>
</dict>
</plist>
`

type plistData struct {
	Label      string
	BinaryPath string
	LogPath    string
}

func launchdPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")
}

func Install(binaryPath string) error {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, "Library", "Logs", "keepalive")
	if err := os.MkdirAll(logPath, 0o755); err != nil {
		return err
	}

	data := plistData{
		Label:      launchdLabel,
		BinaryPath: binaryPath,
		LogPath:    logPath,
	}

	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return err
	}

	plistPath := launchdPlistPath()
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return err
	}

	f, err := os.Create(plistPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return err
	}

	return runLaunchctl("load", plistPath)
}

func Uninstall() error {
	plistPath := launchdPlistPath()
	if err := runLaunchctl("unload", plistPath); err != nil {
		// ignore error if not loaded
	}
	return os.Remove(plistPath)
}

func runLaunchctl(action, path string) error {
	return exec.Command("launchctl", action, path).Run()
}
