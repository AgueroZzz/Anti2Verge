# Anti2Verge

> **Seamless proxy bridge & process supervisor for Google Antigravity and Clash Verge.**  
> 面向 Google Antigravity 的专用无感代理桥接与进程守护工具。

---

## 🌟 核心特性

- 🔍 **智能端口感知**：自动探测 Clash Verge / Mihomo 的活动代理端口（混合端口 7897 / 7890 等）及 External Controller API。
- 🎯 **进程级精准注入**：向 Antigravity 注入专有环境变量（`HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`, `NO_PROXY`）与 Chromium 启动参数（`--proxy-server`），**绝不污染 Windows 全局系统注册表与其它软件**。
- 🌐 **全链路代理覆盖**：确保 Antigravity 的 Electron 前端、Node.js 扩展宿主进程、后台 Agent 守护进程（agy）、以及内置终端（PowerShell / Git / Python）全部无缝走代理。
- 🖥️ **系统托盘与常驻守护**：轻量级常驻任务栏托盘（<10MB 内存），支持一键启动、重启 Antigravity、实时监控代理状态。
- 📦 **零依赖绿色单文件**：采用 Go 语言构建，原生编译为单 `.exe`，开箱即用。

---

## 🚀 快速使用

### 1. 命令行诊断检测
快速检测系统中的 Antigravity 安装路径及 Clash Verge 代理连接：
```powershell
.\bin\anti2verge-cli.exe check
```

### 2. 命令行启动并注入代理
直接拉起 Antigravity 并输出详细的代理注入日志：
```powershell
.\bin\anti2verge-cli.exe run
```

### 3. 系统托盘模式（日常双击推荐）
直接双击 `Anti2Verge.exe` 或 `bin\Anti2Verge.exe`，程序常驻任务栏右侧通知区域，不显示管理窗口或控制台。图标可能位于“显示隐藏的图标”面板中。

右键托盘图标可查看 Antigravity 和代理状态，以及启动、重启、停止 Antigravity、刷新检测、打开配置文件或退出 Anti2Verge。

默认不自动启动 Antigravity；设置 `app.auto_launch: true` 可在托盘就绪后自动启动。
---

## ⚙️ 配置文件说明 (`config.yaml`)

程序启动时会自动寻找当前目录下的 `config.yaml` 或 `%APPDATA%\Anti2Verge\config.yaml`。若未找到则会自动使用缺省推荐值：

```yaml
app:
  auto_start: false           # 是否开机自启
  minimize_to_tray: true      # 启动后常驻系统托盘
  log_level: "info"           # 日志级别: debug, info, warn, error

clash:
  auto_detect: true           # 自动探测 Clash Verge / Mihomo 端口
  fallback_http_port: 7897    # 默认备用 HTTP/混合 端口
  fallback_socks_port: 7897   # 默认备用 SOCKS5 端口
  probe_ports:                # 候选探测端口列表
    - 7897
    - 7890
    - 10809
    - 7891
  api_endpoint: "http://127.0.0.1:9097" # External Controller 地址
  api_secret: ""              # External Controller 密钥 (若有)

antigravity:
  executable_path: ""         # Antigravity.exe 路径，留空则启用自动探测
  extra_args: []              # 附加启动参数，例如 ["--disable-gpu"]
  bypass_tls_verify: false    # 是否禁用 Node.js TLS 校验 (仅在自签证书/MITM 抓包时开启)
```

---

## 🛠️ 项目构建指南

### 本地编译（PowerShell 一键构建）
```powershell
.\build.ps1 -Mode all
```
构建产物输出在 `bin/` 目录：
- `bin/Anti2Verge.exe`: 无控制台黑框的系统托盘版本（日常使用）
- `bin/anti2verge-cli.exe`: 带控制台输出的 CLI 版本（调试/自动化脚本）

---

## 📖 技术规范与架构设计
关于本项目详细的技术架构分解、模块职责与 Windows 底层 API 实现，请参阅：
👉 **[技术规格与设计规范文档 (TECH_SPEC.md)](./TECH_SPEC.md)**

## 许可证

本项目采用 [MIT License](LICENSE)。

