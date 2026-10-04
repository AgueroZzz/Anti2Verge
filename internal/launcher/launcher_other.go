//go:build !windows

package launcher

import (
	"os"
	"os/exec"
	"syscall"
)

func setupProcessAttributes(cmd *exec.Cmd) {
	// 非 Windows 平台设置进程组
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

func KillProcessByPID(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}

func IsProcessAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil
}
