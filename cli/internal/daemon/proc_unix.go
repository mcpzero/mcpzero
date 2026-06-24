//go:build unix

package daemon

import "syscall"

const daemonSupported = true

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Signal 0 probes existence without actually delivering a signal.
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

func signalStop(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

func signalKill(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}

// killGroup force-terminates an entire process group (negative pid), used as a
// safety net to reap a tunnel's MCP subprocess tree even if the daemon process
// was killed before it could clean up.
func killGroup(pgid int) error {
	if pgid <= 0 {
		return nil
	}
	return syscall.Kill(-pgid, syscall.SIGKILL)
}

// detachSysProcAttr starts the child in a new session so it survives
// the parent terminal closing.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
