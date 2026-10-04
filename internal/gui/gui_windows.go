//go:build windows

package gui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/AgueroZzz/Anti2Verge/internal/config"
	"github.com/AgueroZzz/Anti2Verge/internal/detector"
	"github.com/AgueroZzz/Anti2Verge/internal/resolver"
	"github.com/AgueroZzz/Anti2Verge/internal/supervisor"
	"github.com/AgueroZzz/Anti2Verge/pkg/winutil"
	"golang.org/x/sys/windows"
)

var (
	u32               = windows.NewLazySystemDLL("user32.dll")
	g32               = windows.NewLazySystemDLL("gdi32.dll")
	k32               = windows.NewLazySystemDLL("kernel32.dll")
	pCreateWindowEx   = u32.NewProc("CreateWindowExW")
	pDefWindowProc    = u32.NewProc("DefWindowProcW")
	pRegisterClassEx  = u32.NewProc("RegisterClassExW")
	pShowWindow       = u32.NewProc("ShowWindow")
	pUpdateWindow     = u32.NewProc("UpdateWindow")
	pGetMessage       = u32.NewProc("GetMessageW")
	pTranslateMessage = u32.NewProc("TranslateMessage")
	pDispatchMessage  = u32.NewProc("DispatchMessageW")
	pPostQuitMessage  = u32.NewProc("PostQuitMessage")
	pSendMessage      = u32.NewProc("SendMessageW")
	pSetWindowText    = u32.NewProc("SetWindowTextW")
	pGetModuleHandle  = k32.NewProc("GetModuleHandleW")
	pLoadIcon         = u32.NewProc("LoadIconW")
	pLoadCursor       = u32.NewProc("LoadCursorW")
	pGetStockObject   = g32.NewProc("GetStockObject")
	pSetForegroundWin = u32.NewProc("SetForegroundWindow")
	pShowWindowAsync  = u32.NewProc("ShowWindow")
)

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_BORDER           = 0x00800000
	WS_VSCROLL          = 0x00200000
	ES_MULTILINE        = 0x0004
	ES_AUTOVSCROLL      = 0x0040
	ES_READONLY         = 0x0800
	BS_PUSHBUTTON       = 0x0000
	SS_LEFT             = 0x0000
	SW_SHOW             = 5
	SW_RESTORE          = 9
	WM_COMMAND          = 0x0111
	WM_DESTROY          = 0x0002
	WM_CLOSE            = 0x0010
	WM_SETFONT          = 0x0030
	DEFAULT_GUI_FONT    = 17

	// 按钮控件 ID
	ID_BTN_START   = 1001
	ID_BTN_RESTART = 1002
	ID_BTN_STOP    = 1003
	ID_BTN_CHECK   = 1004
	ID_BTN_CONFIG  = 1005
)

type MainWindow struct {
	cfg        *config.Config
	sup        *supervisor.Supervisor
	hwnd       windows.Handle
	lblStatus  windows.Handle
	lblProxy   windows.Handle
	lblPath    windows.Handle
	editLog    windows.Handle
	btnStart   windows.Handle
	btnRestart windows.Handle
	btnStop    windows.Handle
	logMu      sync.Mutex
	logBuffer  string
}

var currentWin *MainWindow

func NewMainWindow(cfg *config.Config, sup *supervisor.Supervisor) *MainWindow {
	win := &MainWindow{
		cfg: cfg,
		sup: sup,
	}
	currentWin = win
	return win
}

func (w *MainWindow) appendLog(text string) {
	w.logMu.Lock()
	timestamp := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s\r\n", timestamp, text)
	w.logBuffer += line
	full := w.logBuffer
	w.logMu.Unlock()

	if w.editLog != 0 {
		strPtr, _ := windows.UTF16PtrFromString(full)
		pSetWindowText.Call(uintptr(w.editLog), uintptr(unsafe.Pointer(strPtr)))
		// 自动滚到底部
		pSendMessage.Call(uintptr(w.editLog), 0x00B6, 0, uintptr(len(full))) // EM_LINESCROLL
	}
}

