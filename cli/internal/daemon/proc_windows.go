//go:build windows

package daemon

import (
	"os"
	"syscall"
)

const daemonSupported = false

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p != nil
}

func signalStop(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}

func signalKill(pid int) error {
	return signalStop(pid)
}

func killGroup(pgid int) error {
	if pgid <= 0 {
		return nil
	}
	p, err := os.FindProcess(pgid)
	if err != nil {
		return err
	}
	return p.Kill()
}

func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
