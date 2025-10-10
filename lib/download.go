package lib

import (
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func IsHTTPURL(s string) bool {
	ls := strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(ls, "http://") || strings.HasPrefix(ls, "https://")
}

// DownloadToTemp 下载 URL 到临时文件，返回路径与清理函数
func DownloadToTemp(raw string) (string, func(), error) {
	u, err := neturl.Parse(raw)
	if err != nil {
		return "", nil, fmt.Errorf("invalid url: %w", err)
	}
	// 推断文件名与扩展名
	base := path.Base(u.Path)
	if base == "." || base == "/" || base == "" {
		base = fmt.Sprintf("download-%d", time.Now().UnixNano())
	}
	ext := filepath.Ext(base)
	if ext == "" {
		ext = ".bin"
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("bizy-cover-%d%s", time.Now().UnixNano(), ext))

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(u.String())
	if err != nil {
		return "", nil, fmt.Errorf("http get failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("download failed: %s", resp.Status)
	}

	f, err := os.Create(tmp)
	if err != nil {
		return "", nil, fmt.Errorf("create temp file failed: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(tmp)
		return "", nil, fmt.Errorf("write temp file failed: %w", err)
	}
	return tmp, func() { _ = os.Remove(tmp) }, nil
}