type wndClassExW struct {
	Size, Style                        uint32
	WndProc                            uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background windows.Handle
	MenuName, ClassName                *uint16
	IconSm                             windows.Handle
}

func wndProc(hWnd, msg, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case WM_CLOSE:
		if currentWin != nil && currentWin.cfg.App.MinimizeToTray {
			pShowWindowAsync.Call(hWnd, 0) // hide; tray remains active
			return 0
		}
		pPostQuitMessage.Call(0)
		return 0
	case WM_COMMAND:
		id := uint32(wParam & 0xFFFF)
		if currentWin != nil {
			go currentWin.handleButton(id)
		}
		return 0
	case WM_DESTROY:
		pPostQuitMessage.Call(0)
		return 0
	}
	res, _, _ := pDefWindowProc.Call(hWnd, msg, wParam, lParam)
	return res
}

// ShowWindow restores and foregrounds the manager window from the tray menu.
func ShowWindow() {
	if currentWin == nil || currentWin.hwnd == 0 {
		return
	}
	pShowWindowAsync.Call(uintptr(currentWin.hwnd), SW_RESTORE)
	pSetForegroundWin.Call(uintptr(currentWin.hwnd))
}

func (w *MainWindow) handleButton(id uint32) {
	switch id {
	case ID_BTN_START:
		w.appendLog("正在启动 Antigravity (带代理)...")
		if err := w.sup.Start(); err != nil {
			w.appendLog(fmt.Sprintf("启动失败: %v", err))
		}
	case ID_BTN_RESTART:
		w.appendLog("正在重启 Antigravity...")
		if err := w.sup.Restart(); err != nil {
			w.appendLog(fmt.Sprintf("重启失败: %v", err))
		}
	case ID_BTN_STOP:
		w.appendLog("正在终止 Antigravity 实例...")
		if err := w.sup.Stop(); err != nil {
			w.appendLog(fmt.Sprintf("停止失败: %v", err))
		}
	case ID_BTN_CHECK:
		w.appendLog("正在重新检测 Clash 代理与路径...")
		w.refreshStatus()
	case ID_BTN_CONFIG:
		cfgPath := config.GetConfigPath()
		w.appendLog(fmt.Sprintf("正在打开配置文件: %s", cfgPath))
		_ = exec.Command("cmd", "/c", "start", "", cfgPath).Start()
	}
}

func (w *MainWindow) refreshStatus() {
	det := detector.NewClashDetector(&w.cfg.Clash)
	proxyInfo, err := det.Detect()
	if err != nil {
		w.setLabel(w.lblProxy, fmt.Sprintf("🌐 代理状态: 未检测到 (备用端口 %d)", w.cfg.Clash.FallbackHTTPPort))
		w.appendLog(fmt.Sprintf("代理探测未就绪: %v", err))
	} else {
		w.setLabel(w.lblProxy, fmt.Sprintf("🌐 代理状态: 正常在线 (127.0.0.1:%d | %s)", proxyInfo.HTTPPort, proxyInfo.Mode))
		w.appendLog(fmt.Sprintf("已成功连接 Clash Verge: 端口 %d, 模式 %s", proxyInfo.HTTPPort, proxyInfo.Mode))
	}

	res := resolver.NewResolver(w.cfg.Antigravity.ExecutablePath)
	path, err := res.Resolve()
	if err != nil {
		w.setLabel(w.lblPath, "📁 软件路径: 未找到 Antigravity.exe")
	} else {
		w.setLabel(w.lblPath, fmt.Sprintf("📁 软件路径: %s", path))
	}
}

