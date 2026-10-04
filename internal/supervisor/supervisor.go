package supervisor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AgueroZzz/Anti2Verge/internal/config"
	"github.com/AgueroZzz/Anti2Verge/internal/detector"
	"github.com/AgueroZzz/Anti2Verge/internal/launcher"
	"github.com/AgueroZzz/Anti2Verge/internal/resolver"
)

// ProcessState 进程状态枚举
type ProcessState string

const (
	StateStopped ProcessState = "已停止"
	StateRunning ProcessState = "运行中"
	StateError   ProcessState = "异常"
)

// StatusListener 状态变更监听器
type StatusListener func(state ProcessState, pid int, proxyPort int, errMsg string)

// Supervisor 管理 Antigravity 的整个生命周期
type Supervisor struct {
	mu           sync.RWMutex
	cfg          *config.Config
	resolver     *resolver.Resolver
	detector     *detector.ClashDetector
	launcher     *launcher.Launcher
	currentState ProcessState
	currentPID   int
	currentPort  int
	lastError    error
	listeners    []StatusListener
	cancelWatch  context.CancelFunc
}

// NewSupervisor 创建生命周期管理器
func NewSupervisor(cfg *config.Config) *Supervisor {
	return &Supervisor{
		cfg:          cfg,
		resolver:     resolver.NewResolver(cfg.Antigravity.ExecutablePath),
		detector:     detector.NewClashDetector(&cfg.Clash),
		launcher:     launcher.NewLauncher(),
		currentState: StateStopped,
	}
}

// AddListener 注册状态变化监听器
func (s *Supervisor) AddListener(listener StatusListener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, listener)
}

func (s *Supervisor) notifyListeners() {
	s.mu.RLock()
	state := s.currentState
	pid := s.currentPID
	port := s.currentPort
	var errMsg string
	if s.lastError != nil {
		errMsg = s.lastError.Error()
	}
	listeners := append([]StatusListener(nil), s.listeners...)
	s.mu.RUnlock()

	for _, l := range listeners {
		l(state, pid, port, errMsg)
	}
}

// GetStatus 获取当前状态
func (s *Supervisor) GetStatus() (ProcessState, int, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentState, s.currentPID, s.currentPort, s.lastError
}

// StartProxyAndApp 探测代理并拉起 Antigravity
func (s *Supervisor) Start() error {
	s.mu.Lock()
	if s.currentState == StateRunning && launcher.IsProcessAlive(s.currentPID) {
		s.mu.Unlock()
		return fmt.Errorf("Antigravity 已经在运行中 (PID: %d)", s.currentPID)
	}
	s.mu.Unlock()

	// 1. 查找可执行路径
	appPath, err := s.resolver.Resolve()
	if err != nil {
		s.mu.Lock()
		s.currentState = StateError
		s.lastError = err
		s.mu.Unlock()
		s.notifyListeners()
		return err
	}

	// 2. 探测 Clash 代理端口
	proxyInfo, err := s.detector.Detect()
	if err != nil && !s.cfg.Clash.AutoDetect {
		s.mu.Lock()
		s.currentState = StateError
		s.lastError = err
		s.mu.Unlock()
		s.notifyListeners()
		return err
	}

	activePort := proxyInfo.HTTPPort
	if activePort <= 0 {
		activePort = s.cfg.Clash.FallbackHTTPPort
	}

	// 3. 构建参数并拉起
	opts := launcher.BuildLaunchOptions(s.cfg, appPath, activePort)
	proc, err := s.launcher.Launch(opts)
	if err != nil {
		s.mu.Lock()
		s.currentState = StateError
		s.lastError = err
		s.mu.Unlock()
		s.notifyListeners()
		return err
	}

	// 4. 更新状态与开启监视
	s.mu.Lock()
	s.currentState = StateRunning
	s.currentPID = proc.PID
	s.currentPort = activePort
	s.lastError = nil

	if s.cancelWatch != nil {
		s.cancelWatch()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelWatch = cancel
	s.mu.Unlock()

	s.notifyListeners()

	go s.watchProcess(ctx, proc.PID)

	return nil
}

// Stop 停止当前运行的 Antigravity 实例
func (s *Supervisor) Stop() error {
	s.mu.Lock()
	pid := s.currentPID
	if s.currentState != StateRunning || pid <= 0 {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	err := launcher.KillProcessByPID(pid)

	s.mu.Lock()
	if s.cancelWatch != nil {
		s.cancelWatch()
		s.cancelWatch = nil
	}
	s.currentState = StateStopped
	s.currentPID = 0
	s.mu.Unlock()

	s.notifyListeners()
	return err
}

// Restart 优雅重启 Antigravity
func (s *Supervisor) Restart() error {
	_ = s.Stop()
	time.Sleep(1 * time.Second)
	return s.Start()
}

func (s *Supervisor) watchProcess(ctx context.Context, pid int) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !launcher.IsProcessAlive(pid) {
				s.mu.Lock()
				if s.currentPID == pid {
					s.currentState = StateStopped
					s.currentPID = 0
				}
				s.mu.Unlock()
				s.notifyListeners()
				return
			}
		}
	}
}
