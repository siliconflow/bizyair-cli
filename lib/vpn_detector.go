package lib

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// VPNDetectionResult VPN检测结果
type VPNDetectionResult struct {
	IsUsingVPN      bool   // 是否正在使用VPN
	DetectionMethod string // 检测方法
	Confidence      string // 置信度：high/medium/low
}

// DetectVPN 检测用户是否在使用VPN
func DetectVPN(ctx context.Context) *VPNDetectionResult {
	result := &VPNDetectionResult{
		IsUsingVPN: false,
		Confidence: "low",
	}

	// 方法1: 检查活跃的VPN网络接口（带流量检查）
	if hasActiveVPNInterface() {
		result.IsUsingVPN = true
		result.DetectionMethod = "网络接口检测"
		result.Confidence = "high"
		return result
	}

	// 方法2: 检查路由表是否有VPN路由
	if hasVPNRoutes() {
		result.IsUsingVPN = true
		result.DetectionMethod = "路由表检测"
		result.Confidence = "high"
		return result
	}

	// 方法3: 检查到目标服务器的连接质量（降低为辅助判断）
	if hasHighLatency(ctx) {
		result.IsUsingVPN = true
		result.DetectionMethod = "网络延迟检测"
		result.Confidence = "low"
		return result
	}

	return result
}

// hasActiveVPNInterface 检查是否存在活跃的VPN网络接口
func hasActiveVPNInterface() bool {
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}

	// 常见VPN接口名称（更精确的匹配）
	vpnInterfacePatterns := map[string]bool{
		"tun":       true,
		"tap":       true,
		"ppp":       true,
		"ipsec":     true,
		"wg":        true, // WireGuard
		"wireguard": true,
		"openvpn":   true,
		"l2tp":      true,
		"pptp":      true,
	}

	for _, iface := range interfaces {
		name := strings.ToLower(iface.Name)

		// 必须同时满足：1) 接口UP 2) 接口RUNNING 3) 有IP地址
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagRunning == 0 {
			continue
		}

		// 检查是否有IP地址（真正在使用）
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}

		// 过滤掉macOS系统自带的utun接口（通常utun0, utun1是系统用的）
		// 真正的VPN通常会使用utun2及以上，或者其他名称
		if strings.HasPrefix(name, "utun") {
			// 检查是否有非本地的IP地址
			hasNonLocalIP := false
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if !ok {
					continue
				}
				// 排除本地链路地址（169.254.x.x）和环回地址
				if !ipNet.IP.IsLoopback() && !ipNet.IP.IsLinkLocalUnicast() {
					hasNonLocalIP = true
					break
				}
			}

			// 如果utun接口有真实IP，且编号>=2，可能是VPN
			if hasNonLocalIP {
				// 提取utun后面的数字
				numStr := strings.TrimPrefix(name, "utun")
				if len(numStr) > 0 && numStr[0] >= '2' {
					return true
				}
			}
			continue
		}

		// 检查其他VPN接口名称
		for pattern := range vpnInterfacePatterns {
			if strings.Contains(name, pattern) {
				// 确认有非本地IP地址
				for _, addr := range addrs {
					ipNet, ok := addr.(*net.IPNet)
					if ok && !ipNet.IP.IsLoopback() && !ipNet.IP.IsLinkLocalUnicast() {
						return true
					}
				}
			}
		}
	}

	return false
}

// hasVPNRoutes 检查路由表是否有VPN相关的路由
func hasVPNRoutes() bool {
	// 检查默认网关是否指向VPN接口
	// 通过实际建立连接来判断使用的是哪个接口

	// 获取到目标服务器的路由
	conn, err := net.DialTimeout("tcp", "bizyair.cn:443", 3*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	// 获取本地地址
	localAddr, ok := conn.LocalAddr().(*net.TCPAddr)
	if !ok {
		return false
	}

	// 检查本地地址是否来自VPN接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// 如果本地地址匹配到VPN接口的IP
			if ipNet.IP.Equal(localAddr.IP) {
				name := strings.ToLower(iface.Name)
				// 检查是否是VPN接口
				if strings.Contains(name, "tun") ||
					strings.Contains(name, "tap") ||
					strings.Contains(name, "ppp") ||
					strings.Contains(name, "wg") {
					return true
				}
			}
		}
	}

	return false
}

// hasHighLatency 检查到目标服务器的延迟是否异常高
func hasHighLatency(ctx context.Context) bool {
	// 测试到bizyair.cn的连接延迟
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 3 * time.Second,
			}).DialContext,
		},
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "HEAD", "https://bizyair.cn", nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		// 连接失败不一定是VPN问题，可能是网络问题
		return false
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	// 提高阈值到3秒，减少误报
	return latency > 3*time.Second
}

// GetVPNInterfaceInfo 获取VPN接口详细信息（用于调试）
func GetVPNInterfaceInfo() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Sprintf("获取接口失败: %v", err)
	}

	var info strings.Builder
	info.WriteString("网络接口信息:\n")

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, _ := iface.Addrs()
		if len(addrs) == 0 {
			continue
		}

		info.WriteString(fmt.Sprintf("\n接口: %s\n", iface.Name))
		info.WriteString(fmt.Sprintf("  状态: UP=%v, RUNNING=%v\n",
			iface.Flags&net.FlagUp != 0,
			iface.Flags&net.FlagRunning != 0))
		info.WriteString("  地址:\n")
		for _, addr := range addrs {
			info.WriteString(fmt.Sprintf("    %s\n", addr.String()))
		}
	}

	return info.String()
}
