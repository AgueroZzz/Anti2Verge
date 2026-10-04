//go:build windows

package winutil

import (
	"golang.org/x/sys/windows"
)

const (
	MB_OK          = 0x00000000
	MB_ICONERROR   = 0x00000010
	MB_ICONWARNING = 0x00000030
	MB_ICONINFO    = 0x00000040
)

// ShowMessageBox 弹出标准的 Windows 系统模态提示框
func ShowMessageBox(title, message string, uType uint32) {
	pTitle, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	pMsg, err := windows.UTF16PtrFromString(message)
	if err != nil {
		return
	}
	windows.MessageBox(0, pMsg, pTitle, uType)
}
