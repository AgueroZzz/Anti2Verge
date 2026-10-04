//go:build !windows

package gui

import (
	"fmt"

	"github.com/AgueroZzz/Anti2Verge/internal/config"
	"github.com/AgueroZzz/Anti2Verge/internal/supervisor"
)

type MainWindow struct{}

func NewMainWindow(cfg *config.Config, sup *supervisor.Supervisor) *MainWindow {
	return &MainWindow{}
}

func (w *MainWindow) Show() {
	fmt.Println("GUI 仅支持在 Windows 平台运行")
}

func ShowWindow() {}
