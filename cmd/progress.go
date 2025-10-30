package cmd

import (
	"strings"
)

// 渲染简单的进度条（CLI 特有）
func renderProgressBar(percent float64) string {
	width := 20
	filled := int(percent * float64(width))
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return bar
}
