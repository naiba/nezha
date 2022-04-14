//go:build windows

package processgroup

import (
	"fmt"
	"os/exec"
)

type ProcessExitGroup struct {
	cmds []*exec.Cmd
}

func NewProcessExitGroup() (ProcessExitGroup, error) {
	return ProcessExitGroup{}, nil
}

func (g *ProcessExitGroup) Dispose() error {
	for _, c := range g.cmds {
		if err := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(c.Process.Pid)).Run(); err != nil {
			return err
		}
	}
	return nil
}

func (g *ProcessExitGroup) AddProcess(cmd *exec.Cmd) error {
	g.cmds = append(g.cmds, cmd)
	return nil
}
