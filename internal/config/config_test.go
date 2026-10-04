package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AgueroZzz/Anti2Verge/internal/config"
)

func TestEnsureConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "custom.yaml")
	cfg, err := config.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	created, err := cfg.EnsureConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if created != path {
		t.Fatalf("opened %s, wanted %s", created, path)
	}
	loaded, err := config.LoadConfig(created)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Clash.FallbackHTTPPort != 7897 {
		t.Fatal("default port not saved")
	}
	content := []byte("# user comment\nclash:\n  fallback_http_port: 9999\n")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.EnsureConfigFile(); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(content) {
		t.Fatal("existing configuration overwritten")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig 不应为空")
	}

	if cfg.Clash.FallbackHTTPPort != 7897 {
		t.Errorf("期望默认 FallbackHTTPPort 为 7897，实际为 %d", cfg.Clash.FallbackHTTPPort)
	}

	if !cfg.Clash.AutoDetect {
		t.Errorf("期望默认 AutoDetect 为 true")
	}

	if !cfg.App.MinimizeToTray {
		t.Errorf("期望默认 MinimizeToTray 为 true")
	}
}

func TestLoadConfigFallback(t *testing.T) {
	// 加载一个不存在的配置文件，应该返回默认配置而不会 panic
	cfg, err := config.LoadConfig("non_existent_config_file_xyz.yaml")
	if err != nil {
		t.Fatalf("加载不存在文件不应报错: %v", err)
	}

	if cfg.Clash.FallbackHTTPPort != 7897 {
		t.Errorf("期望 FallbackHTTPPort 为 7897")
	}
}
