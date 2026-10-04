package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AgueroZzz/Anti2Verge/internal/config"
)

// LaunchOptions 启动 Antigravity 时的选项
type LaunchOptions struct {
	ExecutablePath  string
	ProxyPort       int
	ExtraArgs       []string
	BypassTLSVerify bool
	WorkingDir      string
}

// LaunchedProcess 启动后的进程信息
type LaunchedProcess struct {
	PID int
	Cmd *exec.Cmd
}

// Launcher 负责装配环境并拉起 Antigravity
type Launcher struct{}

// NewLauncher 创建启动器实例
func NewLauncher() *Launcher {
	return &Launcher{}
}

// Launch 注入环境变量与命令行参数并启动 Antigravity
func (l *Launcher) Launch(opts *LaunchOptions) (*LaunchedProcess, error) {
	if opts.ExecutablePath == "" {
		return nil, fmt.Errorf("启动失败：未提供 Antigravity.exe 路径")
	}

	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", opts.ProxyPort)
	socksURL := fmt.Sprintf("socks5://127.0.0.1:%d", opts.ProxyPort)

	// 1. 构建命令行参数
	// Chromium / Electron 代理启动参数
	args := []string{
		fmt.Sprintf("--proxy-server=%s", proxyURL),
	}
	if len(opts.ExtraArgs) > 0 {
		args = append(args, opts.ExtraArgs...)
	}

	cmd := exec.Command(opts.ExecutablePath, args...)

	// 设置工作目录
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	} else {
		cmd.Dir = os.Getenv("USERPROFILE")
	}

	// 2. 注入专用环境变量 (不污染全局注册表)
	currentEnv := os.Environ()
	newEnvMap := make(map[string]string)

	// 遍历现有环境变量
	for _, envStr := range currentEnv {
		parts := strings.SplitN(envStr, "=", 2)
		if len(parts) == 2 {
			newEnvMap[strings.ToUpper(parts[0])] = envStr
		}
	}

	// 覆盖与代理相关的环境变量 (覆盖大写和小写形式，确保 Node.js/Python/Go/Curl 都能识别)
	proxyOverrides := map[string]string{
		"HTTP_PROXY":  fmt.Sprintf("HTTP_PROXY=%s", proxyURL),
		"http_proxy":  fmt.Sprintf("http_proxy=%s", proxyURL),
		"HTTPS_PROXY": fmt.Sprintf("HTTPS_PROXY=%s", proxyURL),
		"https_proxy": fmt.Sprintf("https_proxy=%s", proxyURL),
		"ALL_PROXY":   fmt.Sprintf("ALL_PROXY=%s", socksURL),
		"all_proxy":   fmt.Sprintf("all_proxy=%s", socksURL),
		"NO_PROXY":    "NO_PROXY=localhost,127.0.0.1,::1",
		"no_proxy":    "no_proxy=localhost,127.0.0.1,::1",
	}

	for k, v := range proxyOverrides {
		newEnvMap[strings.ToUpper(k)] = v
	}

	if opts.BypassTLSVerify {
		newEnvMap["NODE_TLS_REJECT_UNAUTHORIZED"] = "NODE_TLS_REJECT_UNAUTHORIZED=0"
	}

	finalEnv := make([]string, 0, len(newEnvMap))
	for _, v := range newEnvMap {
		finalEnv = append(finalEnv, v)
	}
	cmd.Env = finalEnv

	// 平台特定的进程创建属性设置 (Windows 下独立进程组与脱离)
	setupProcessAttributes(cmd)

	// 3. 异步启动子进程
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动 Antigravity 失败: %w", err)
	}

	return &LaunchedProcess{
		PID: cmd.Process.Pid,
		Cmd: cmd,
	}, nil
}

// BuildLaunchOptions 从配置与检测到的代理端口构建启动选项
func BuildLaunchOptions(cfg *config.Config, execPath string, proxyPort int) *LaunchOptions {
	return &LaunchOptions{
		ExecutablePath:  execPath,
		ProxyPort:       proxyPort,
		ExtraArgs:       cfg.Antigravity.ExtraArgs,
		BypassTLSVerify: cfg.Antigravity.BypassTLSVerify,
	}
}
