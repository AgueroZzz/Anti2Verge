package main

import (
	"fmt"
	"os"

	"github.com/AgueroZzz/Anti2Verge/assets"
	"github.com/akavel/rsrc/rsrc"
)

func main() {
	icoPath := "assets/icon.ico"
	sysoPath := "cmd/anti2verge/anti2verge_windows_amd64.syso"

	// 1. 生成高分辨率多尺寸 Windows ICO 文件
	icoData := assets.GenerateDefaultIcon()
	if err := os.WriteFile(icoPath, icoData, 0644); err != nil {
		fmt.Printf("生成 %s 失败: %v\n", icoPath, err)
		os.Exit(1)
	}
	fmt.Printf("[1/2] 成功生成 %s (大小: %d 字节)\n", icoPath, len(icoData))

	// 2. 将 ICO 打包为 Windows PE 资源目标文件 (.syso)
	if err := rsrc.Embed(sysoPath, "amd64", "", icoPath); err != nil {
		fmt.Printf("生成 %s 失败: %v\n", sysoPath, err)
		os.Exit(1)
	}

	info, err := os.Stat(sysoPath)
	if err != nil {
		fmt.Printf("检查 %s 失败: %v\n", sysoPath, err)
		os.Exit(1)
	}
	fmt.Printf("[2/2] 成功嵌入并生成 %s (大小: %d 字节)\n", sysoPath, info.Size())
}
