//go:build windows

package stdio

import "os/exec"

// configureProcAttr is a no-op on Windows; process-group teardown is handled
// by the default CommandContext cancellation.
func configureProcAttr(_ *exec.Cmd) {}
