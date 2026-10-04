//go:build !windows

package winutil

import "fmt"

const (
	MB_OK          = 0
	MB_ICONERROR   = 0
	MB_ICONWARNING = 0
	MB_ICONINFO    = 0
)

func ShowMessageBox(title, message string, uType uint32) {
	fmt.Printf("[%s] %s\n", title, message)
}
