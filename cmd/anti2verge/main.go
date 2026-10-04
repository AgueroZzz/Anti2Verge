package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"runtime/debug"
	"time"

	"github.com/AgueroZzz/Anti2Verge/internal/config"
	"github.com/AgueroZzz/Anti2Verge/internal/detector"
	"github.com/AgueroZzz/Anti2Verge/internal/resolver"
	"github.com/AgueroZzz/Anti2Verge/internal/supervisor"
	"github.com/AgueroZzz/Anti2Verge/internal/tray"
	"github.com/AgueroZzz/Anti2Verge/pkg/winutil"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	version = "1.0.0"
)

func logToFile(msg string) {
	f, err := os.OpenFile("anti2verge.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer f.Close()
		_, _ = f.WriteString(fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg))
	}
}

func main() {
	// The default command is a tray application; Explorer launches must proceed.
	cobra.MousetrapHelpText = ""
	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("Anti2Verge 发生崩溃: %v\n堆栈:\n%s", r, string(debug.Stack()))
			logToFile(errStr)
			winutil.ShowMessageBox("Anti2Verge 运行异常", errStr, winutil.MB_ICONERROR)
		}
	}()

	logToFile("Anti2Verge 进程已启动，命令行参数: " + fmt.Sprint(os.Args))

	rootCmd := &cobra.Command{
		Use:   "anti2verge",
		Short: "Anti2Verge: 面向 Google Antigravity 的专用无感代理桥接与进程守护工具",
		Run: func(cmd *cobra.Command, args []string) {
			logToFile("进入 rootCmd.Run，正在加载配置...")
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				logToFile(fmt.Sprintf("加载配置警告: %v", err))
				cfg = config.DefaultConfig()
			}

			sup := supervisor.NewSupervisor(cfg)

			logToFile("正在启动系统托盘...")
			tray.NewTrayApp(cfg, sup).Run()
		},
	}

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "配置文件路径 (例如: config.yaml)")

	// run 子命令
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "直接注入代理并启动 Antigravity 进程",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				fmt.Printf("加载配置警告: %v\n", err)
				cfg = config.DefaultConfig()
			}
			sup := supervisor.NewSupervisor(cfg)
			runForeground(cfg, sup)
		},
	}

	// check 子命令
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "检查 Antigravity 安装路径及 Clash Verge 代理端口连通性",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				cfg = config.DefaultConfig()
			}
			runCheck(cfg)
		},
	}

	// tray 子命令
	trayCmd := &cobra.Command{
		Use:   "tray",
		Short: "启动系统托盘常驻模式",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				cfg = config.DefaultConfig()
			}
			sup := supervisor.NewSupervisor(cfg)
			trayApp := tray.NewTrayApp(cfg, sup)
			trayApp.Run()
		},
	}

	// version 子命令
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "输出版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Anti2Verge v%s (Google Antigravity & Clash Verge Bridge)\n", version)
		},
	}

	rootCmd.AddCommand(runCmd, checkCmd, trayCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		logToFile(fmt.Sprintf("命令执行失败: %v", err))
		winutil.ShowMessageBox("Anti2Verge 启动失败", err.Error(), winutil.MB_ICONERROR)
		os.Exit(1)
	}
}

func runCheck(cfg *config.Config) {
	fmt.Println("==================================================")
	fmt.Println("  Anti2Verge 诊断检测工具")
	fmt.Println("==================================================")

	// 1. 检查 Antigravity 路径
	res := resolver.NewResolver(cfg.Antigravity.ExecutablePath)
	appPath, err := res.Resolve()
	if err != nil {
		fmt.Printf("[❌] Antigravity 安装路径: 未找到 (%v)\n", err)
	} else {
		fmt.Printf("[✅] Antigravity 安装路径: %s\n", appPath)
	}

	// 2. 检查 Clash Verge / Mihomo
	det := detector.NewClashDetector(&cfg.Clash)
	proxyInfo, err := det.Detect()
	if err != nil {
		fmt.Printf("[❌] Clash 代理探针: 失败 (%v)\n", err)
		fmt.Printf("    当前备用端口: HTTP=%d, SOCKS=%d\n", cfg.Clash.FallbackHTTPPort, cfg.Clash.FallbackSOCKSPort)
	} else {
		fmt.Printf("[✅] Clash 代理状态: 正常在线\n")
		fmt.Printf("    - 代理端口: 127.0.0.1:%d\n", proxyInfo.HTTPPort)
		if proxyInfo.Controller != "" {
			fmt.Printf("    - 控制器端点: %s\n", proxyInfo.Controller)
		}
		if proxyInfo.Mode != "" {
			fmt.Printf("    - 分流模式: %s\n", proxyInfo.Mode)
		}
		if proxyInfo.CoreType != "" {
			fmt.Printf("    - 内核版本: %s\n", proxyInfo.CoreType)
		}
	}
	fmt.Println("==================================================")
}

func runForeground(cfg *config.Config, sup *supervisor.Supervisor) {
	fmt.Println("[Anti2Verge] 正在准备拉起 Antigravity 并注入代理环境...")
	if err := sup.Start(); err != nil {
		fmt.Printf("[Anti2Verge] 启动失败: %v\n", err)
		os.Exit(1)
	}

	_, pid, port, _ := sup.GetStatus()
	fmt.Printf("[Anti2Verge] 启动成功！Antigravity PID: %d, 代理端口: %d\n", pid, port)
	fmt.Println("[Anti2Verge] 进程守护中，按 Ctrl+C 退出守护...")

	// 监听中断退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	sup.AddListener(func(state supervisor.ProcessState, pid int, proxyPort int, errMsg string) {
		if state == supervisor.StateStopped {
			fmt.Println("[Anti2Verge] Antigravity 已退出。")
			os.Exit(0)
		}
	})

	<-sigCh
	fmt.Println("\n[Anti2Verge] 收到退出信号，正在退出...")
}
