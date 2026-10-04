package winutil

import (
	"fmt"
	"net"
	"time"
)

// CheckTCPPort 测试指定的 host:port 是否可连接
func CheckTCPPort(host string, port int, timeout time.Duration) bool {
	target := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
