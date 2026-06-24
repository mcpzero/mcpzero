//go:build unix

package stdio

import (
	"os/exec"
	"syscall"
	"time"
)

// configureProcAttr starts the subprocess in its own process group and, on
// context cancellation, signals the entire group (negative pid) so that the
// MCP server and any processes it spawned are all terminated.
func configureProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 5 * time.Second
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// With Setpgid, the child's pgid equals its pid.
		pgid := cmd.Process.Pid
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
		go func(pgid int) {
			time.Sleep(2 * time.Second)
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		}(pgid)
		return nil
	}
}
