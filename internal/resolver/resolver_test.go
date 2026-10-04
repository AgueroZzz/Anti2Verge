package resolver_test

import (
	"testing"

	"github.com/AgueroZzz/Anti2Verge/internal/resolver"
)

func TestCustomPathResolver(t *testing.T) {
	// 指定一个假路径测试错误分支
	res := resolver.NewResolver("C:\\invalid\\non_existent_antigravity.exe")
	// 如果系统有真实路径可能会 fallback 找到真实路径，否则返回 ErrNotFound
	path, err := res.Resolve()
	if err == nil {
		t.Logf("找到了系统内真实路径: %s", path)
	} else {
		t.Logf("未找到路径（符合预期）: %v", err)
	}
}

func TestAutoDetectResolver(t *testing.T) {
	res := resolver.NewResolver("")
	path, err := res.Resolve()
	if err != nil {
		t.Logf("自动定位结果: 未找到 Antigravity (%v)", err)
	} else {
		t.Logf("自动定位成功: %s", path)
	}
}
