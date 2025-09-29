package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
)

// 创建上传进度回调函数
func createUploadProgressCallback(fileIndex string, fileName string) func(int64, int64) {
	return func(consumed, total int64) {
		if total > 0 {
			percent := float64(consumed) / float64(total)
			fmt.Printf("\r(%s) %s: %.1f%% (%s/%s)",
				fileIndex,
				filepath.Base(fileName),
				percent*100,
				formatBytes(consumed),
				formatBytes(total))
		}
	}
}

// 格式化字节数
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// 渲染简单的进度条
func renderProgressBar(percent float64) string {
	width := 20
	filled := int(percent * float64(width))
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return bar
}
