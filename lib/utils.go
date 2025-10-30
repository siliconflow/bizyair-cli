package lib

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Throttle(fn func(x int64), wait time.Duration) func(x int64) {
	lastTime := time.Now()
	return func(x int64) {
		now := time.Now()
		if now.Sub(lastTime) >= wait {
			fn(x)
			lastTime = now
		}
	}
}

// IsHTTPURL 判断字符串是否为 HTTP/HTTPS URL
func IsHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// DownloadToTemp 下载 URL 到临时文件
// 返回临时文件路径、清理函数、错误
func DownloadToTemp(url string) (string, func(), error) {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "cover-*"+filepath.Ext(url))
	if err != nil {
		return "", nil, fmt.Errorf("创建临时文件失败: %v", err)
	}
	tmpPath := tmpFile.Name()

	cleanup := func() {
		os.Remove(tmpPath)
	}

	// 下载文件
	resp, err := http.Get(url)
	if err != nil {
		tmpFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("下载失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tmpFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	// 写入临时文件
	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("写入文件失败: %v", err)
	}

	return tmpPath, cleanup, nil
}
