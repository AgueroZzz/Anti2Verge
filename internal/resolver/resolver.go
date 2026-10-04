package resolver

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/AgueroZzz/Anti2Verge/pkg/winutil"
)

// ErrNotFound 表示未能在系统中找到 Antigravity 可执行文件
var ErrNotFound = errors.New("未能自动定位到 Antigravity.exe，请在配置文件中手动指定路径")

// Resolver 负责解析 Antigravity 的二进制路径
type Resolver struct {
	customPath string
}

// NewResolver 创建路径解析器
func NewResolver(customPath string) *Resolver {
	return &Resolver{
		customPath: customPath,
	}
}

// Resolve 查找并返回 Antigravity.exe 的有效绝对路径
func (r *Resolver) Resolve() (string, error) {
	// 1. 如果用户显式配置了路径且文件存在，直接使用
	if r.customPath != "" {
		if fileExists(r.customPath) {
			return filepath.Abs(r.customPath)
		}
	}

	// 2. 检查常见安装目录
	candidates := getCommonCandidatePaths()
	for _, p := range candidates {
		if fileExists(p) {
			return filepath.Abs(p)
		}
	}

	// 3. 针对 Windows：查询注册表
	if runtime.GOOS == "windows" {
		if regPath, err := winutil.QueryAppPathFromRegistry("Antigravity.exe"); err == nil && fileExists(regPath) {
			return filepath.Abs(regPath)
		}
		if regPath, err := winutil.QueryAppPathFromRegistry("antigravity.exe"); err == nil && fileExists(regPath) {
			return filepath.Abs(regPath)
		}
	}

	// 4. 从系统 PATH 中查找
	if lookPath, err := exec.LookPath("Antigravity.exe"); err == nil && fileExists(lookPath) {
		return filepath.Abs(lookPath)
	}
	if lookPath, err := exec.LookPath("antigravity.exe"); err == nil && fileExists(lookPath) {
		return filepath.Abs(lookPath)
	}

	return "", ErrNotFound
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func getCommonCandidatePaths() []string {
	var list []string

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		list = append(list,
			filepath.Join(localAppData, "Programs", "antigravity", "Antigravity.exe"),
			filepath.Join(localAppData, "Programs", "antigravity", "antigravity.exe"),
			filepath.Join(localAppData, "Programs", "Antigravity", "Antigravity.exe"),
			filepath.Join(localAppData, "Programs", "Antigravity", "antigravity.exe"),
		)
	}

	programFiles := os.Getenv("ProgramFiles")
	if programFiles != "" {
		list = append(list,
			filepath.Join(programFiles, "Antigravity", "Antigravity.exe"),
			filepath.Join(programFiles, "antigravity", "antigravity.exe"),
		)
	}

	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	if programFilesX86 != "" {
		list = append(list,
			filepath.Join(programFilesX86, "Antigravity", "Antigravity.exe"),
			filepath.Join(programFilesX86, "antigravity", "antigravity.exe"),
		)
	}

	return list
}
