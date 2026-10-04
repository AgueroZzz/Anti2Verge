package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

type AppConfig struct {
	AutoStart      bool   `mapstructure:"auto_start" yaml:"auto_start"`
	AutoLaunch     bool   `mapstructure:"auto_launch" yaml:"auto_launch"`
	MinimizeToTray bool   `mapstructure:"minimize_to_tray" yaml:"minimize_to_tray"`
	LogLevel       string `mapstructure:"log_level" yaml:"log_level"`
}

type ClashConfig struct {
	AutoDetect        bool   `mapstructure:"auto_detect" yaml:"auto_detect"`
	FallbackHTTPPort  int    `mapstructure:"fallback_http_port" yaml:"fallback_http_port"`
	FallbackSOCKSPort int    `mapstructure:"fallback_socks_port" yaml:"fallback_socks_port"`
	ProbePorts        []int  `mapstructure:"probe_ports" yaml:"probe_ports"`
	APIEndpoint       string `mapstructure:"api_endpoint" yaml:"api_endpoint"`
	APISecret         string `mapstructure:"api_secret" yaml:"api_secret"`
}

type AntigravityConfig struct {
	ExecutablePath  string   `mapstructure:"executable_path" yaml:"executable_path"`
	ExtraArgs       []string `mapstructure:"extra_args" yaml:"extra_args"`
	BypassTLSVerify bool     `mapstructure:"bypass_tls_verify" yaml:"bypass_tls_verify"`
}

type Config struct {
	filePath    string
	App         AppConfig         `mapstructure:"app" yaml:"app"`
	Clash       ClashConfig       `mapstructure:"clash" yaml:"clash"`
	Antigravity AntigravityConfig `mapstructure:"antigravity" yaml:"antigravity"`
}

// DefaultConfig 返回预设的默认配置
func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			AutoStart:      false,
			AutoLaunch:     false, // 默认由用户从托盘菜单启动 Antigravity
			MinimizeToTray: true,
			LogLevel:       "info",
		},
		Clash: ClashConfig{
			AutoDetect:        true,
			FallbackHTTPPort:  7897,
			FallbackSOCKSPort: 7897,
			ProbePorts:        []int{7897, 7890, 10809, 7891},
			APIEndpoint:       "http://127.0.0.1:9097",
			APISecret:         "",
		},
		Antigravity: AntigravityConfig{
			ExecutablePath:  "",
			ExtraArgs:       []string{},
			BypassTLSVerify: false,
		},
	}
}

// GetConfigPath 返回配置文件应该被存储的位置
func GetConfigPath() string {
	paths := []string{"config.yaml", filepath.Join("configs", "config.yaml")}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "config.yaml"))
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			absolute, _ := filepath.Abs(path)
			return absolute
		}
	}

	// 其次检查 APPDATA/Anti2Verge/config.yaml
	appData := os.Getenv("APPDATA")
	if appData != "" {
		p := filepath.Join(appData, "Anti2Verge", "config.yaml")
		return p
	}

	return "config.yaml"
}

// EnsureConfigFile creates a missing configuration without overwriting edits.
func (cfg *Config) EnsureConfigFile() (string, error) {
	path := cfg.filePath
	if path == "" {
		path = GetConfigPath()
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if os.IsExist(err) {
		return path, nil
	}
	if err != nil {
		return "", err
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return "", writeErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return path, nil
}

// LoadConfig 从指定文件或默认路径加载配置
func LoadConfig(cfgFile string) (*Config, error) {
	v := viper.New()
	cfg := DefaultConfig()

	// 设置默认值
	v.SetDefault("app.auto_start", cfg.App.AutoStart)
	v.SetDefault("app.auto_launch", cfg.App.AutoLaunch)
	v.SetDefault("app.minimize_to_tray", cfg.App.MinimizeToTray)
	v.SetDefault("app.log_level", cfg.App.LogLevel)

	v.SetDefault("clash.auto_detect", cfg.Clash.AutoDetect)
	v.SetDefault("clash.fallback_http_port", cfg.Clash.FallbackHTTPPort)
	v.SetDefault("clash.fallback_socks_port", cfg.Clash.FallbackSOCKSPort)
	v.SetDefault("clash.probe_ports", cfg.Clash.ProbePorts)
	v.SetDefault("clash.api_endpoint", cfg.Clash.APIEndpoint)
	v.SetDefault("clash.api_secret", cfg.Clash.APISecret)

	v.SetDefault("antigravity.executable_path", cfg.Antigravity.ExecutablePath)
	v.SetDefault("antigravity.extra_args", cfg.Antigravity.ExtraArgs)
	v.SetDefault("antigravity.bypass_tls_verify", cfg.Antigravity.BypassTLSVerify)

	if cfgFile == "" {
		cfgFile = GetConfigPath()
	}
	path, err := filepath.Abs(cfgFile)
	if err != nil {
		return nil, err
	}
	cfg.filePath = path
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok || os.IsNotExist(err) {
			// 文件未找到则使用默认配置
			return cfg, nil
		}
		// 其他读取解析错误
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return cfg, nil
}
