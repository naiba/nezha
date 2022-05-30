package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func exePath() (string, error) {
	prog := os.Args[0]
	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}
	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
		}
	}
	return "", err
}

func installService() error {
	return nil
}
func removeService() error {
	return nil
}
func startService() error {
	return nil
}

func controlServiceStop() error {
	return nil
}
func controlServicePause() error {
	return nil
}
func controlServiceContinue() error {
	return nil
}
func runService(isDebug bool) {
}
func runSvc() {
	run()
}
