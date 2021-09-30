//go:build !windows
// +build !windows

package processgroup

import (
	"os/exec"
	"sync"
	"syscall"
)

type ProcessExitGroup struct {
	cmds []*exec.Cmd
}

func NewProcessExitGroup() (ProcessExitGroup, error) {
	return ProcessExitGroup{}, nil
}

func (g *ProcessExitGroup) killChildProcess(c *exec.Cmd) error {
	pgid, err := syscall.Getpgid(c.Process.Pid)
	if err != nil {
		// Fall-back on error. Kill the main process only.
		c.Process.Kill()
	}
	// Kill the whole process group.
	syscall.Kill(-pgid, syscall.SIGTERM)
	return c.Wait()
}

func (g *ProcessExitGroup) Dispose() []error {
	var errors []error
	mutex := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	wg.Add(len(g.cmds))
	for _, c := range g.cmds {
		go func(c *exec.Cmd) {
			defer wg.Done()
			if err := g.killChildProcess(c); err != nil {
				mutex.Lock()
				defer mutex.Unlock()
				errors = append(errors, err)
			}
		}(c)
	}
	wg.Wait()
	return errors
}

func (g *ProcessExitGroup) AddProcess(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	g.cmds = append(g.cmds, cmd)
	return nil
}
