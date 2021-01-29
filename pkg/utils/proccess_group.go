// +build !windows

package utils

import (
	"os/exec"
	"syscall"
)

type ProcessExitGroup struct {
	cmds []*exec.Cmd
}

func NewProcessExitGroup() (ProcessExitGroup, error) {
	return ProcessExitGroup{}, nil
}

func (g *ProcessExitGroup) Dispose() error {
	for _, c := range g.cmds {
		if err := syscall.Kill(-c.Process.Pid, syscall.SIGKILL); err != nil {
			return err
		}
	}
	return nil
}

func (g *ProcessExitGroup) AddProcess(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	g.cmds = append(g.cmds, cmd)
	return nil
}
