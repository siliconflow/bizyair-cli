package lib

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

// DownloadFileOptions 下载文件的选项
type DownloadFileOptions struct {
	URL          string
	DestPath     string
	ProgressFunc func(downloaded, total int64)
	Context      context.Context
}

// DownloadFile 下载文件到指定路径
func DownloadFile(opts DownloadFileOptions) (retErr error) {
	out, err := os.Create(opts.DestPath)
	if err != nil {
		return i18n.NewError("error.io.create_file_failed", map[string]any{"Path": opts.DestPath}, err)
	}
	return downloadFileTo(opts, out)
}

func downloadFileTo(opts DownloadFileOptions, out *os.File) (retErr error) {
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	defer func() {
		if closeErr := out.Close(); retErr == nil && closeErr != nil {
			retErr = i18n.NewError("error.io.write_failed", map[string]any{"Path": opts.DestPath}, closeErr)
		}
		if retErr != nil {
			_ = os.Remove(opts.DestPath)
		}
	}()

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "GET", opts.URL, nil)
	if err != nil {
		return i18n.NewError("error.download.request_failed", map[string]any{"URL": opts.URL}, err)
	}

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Minute, // 大文件下载，设置较长超时
	}
	resp, err := client.Do(req)
	if err != nil {
		return i18n.NewError("error.download.failed", map[string]any{"URL": opts.URL}, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return i18n.NewError("error.download.status", map[string]any{"Status": resp.Status, "URL": opts.URL}, nil)
	}

	// 获取文件总大小
	totalSize := resp.ContentLength

	// 使用带进度的 reader
	var reader io.Reader = resp.Body
	if opts.ProgressFunc != nil && totalSize > 0 {
		reader = &downloadProgressReader{
			reader:       resp.Body,
			total:        totalSize,
			progressFunc: opts.ProgressFunc,
		}
	}

	// 复制数据
	written, err := io.Copy(out, reader)
	if err != nil {
		return i18n.NewError("error.io.write_failed", map[string]any{"Path": opts.DestPath}, err)
	}

	logs.Debugf("downloaded %d bytes to %s", written, opts.DestPath)

	// 确保进度回调显示 100%
	if opts.ProgressFunc != nil && totalSize > 0 {
		opts.ProgressFunc(totalSize, totalSize)
	}

	return nil
}

// downloadProgressReader 带进度回调的 reader
type downloadProgressReader struct {
	reader       io.Reader
	total        int64
	downloaded   int64
	progressFunc func(downloaded, total int64)
	lastUpdate   time.Time
}

func (r *downloadProgressReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.downloaded += int64(n)

	// 限制更新频率（每 100ms）
	now := time.Now()
	if now.Sub(r.lastUpdate) > 100*time.Millisecond || err == io.EOF {
		r.progressFunc(r.downloaded, r.total)
		r.lastUpdate = now
	}

	return n, err
}
