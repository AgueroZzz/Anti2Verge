# Anti2Verge 技术路线与系统架构设计规范 (Technical Specification)

> **项目名称**：Anti2Verge  
> **定位**：面向 Google Antigravity 的专用无感代理桥接与进程守护工具（开源路线）  
> **选定路线**：路线 B —— Go 语言极客网络原生路线  
> **目标读者**：后续参与开发的 AI Agent / 核心开发者  

---

## 1. 项目背景与痛点分析

### 1.1 背景
Google Antigravity 是新一代 AI 辅助开发环境，内部包含复杂的多进程协作生态：
- **Electron / Chromium 前端界面**
- **Node.js 语言服务与扩展宿主（Extension Host）**
- **后台智能体编排引擎（Agent Daemon / agy CLI）**
- **集成终端（PowerShell / CMD / Bash）与工具链（Git, Python, Curl 等）**

### 1.2 核心痛点
许多用户在 Windows 上使用 **Clash Verge** 时，虽然开启了“系统代理”，但 Antigravity 的模型请求或子进程仍然频繁遇到网络超时。其根本原因如下：
1. **系统代理局限性**：Windows 的系统代理仅写入注册表，标准 WinINet 应用生效，但 Node.js、gRPC、Python `requests`、Git 及终端子进程默认不读取注册表代理。
2. **环境隔离性**：终端和 Agent 派生进程需要显式注入环境变量（`HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`）。
3. **Chromium 命令行缺省**：Electron 内核需要显式传递 `--proxy-server` 参数，才能确保主渲染进程与网络进程完全走指定端口。

### 1.3 Anti2Verge 的解决策略
Anti2Verge 充当 **Antigravity** 与 **Clash Verge** 之间的智能桥梁，负责：
1. 自动探测 Clash Verge 的代理端口与运行状态。
2. 精确注入进程级环境与启动参数，拉起 Antigravity 主进程。
3. 守护其衍生子进程树（Terminal、Agent、Git 等），保证流量全链路经由 Clash Verge 转发。

---

## 2. 技术栈选型规范 (路线 B)

本项目全栈采用 Go 生态进行构建，确保最终产物具备**单执行文件、零外部运行时依赖、低内存占用、启动秒开**的特性。

| 模块类别 | 选型技术 | 选型理由 |
| :--- | :--- | :--- |
| **开发语言** | Go 1.22+ | 原生网络库极其成熟，内存占用极低，编译为单 exe |
| **CLI 框架** | `github.com/spf13/cobra` | 业界标准的命令行开发框架 |
| **配置管理** | `github.com/spf13/viper` | 支持 YAML/JSON 配置文件与热加载 |
| **系统托盘** | `github.com/getlantern/systray` | 轻量级跨平台系统托盘，内存占用 < 10MB |
| **进程/系统调用** | `golang.org/x/sys/windows` | Windows 注册表读取、进程句柄监控、环境变量注入 |
| **进程树感知** | `github.com/shirou/gopsutil/v3` | 跨平台进程生命周期与资源监控 |
| **HTTP 客户端** | `github.com/go-resty/resty/v2` | 用于调用 Clash Verge 的 External Controller REST API |
| **进阶 GUI (Phase 3)** | `Wails v2/v3` (Go + Web 前端) | 当后期需要完整可视化面板时，无缝衔接现代前端框架 |

---

## 3. 系统架构设计

```
+-----------------------------------------------------------------------+
|                             Anti2Verge                                |
|                                                                       |
|  +--------------------+   +-------------------+   +----------------+  |
|  |     Tray / CLI     |   |   Config Engine   |   | Log & Monitor  |  |
|  | (systray / cobra)  |   | (viper / yaml)    |   |                |  |
|  +---------+----------+   +---------+---------+   +--------+-------+  |
|            |                        |                      |          |
|  +---------v------------------------v----------------------v-------+  |
|  |                       Core Coordinator                          |  |
|  +---------+------------------------+----------------------+-------+  |
|            |                        |                      |          |
|  +---------v----------+   +---------v---------+  +---------v-------+  |
|  |   Clash Detector   |   | Launcher/Injector |  | Rule Controller |  |
|  |  (Probe & API)     |   | (Process & Env)   |  | (Mihomo REST)   |  |
|  +---------+----------+   +---------+---------+  +---------+-------+  |
+------------|------------------------|----------------------|----------+
             |                        |                      |
             v                        v                      v
     [Clash Verge REST]      [Antigravity Process]     [Clash Rules]
      127.0.0.1:9097/7897       (Injected Env)         (Routing Table)
```

### 3.1 核心功能模块划分

#### 模块 1：Clash 探针与控制器 (`detector`)
- **功能**：
  - 探查本地常见的 Clash 代理端口（如 `7897`, `7890`, `10809` 等）。
  - 支持配置 Clash Verge 的外部控制接口（External Controller API，如 `http://127.0.0.1:9097`），并校验 Secret。
  - 获取当前活跃的代理模式（Rule, Global）与节点延迟状态。

#### 模块 2：路径解析器 (`resolver`)
- **功能**：
  - 自动在常见目录中定位 `antigravity.exe`：
    - 用户本地目录：`%LOCALAPPDATA%\Programs\antigravity\antigravity.exe`
    - 系统安装目录：`C:\Program Files\Antigravity\antigravity.exe`
    - 环境变量 `PATH`
  - 允许在配置文件中手动自定义覆盖可执行文件路径。

