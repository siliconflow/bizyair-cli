package lib

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

const (
	maxRemoteCoverSize   = int64(100 * 1024 * 1024)
	coverDownloadTimeout = 2 * time.Minute
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
	return DownloadToTempContext(context.Background(), url)
}

// DownloadToTempContext downloads a remote cover with cancellation, a bounded
// request duration, and a hard size limit enforced while streaming.
func DownloadToTempContext(ctx context.Context, urlStr string) (string, func(), error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", nil, i18n.NewError("error.network.invalid_url", map[string]any{"URL": urlStr}, err)
	}
	ext := filepath.Ext(parsedURL.Path)
	if ext == "" {
		ext = ".tmp"
	}

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "cover-*"+ext)
	if err != nil {
		return "", nil, i18n.NewError("error.io.temp_file_failed", nil, err)
	}
	tmpPath := tmpFile.Name()

	cleanup := func() {
		os.Remove(tmpPath)
	}

	// 下载文件
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		tmpFile.Close()
		cleanup()
		return "", nil, i18n.NewError("error.download.failed", map[string]any{"URL": urlStr}, err)
	}
	client := &http.Client{Timeout: coverDownloadTimeout}
	resp, err := client.Do(req)
	if err != nil {
		tmpFile.Close()
		cleanup()
		return "", nil, i18n.NewError("error.download.failed", map[string]any{"URL": urlStr}, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tmpFile.Close()
		cleanup()
		return "", nil, i18n.NewError("error.download.http_status", map[string]any{"Status": resp.StatusCode, "URL": urlStr}, nil)
	}
	if resp.ContentLength > maxRemoteCoverSize {
		tmpFile.Close()
		cleanup()
		return "", nil, i18n.NewError("validation.cover_too_large", map[string]any{
			"Size": fmt.Sprintf("%.1f", float64(resp.ContentLength)/(1024*1024)), "Limit": 100,
		}, nil)
	}

	// 写入临时文件
	written, err := io.Copy(tmpFile, io.LimitReader(resp.Body, maxRemoteCoverSize+1))
	tmpFile.Close()
	if err != nil {
		cleanup()
		return "", nil, i18n.NewError("error.io.write_failed", map[string]any{"Path": tmpPath}, err)
	}
	if written > maxRemoteCoverSize {
		cleanup()
		return "", nil, i18n.NewError("validation.cover_too_large", map[string]any{
			"Size": fmt.Sprintf(">%.1f", float64(maxRemoteCoverSize)/(1024*1024)), "Limit": 100,
		}, nil)
	}

	return tmpPath, cleanup, nil
}
