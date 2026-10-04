package tray

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/AgueroZzz/Anti2Verge/assets"
	"github.com/AgueroZzz/Anti2Verge/internal/config"
	"github.com/AgueroZzz/Anti2Verge/internal/detector"
	"github.com/AgueroZzz/Anti2Verge/internal/supervisor"
	"github.com/AgueroZzz/Anti2Verge/pkg/winutil"
	"github.com/getlantern/systray"
)

// TrayApp 托盘应用主体
type TrayApp struct {
	cfg        *config.Config
	supervisor *supervisor.Supervisor
	detector   *detector.ClashDetector

	statusItem  *systray.MenuItem
	proxyItem   *systray.MenuItem
	startItem   *systray.MenuItem
	restartItem *systray.MenuItem
	stopItem    *systray.MenuItem
	checkItem   *systray.MenuItem
	configItem  *systray.MenuItem
	quitItem    *systray.MenuItem
}

// NewTrayApp 创建托盘应用
func NewTrayApp(cfg *config.Config, sup *supervisor.Supervisor) *TrayApp {
	return &TrayApp{
		cfg:        cfg,
		supervisor: sup,
		detector:   detector.NewClashDetector(&cfg.Clash),
	}
}

// Run 启动托盘消息循环
func (t *TrayApp) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *TrayApp) onReady() {
	iconData := assets.GenerateDefaultIcon()
	systray.SetIcon(iconData)
	systray.SetTitle("Anti2Verge")
	systray.SetTooltip("Anti2Verge: Antigravity 代理桥接助手")

	// 菜单构建
	t.statusItem = systray.AddMenuItem("状态: 未启动", "当前 Antigravity 运行状态")
	t.statusItem.Disable()

	t.proxyItem = systray.AddMenuItem("代理: 探测中...", "当前检测到的 Clash Verge 端口")
	t.proxyItem.Disable()

	systray.AddSeparator()

	t.startItem = systray.AddMenuItem("⚡ 启动 Antigravity (带代理)", "启动 Antigravity 并自动注入 Clash 代理")
	t.restartItem = systray.AddMenuItem("🔄 重启 Antigravity", "重启 Antigravity 实例")
	t.stopItem = systray.AddMenuItem("⏹ 停止 Antigravity", "终止当前运行的 Antigravity 实例")

	systray.AddSeparator()

	t.checkItem = systray.AddMenuItem("🔍 检查 Clash 状态", "重新检测本地 Clash Verge 运行端口与 API")
	t.configItem = systray.AddMenuItem("⚙ 打开配置文件", "在文本编辑器中打开 config.yaml")

	systray.AddSeparator()

	t.quitItem = systray.AddMenuItem("❌ 退出 Anti2Verge", "退出托盘程序")

	// 初始状态更新
	t.updateClashStatus()

	// 注册 supervisor 状态变更监听
	t.supervisor.AddListener(func(state supervisor.ProcessState, pid int, proxyPort int, errMsg string) {
		t.onSupervisorStateChange(state, pid, proxyPort, errMsg)
	})

	// 启动事件循环
	go t.eventLoop()

	// 立即根据 supervisor 当前真实状态刷新菜单
	state, pid, port, err := t.supervisor.GetStatus()
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	t.onSupervisorStateChange(state, pid, port, errMsg)
	if t.cfg.App.AutoLaunch {
		go func() { _ = t.supervisor.Start() }()
	}
}

func (t *TrayApp) onExit() {
	// 退出时的资源清理
}

func (t *TrayApp) updateClashStatus() {
	info, err := t.detector.Detect()
	if err != nil {
		t.proxyItem.SetTitle(fmt.Sprintf("代理: 未就绪 (备用 %d)", t.cfg.Clash.FallbackHTTPPort))
		systray.SetTooltip("Anti2Verge: 未检测到 Clash 代理端口")
	} else {
		modeStr := info.Mode
		if modeStr == "" {
			modeStr = "已激活"
		}
		t.proxyItem.SetTitle(fmt.Sprintf("代理: 127.0.0.1:%d [%s]", info.HTTPPort, modeStr))
		systray.SetTooltip(fmt.Sprintf("Anti2Verge: Clash 代理就绪 (端口 %d)", info.HTTPPort))
	}
}

func (t *TrayApp) onSupervisorStateChange(state supervisor.ProcessState, pid int, proxyPort int, errMsg string) {
	switch state {
	case supervisor.StateRunning:
		t.statusItem.SetTitle(fmt.Sprintf("状态: 运行中 (PID %d)", pid))
		t.startItem.Disable()
		t.restartItem.Enable()
		t.stopItem.Enable()
	case supervisor.StateStopped:
		t.statusItem.SetTitle("状态: 已停止")
		t.startItem.Enable()
		t.restartItem.Disable()
		t.stopItem.Disable()
	case supervisor.StateError:
		t.statusItem.SetTitle(fmt.Sprintf("状态: 出错 (%s)", errMsg))
		t.startItem.Enable()
		t.restartItem.Enable()
		t.stopItem.Disable()
	}
}

func (t *TrayApp) eventLoop() {
	for {
		select {
		case <-t.startItem.ClickedCh:
			go func() {
				_ = t.supervisor.Start()
			}()

		case <-t.restartItem.ClickedCh:
			go func() {
				_ = t.supervisor.Restart()
			}()

		case <-t.stopItem.ClickedCh:
			go func() {
				_ = t.supervisor.Stop()
			}()

		case <-t.checkItem.ClickedCh:
			go t.updateClashStatus()

		case <-t.configItem.ClickedCh:
			go t.openConfigFile()

		case <-t.quitItem.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func (t *TrayApp) openConfigFile() {
	cfgPath, err := t.cfg.EnsureConfigFile()
	if err != nil {
		winutil.ShowMessageBox("配置文件错误", fmt.Sprintf("无法创建配置文件：%v", err), winutil.MB_ICONERROR)
		return
	}
	var editor *exec.Cmd
	if runtime.GOOS == "windows" {
		editor = exec.Command("notepad.exe", cfgPath)
	} else {
		editor = exec.Command("xdg-open", cfgPath)
	}
	if err := editor.Run(); err != nil {
		winutil.ShowMessageBox("配置文件错误", fmt.Sprintf("无法打开 %s：%v", cfgPath, err), winutil.MB_ICONERROR)
	}
}