#### 模块 3：注入与启动引擎 (`launcher`)
- **功能**：
  - **参数拼接**：为 Antigravity 注入 Chromium 代理参数：
    ```bash
    --proxy-server="http://127.0.0.1:<Port>"
    ```
  - **环境变量注入**：在创建子进程前构建专有环境字典：
    ```env
    HTTP_PROXY=http://127.0.0.1:<Port>
    HTTPS_PROXY=http://127.0.0.1:<Port>
    ALL_PROXY=socks5://127.0.0.1:<Port>
    NO_PROXY=localhost,127.0.0.1,::1
    NODE_TLS_REJECT_UNAUTHORIZED=0  # 可选，用于自签证书兼容
    ```
  - 使用 Windows API (`CreateProcessW`) 拉起进程，使其子进程（终端、Extension Host）天然继承该网络上下文。

#### 模块 4：生命周期守护器 (`supervisor`)
- **功能**：
  - 跟踪 Antigravity 主进程 PID。
  - 当 Antigravity 退出后，自动更新托盘状态。
  - 提供“重启 Antigravity”、“一键修复代理”等快捷动作。

#### 模块 5：规则动态同步器 (`rulesyncer` - 进阶可选)
- **功能**：
  - 通过 Clash Verge REST API 动态向当前活跃配置追加规则：
    ```yaml
    PROCESS-NAME,antigravity.exe,PROXY
    PROCESS-NAME,agy.exe,PROXY
    DOMAIN-SUFFIX,google.com,PROXY
    DOMAIN-SUFFIX,googleapis.com,PROXY
    ```

---

## 4. 推荐目录结构规划 (Repository Structure)

```text
Anti2Verge/
├── cmd/
|   └── anti2verge/          # 应用程序主入口 (main.go)
├── internal/
│   ├── config/              # 配置加载与序列化 (Viper 封装)
│   ├── detector/            # Clash Verge 端口探测与 API 通信
│   ├── launcher/            # 进程拉起、参数拼接与环境变量注入
│   ├── resolver/            # Antigravity 安装路径探测
│   ├── supervisor/          # 进程生命周期守护与健康检查
│   └── tray/                # 系统托盘菜单事件与图标管理
├── pkg/
│   └── winutil/             # Windows 底层 API 辅助封装 (注册表, 环境变量)
├── configs/
│   └── config.example.yaml  # 默认配置文件模版
├── assets/                  # 托盘图标、静态资源
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 5. 配置文件设计 (`config.yaml`)

```yaml
# Anti2Verge 配置文件示例
app:
  auto_start: false           # 是否开机自启
  minimize_to_tray: true      # 启动后最小化到托盘

clash:
  auto_detect: true           # 自动探测 Clash Verge 端口
  fallback_http_port: 7897    # 默认备用 HTTP 端口
  fallback_socks_port: 7897   # 默认备用 SOCKS5 端口
  api_endpoint: "http://127.0.0.1:9097" # External Controller 地址
  api_secret: ""              # External Controller 密钥 (若有)

antigravity:
  executable_path: ""         # 留空则启用自动探测机制
  extra_args: []              # 附加启动参数
  bypass_tls_verify: false    # 是否禁用 Node.js TLS 校验 (仅在需要 MITM 时使用)
```

---

## 6. 开发里程碑与任务拆解 (Milestones)

后续 AI 或开发者应按以下顺序逐步实现，避免一步跨度过大：

### 阶段一：CLI 原型与核心注入能力 (MVP)
- [ ] 初始化 Go 项目结构与 `go.mod`。
- [ ] 实现 `resolver`：自动在 Windows 常见路径中查找 `antigravity.exe`。
- [ ] 实现 `detector`：TCP 探针快速测试 `127.0.0.1:7897` 和 `7890`。
- [ ] 实现 `launcher`：构建注入环境变量与 `--proxy-server` 启动子进程。
- [ ] CLI 验证：运行 `anti2verge run` 成功带代理启动 Antigravity。

### 阶段二：托盘常驻与守护体验 (Tray & Daemon)
- [ ] 引入 `systray`，实现 Windows 任务栏通知区小图标。
- [ ] 菜单选项：
  - “启动 Antigravity (代理模式)”
  - “Clash 连接状态: [正常/异常]”
  - “重启 Antigravity”
  - “配置设置”
  - “退出”
- [ ] 实现进程状态监听与断线告警。

### 阶段三：高级能力与发布包构建
- [ ] 编写 Inno Setup 或 GoReleaser 脚本，产出单文件无控制台窗口的 `Anti2Verge.exe` (`-ldflags="-H windowsgui"`).
- [ ] （可选）对接 Clash Verge API，实现分流规则一键下发与流量状态可视化。
- [ ] （可选）集成 Wails 打造现代化毛玻璃控制面板。

---

## 7. Windows 平台踩坑预警 (Gotchas)

1. **路径包含空格**：
   Windows 默认程序路径（如 `C:\Program Files\...`）常含空格，使用 `exec.Command` 时需严谨处理参数转义，切勿使用原生字符串直接拼接 shell 命令。
2. **隐藏 CMD 黑框**：
   打包生产版本时，Go 必须使用 `-ldflags="-H windowsgui"` 编译参数，否则启动时会伴随黑色的命令行控制台窗口。
3. **子进程脱离与杀死**：
   需要合理处理父子进程生命周期，使用 Windows Job Object 或信号通知，避免托盘退出时 Antigravity 意外被连带杀死，或反之出现孤儿僵尸进程。
4. **系统环境变量污染**：
   注入环境变量必须且仅限于 `antigravity.exe` 进程及其子代上下文（通过 `cmd.Env` 注入），**严禁**修改 Windows 全局注册表环境变量，以免影响其它宿主软件。
