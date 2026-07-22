package lib

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
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
		return "", nil, i18n.NewError("error.io.temp_file_failed", nil, err)
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
		return "", nil, i18n.NewError("error.download.failed", map[string]any{"URL": url}, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tmpFile.Close()
		cleanup()
		return "", nil, i18n.NewError("error.download.http_status", map[string]any{"Status": resp.StatusCode, "URL": url}, nil)
	}

	// 写入临时文件
	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		cleanup()
		return "", nil, i18n.NewError("error.io.write_failed", map[string]any{"Path": tmpPath}, err)
	}

	return tmpPath, cleanup, nil
}
