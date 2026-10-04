//go:build windows

package launcher

import (
	"os/exec"

	"golang.org/x/sys/windows"
)

func setupProcessAttributes(cmd *exec.Cmd) {
	// 在 Windows 下设置独立进程组与脱钩，避免命令行黑框闪烁或父进程退出导致子进程被直接杀死
	cmd.SysProcAttr = &windows.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    false,
	}
}

// KillProcessByPID 杀死指定 PID 的进程
func KillProcessByPID(pid int) error {
	p, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(p)

	return windows.TerminateProcess(p, 1)
}

// IsProcessAlive 检查指定 PID 的进程是否存活
func IsProcessAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(h, &exitCode); err != nil {
		return false
	}
	return exitCode == 259 // STILL_ACTIVE = 259
}
