package detector_test

import (
	"testing"

	"github.com/AgueroZzz/Anti2Verge/internal/config"
	"github.com/AgueroZzz/Anti2Verge/internal/detector"
)

func TestDetector(t *testing.T) {
	cfg := &config.ClashConfig{
		AutoDetect:        true,
		FallbackHTTPPort:  7897,
		FallbackSOCKSPort: 7897,
		ProbePorts:        []int{7897, 7890, 10809},
		APIEndpoint:       "http://127.0.0.1:9097",
	}

	det := detector.NewClashDetector(cfg)
	info, err := det.Detect()

	if err != nil {
		t.Logf("Clash 探针探测失败 (可能当前未运行 Clash): %v", err)
	} else {
		t.Logf("Clash 探针探测成功: Port=%d, Alive=%v, Mode=%s", info.HTTPPort, info.IsAlive, info.Mode)
	}
}