func (w *MainWindow) setLabel(handle windows.Handle, text string) {
	if handle != 0 {
		strPtr, _ := windows.UTF16PtrFromString(text)
		pSetWindowText.Call(uintptr(handle), uintptr(unsafe.Pointer(strPtr)))
	}
}

// Show 启动并展示主窗口，进入 Windows 消息循环
func (w *MainWindow) Show() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	writeStartupLog("GUI: 开始初始化 Windows 窗口")

	inst, _, _ := pGetModuleHandle.Call(0)
	icon, _, _ := pLoadIcon.Call(inst, uintptr(1)) // 尝试载入嵌入的 Resource ID 1 图标
	if icon == 0 {
		icon, _, _ = pLoadIcon.Call(0, 32512) // IDI_APPLICATION 回退
	}
	cursor, _, _ := pLoadCursor.Call(0, 32512) // IDC_ARROW

	className, _ := windows.UTF16PtrFromString("Anti2VergeMainWindowClass")
	windowTitle, _ := windows.UTF16PtrFromString("Anti2Verge - Antigravity 代理桥接助手")

	wcex := &wndClassExW{
		Size:       uint32(unsafe.Sizeof(wndClassExW{})),
		Style:      3, // CS_HREDRAW | CS_VREDRAW
		WndProc:    windows.NewCallback(wndProc),
		Instance:   windows.Handle(inst),
		Icon:       windows.Handle(icon),
		Cursor:     windows.Handle(cursor),
		Background: windows.Handle(16), // COLOR_BTNFACE + 1
		ClassName:  className,
		IconSm:     windows.Handle(icon),
	}
	atom, _, classErr := pRegisterClassEx.Call(uintptr(unsafe.Pointer(wcex)))
	if atom == 0 {
		writeStartupLog(fmt.Sprintf("GUI: RegisterClassExW 失败: %v", classErr))
		winutil.ShowMessageBox("Anti2Verge 启动失败", fmt.Sprintf("注册主窗口类失败：%v", classErr), winutil.MB_ICONERROR)
		return
	}

	// 创建顶层窗口 (居中)
	width := int32(540)
	height := int32(460)
	x := int32(100)
	y := int32(100)

	hwnd, _, _ := pCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		WS_OVERLAPPEDWINDOW|WS_VISIBLE,
		uintptr(x), uintptr(y),
		uintptr(width), uintptr(height),
		0, 0, inst, 0,
	)
	if hwnd == 0 {
		windowErr := windows.GetLastError()
		writeStartupLog(fmt.Sprintf("GUI: CreateWindowExW 失败: %v", windowErr))
		winutil.ShowMessageBox("Anti2Verge 启动失败", fmt.Sprintf("创建主窗口失败：%v", windowErr), winutil.MB_ICONERROR)
		return
	}
	w.hwnd = windows.Handle(hwnd)
	writeStartupLog(fmt.Sprintf("GUI: 主窗口已创建，HWND=%d", hwnd))

	// 获取默认系统 UI 字体
	font, _, _ := pGetStockObject.Call(DEFAULT_GUI_FONT)

	createChild := func(cls, txt string, style uint32, id, cx, cy, cw, ch int32) windows.Handle {
		cCls, _ := windows.UTF16PtrFromString(cls)
		cTxt, _ := windows.UTF16PtrFromString(txt)
		res, _, _ := pCreateWindowEx.Call(
			0,
			uintptr(unsafe.Pointer(cCls)),
			uintptr(unsafe.Pointer(cTxt)),
			uintptr(WS_CHILD|WS_VISIBLE|style),
			uintptr(cx), uintptr(cy),
			uintptr(cw), uintptr(ch),
			uintptr(w.hwnd),
			uintptr(id),
			inst, 0,
		)
		h := windows.Handle(res)
		pSendMessage.Call(uintptr(h), WM_SETFONT, font, 1)
		return h
	}

	// 1. 顶部状态信息标签
	createChild("STATIC", "Anti2Verge 代理桥接与进程守护器", SS_LEFT, 0, 20, 15, 480, 24)
	w.lblStatus = createChild("STATIC", "🚀 Antigravity 状态: 正在初始化...", SS_LEFT, 0, 20, 45, 480, 20)
	w.lblProxy = createChild("STATIC", "🌐 代理状态: 正在探测...", SS_LEFT, 0, 20, 68, 480, 20)
	w.lblPath = createChild("STATIC", "📁 软件路径: 正在查找...", SS_LEFT, 0, 20, 91, 480, 20)

	// 2. 操作按钮组
	w.btnStart = createChild("BUTTON", "⚡ 启动", BS_PUSHBUTTON, ID_BTN_START, 20, 125, 90, 32)
	w.btnRestart = createChild("BUTTON", "🔄 重启", BS_PUSHBUTTON, ID_BTN_RESTART, 120, 125, 90, 32)
	w.btnStop = createChild("BUTTON", "⏹ 停止", BS_PUSHBUTTON, ID_BTN_STOP, 220, 125, 90, 32)
	createChild("BUTTON", "🔍 刷新检测", BS_PUSHBUTTON, ID_BTN_CHECK, 320, 125, 90, 32)
	createChild("BUTTON", "⚙ 配置文件", BS_PUSHBUTTON, ID_BTN_CONFIG, 420, 125, 90, 32)

	// 3. 运行日志多行文本框
	createChild("STATIC", "运行日志与事件监控:", SS_LEFT, 0, 20, 175, 480, 18)
	w.editLog = createChild("EDIT", "", WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, 0, 20, 198, 485, 200)

	// 注册 supervisor 状态更新
	w.sup.AddListener(func(state supervisor.ProcessState, pid int, proxyPort int, errMsg string) {
		switch state {
		case supervisor.StateRunning:
			w.setLabel(w.lblStatus, fmt.Sprintf("🚀 Antigravity 状态: 运行中 (PID %d | 代理端口 %d)", pid, proxyPort))
		case supervisor.StateStopped:
			w.setLabel(w.lblStatus, "🚀 Antigravity 状态: 已停止")
		case supervisor.StateError:
			w.setLabel(w.lblStatus, fmt.Sprintf("🚀 Antigravity 状态: 异常 (%s)", errMsg))
		}
	})

	w.appendLog("Anti2Verge 界面启动成功。")
	w.refreshStatus()

	// 自动启动 Antigravity
	if w.cfg.App.AutoLaunch {
		go func() {
			w.appendLog("正在自动拉起 Antigravity (带代理)...")
			if err := w.sup.Start(); err != nil {
				w.appendLog(fmt.Sprintf("拉起失败: %v", err))
			} else {
				_, pid, port, _ := w.sup.GetStatus()
				w.appendLog(fmt.Sprintf("拉起成功！PID: %d, 代理端口: %d", pid, port))
			}
		}()
	}

	pShowWindow.Call(uintptr(w.hwnd), SW_SHOW)
	pUpdateWindow.Call(uintptr(w.hwnd))
	pSetForegroundWin.Call(uintptr(w.hwnd))
	writeStartupLog("GUI: 窗口已显示，进入消息循环")

	// 标准 Windows 消息循环 (确保任务栏常驻响应)
	type msgStruct struct {
		Hwnd    windows.Handle
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		Pt      struct{ X, Y int32 }
	}
	var msg msgStruct

	for {
		ret, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			break
		}
		if int32(ret) == -1 {
			messageErr := windows.GetLastError()
			writeStartupLog(fmt.Sprintf("GUI: GetMessageW 失败: %v", messageErr))
			winutil.ShowMessageBox("Anti2Verge 运行异常", fmt.Sprintf("Windows 消息循环失败：%v", messageErr), winutil.MB_ICONERROR)
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func writeStartupLog(message string) {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	logPath := filepath.Join(filepath.Dir(exePath), "anti2verge.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)
}
