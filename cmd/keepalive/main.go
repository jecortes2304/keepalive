package main

//go:generate go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo -o resource_windows.syso

import "keepalive/internal/cmd"

func main() {
	cmd.Execute()
}
