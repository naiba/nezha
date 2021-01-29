package utils

import (
	"errors"
	"os"
)

type ProcessExitGroup struct{}

func NewProcessExitGroup() (ProcessExitGroup, error) {
	return ProcessExitGroup{}, errors.New("not implement")
}

func (g ProcessExitGroup) Dispose() error {
	return errors.New("not implement")
}

func (g ProcessExitGroup) AddProcess(p *os.Process) error {
	return errors.New("not implement")
}
