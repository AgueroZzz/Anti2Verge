package rulesyncer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// RuleSyncer 负责与 Clash / Mihomo 规则列表交互
type RuleSyncer struct {
	apiEndpoint string
	apiSecret   string
	httpClient  *http.Client
}

// NewRuleSyncer 创建规则同步器
func NewRuleSyncer(apiEndpoint, apiSecret string) *RuleSyncer {
	return &RuleSyncer{
		apiEndpoint: apiEndpoint,
		apiSecret:   apiSecret,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// RuleItem 规则项
type RuleItem struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
}

type rulesResponse struct {
	Rules []RuleItem `json:"rules"`
}

// GetRules 获取当前内核加载的路由规则
func (s *RuleSyncer) GetRules() ([]RuleItem, error) {
	if s.apiEndpoint == "" {
		return nil, fmt.Errorf("未配置 API 端点")
	}

	req, err := http.NewRequest("GET", s.apiEndpoint+"/rules", nil)
	if err != nil {
		return nil, err
	}
	if s.apiSecret != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiSecret)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 响应状态码: %d", resp.StatusCode)
	}

	var res rulesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return res.Rules, nil
}

// CheckAntigravityRules 检查是否存在针对 Antigravity 的分流规则
func (s *RuleSyncer) CheckAntigravityRules() (bool, error) {
	rules, err := s.GetRules()
	if err != nil {
		return false, err
	}

	for _, r := range rules {
		if r.Payload == "antigravity.exe" || r.Payload == "agy.exe" {
			return true, nil
		}
	}
	return false, nil
}

// UpdateProxyGroup 切换指定代理策略组的节点 (如果需要)
func (s *RuleSyncer) UpdateProxyGroup(groupName, selectorName string) error {
	payload, _ := json.Marshal(map[string]string{"name": selectorName})
	url := fmt.Sprintf("%s/proxies/%s", s.apiEndpoint, groupName)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiSecret != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiSecret)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("切换代理节点失败: %d", resp.StatusCode)
	}
	return nil
}
