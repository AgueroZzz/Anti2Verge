package detector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AgueroZzz/Anti2Verge/internal/config"
	"github.com/AgueroZzz/Anti2Verge/pkg/winutil"
)

// ProxyInfo 包含检测到的代理状态
type ProxyInfo struct {
	HTTPPort   int    `json:"http_port"`
	SOCKSPort  int    `json:"socks_port"`
	IsAlive    bool   `json:"is_alive"`
	Mode       string `json:"mode"`
	CoreType   string `json:"core_type"`
	Controller string `json:"controller"`
}

// ClashDetector 负责探测和交互 Clash / Mihomo
type ClashDetector struct {
	cfg        *config.ClashConfig
	httpClient *http.Client
}

// NewClashDetector 创建探测器
func NewClashDetector(cfg *config.ClashConfig) *ClashDetector {
	return &ClashDetector{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 1 * time.Second,
		},
	}
}

// mihomoConfig 对应 /configs 接口返回的部分字段
type mihomoConfig struct {
	Port      int    `json:"port"`
	SocksPort int    `json:"socks-port"`
	MixedPort int    `json:"mixed-port"`
	Mode      string `json:"mode"`
}

type mihomoVersion struct {
	Version string `json:"version"`
}

// Detect 执行全套探测逻辑
func (d *ClashDetector) Detect() (*ProxyInfo, error) {
	info := &ProxyInfo{
		HTTPPort:  d.cfg.FallbackHTTPPort,
		SOCKSPort: d.cfg.FallbackSOCKSPort,
		IsAlive:   false,
	}

	// 1. 优先尝试通过 External Controller API 获取精确配置
	if d.cfg.APIEndpoint != "" {
		if apiInfo, err := d.probeAPI(d.cfg.APIEndpoint, d.cfg.APISecret); err == nil {
			info = apiInfo
			info.IsAlive = true
			return info, nil
		}
	}

	// 尝试常见 External Controller 端口 (9097, 9090)
	for _, endpoint := range []string{"http://127.0.0.1:9097", "http://127.0.0.1:9090"} {
		if endpoint == d.cfg.APIEndpoint {
			continue
		}
		if apiInfo, err := d.probeAPI(endpoint, d.cfg.APISecret); err == nil {
			info = apiInfo
			info.IsAlive = true
			return info, nil
		}
	}

	// 2. 如果未配置或无法连通 API，则使用 TCP 探针轮询候选代理端口
	probePorts := d.cfg.ProbePorts
	if len(probePorts) == 0 {
		probePorts = []int{7897, 7890, 10809, 7891}
	}

	for _, port := range probePorts {
		if winutil.CheckTCPPort("127.0.0.1", port, 300*time.Millisecond) {
			info.HTTPPort = port
			info.SOCKSPort = port
			info.IsAlive = true
			info.Mode = "TCP Verified"
			return info, nil
		}
	}

	// 3. 所有探测失败，返回默认备用端口，但标记未激活
	return info, fmt.Errorf("未能探测到活动的 Clash/Mihomo 代理端口 (已扫描: %v)", probePorts)
}

// probeAPI 尝试请求 Clash / Mihomo REST API
func (d *ClashDetector) probeAPI(endpoint, secret string) (*ProxyInfo, error) {
	req, err := http.NewRequest("GET", endpoint+"/configs", nil)
	if err != nil {
		return nil, err
	}
	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 响应非 200: %d", resp.StatusCode)
	}

	var conf mihomoConfig
	if err := json.NewDecoder(resp.Body).Decode(&conf); err != nil {
		return nil, err
	}

	info := &ProxyInfo{
		Controller: endpoint,
		Mode:       conf.Mode,
		IsAlive:    true,
	}

	if conf.MixedPort > 0 {
		info.HTTPPort = conf.MixedPort
		info.SOCKSPort = conf.MixedPort
	} else {
		if conf.Port > 0 {
			info.HTTPPort = conf.Port
		} else {
			info.HTTPPort = d.cfg.FallbackHTTPPort
		}
		if conf.SocksPort > 0 {
			info.SOCKSPort = conf.SocksPort
		} else {
			info.SOCKSPort = d.cfg.FallbackSOCKSPort
		}
	}

	// 获取版本
	vReq, _ := http.NewRequest("GET", endpoint+"/version", nil)
	if secret != "" {
		vReq.Header.Set("Authorization", "Bearer "+secret)
	}
	if vResp, err := d.httpClient.Do(vReq); err == nil {
		defer vResp.Body.Close()
		var ver mihomoVersion
		if json.NewDecoder(vResp.Body).Decode(&ver) == nil {
			info.CoreType = ver.Version
		}
	}

	return info, nil
}
