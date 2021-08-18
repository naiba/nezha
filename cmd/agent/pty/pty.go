//go:build !windows
//+build !windows

package pty

import (
	"os"
	"os/exec"

	opty "github.com/creack/pty"
)

type Pty struct {
	tty *os.File
	cmd *exec.Cmd
}

func DownloadDependency() {
}

func Start() (*Pty, error) {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellPath = "sh"
	}
	cmd := exec.Command(shellPath)
	cmd.Env = append(os.Environ(), "TERM=xterm")
	tty, err := opty.Start(cmd)
	return &Pty{tty: tty, cmd: cmd}, err
}

func (pty *Pty) Write(p []byte) (n int, err error) {
	return pty.tty.Write(p)
}

func (pty *Pty) Read(p []byte) (n int, err error) {
	return pty.tty.Read(p)
}

func (pty *Pty) Setsize(cols, rows uint32) error {
	return opty.Setsize(pty.tty, &opty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

func (pty *Pty) Close() error {
	if err := pty.tty.Close(); err != nil {
		return err
	}
	return pty.cmd.Process.Kill()
}
