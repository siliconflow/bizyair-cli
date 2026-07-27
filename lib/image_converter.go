package lib

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

const (
	webPVersion           = "1.6.0"
	maxWebPArchiveSize    = 100 << 20
	maxWebPExecutableSize = 20 << 20
)

var webPSetupLock = make(chan struct{}, 1)

// getWebPVendorPath 获取用户可写的 WebP 工具缓存路径。
func getWebPVendorPath() (string, error) {
	platform := runtime.GOOS + "-" + runtime.GOARCH
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "bizyair", "webp", webPVersion, platform), nil
}

// ConvertImageToWebP 将图片转换为WebP格式
// 如果不需要转换（已经是webp或视频文件），返回原路径
// 返回值：转换后的文件路径、清理函数、错误
func ConvertImageToWebP(sourcePath string) (string, func(), error) {
	return ConvertImageToWebPContext(context.Background(), sourcePath)
}

func ConvertImageToWebPContext(ctx context.Context, sourcePath string) (string, func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	ext := strings.ToLower(filepath.Ext(sourcePath))

	// 需要转换的图片格式
	needsConversion := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif"

	// 如果不需要转换，直接返回原路径
	if !needsConversion {
		return sourcePath, nil, nil
	}

	// 创建临时输出文件
	tmpFile, err := os.CreateTemp("", "cover-*.webp")
	if err != nil {
		return "", nil, i18n.NewError("error.io.temp_file_failed", nil, err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close() // CWebP需要文件路径，先关闭文件

	// 定义清理函数
	cleanup := func() {
		os.Remove(tmpPath)
	}

	vendorPath, err := getWebPVendorPath()
	if err != nil {
		cleanup()
		return "", nil, i18n.NewError("error.cover.convert_failed", map[string]any{"Path": sourcePath}, err)
	}
	executable, err := webPExecutableContext(ctx, vendorPath)
	if err != nil {
		cleanup()
		return "", nil, i18n.NewError("error.cover.convert_failed", map[string]any{"Path": sourcePath}, err)
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, executable, "-q", "75", sourcePath, "-o", tmpPath)
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		cleanup()
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", nil, ctxErr
		}
		if message := strings.TrimSpace(stderr.String()); message != "" {
			err = fmt.Errorf("%w: %s", err, message)
		}
		return "", nil, i18n.NewError("error.cover.convert_failed", map[string]any{"Path": sourcePath}, err)
	}

	return tmpPath, cleanup, nil
}

func webPExecutableContext(ctx context.Context, vendorPath string) (string, error) {
	name := "cwebp"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	executable := filepath.Join(vendorPath, name)
	if _, err := os.Stat(executable); err == nil {
		return executable, nil
	}

	select {
	case webPSetupLock <- struct{}{}:
		defer func() { <-webPSetupLock }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	if _, err := os.Stat(executable); err == nil {
		return executable, nil
	}
	archiveURL, checksum, isZip, err := webPArchiveURL()
	if err != nil {
		return "", err
	}
	archivePath, err := downloadWebPArchiveContext(ctx, archiveURL, checksum, vendorPath)
	if err != nil {
		return "", err
	}
	defer os.Remove(archivePath)
	if err := extractWebPExecutableContext(ctx, archivePath, executable, name, isZip); err != nil {
		return "", err
	}
	return executable, nil
}

func webPArchiveURL() (string, string, bool, error) {
	var platform, checksum string
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/amd64":
		platform, checksum = "mac-x86-64.tar.gz", "f112dd83b420ab2a4b27d46610d9827ddf4200216023281de378647ecca31c2a"
	case "darwin/arm64":
		platform, checksum = "mac-arm64.tar.gz", "bc6bf84cc70f3f8574fba797d1e4a7dea4feebe9fa4be919f202413ea2b3b8f2"
	case "linux/amd64":
		platform, checksum = "linux-x86-64.tar.gz", "1c5ffab71efecefa0e3c23516c3a3a1dccb45cc310ae1095c6f14ae268e38067"
	case "linux/arm64":
		platform, checksum = "linux-aarch64.tar.gz", "69f5eebe203e0f3942fe37986209a1725741be19c152950a4283b376c95ec798"
	case "windows/amd64":
		platform, checksum = "windows-x64.zip", "48886f506b21f62e4661f0f4cbfca19800897c385128e8902542d29a950c93f1"
	default:
		return "", "", false, i18n.NewError("error.cover.webp_platform_unsupported", map[string]any{
			"OS": runtime.GOOS, "Arch": runtime.GOARCH,
		}, nil)
	}
	archive := fmt.Sprintf("libwebp-%s-%s", webPVersion, platform)
	return "https://storage.googleapis.com/downloads.webmproject.org/releases/webp/" + archive,
		checksum, strings.HasSuffix(archive, ".zip"), nil
}

func downloadWebPArchiveContext(ctx context.Context, url, expectedChecksum, vendorPath string) (string, error) {
	if err := os.MkdirAll(vendorPath, 0o755); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", i18n.NewError("error.cover.webp_download_failed", map[string]any{"Status": resp.Status}, nil)
	}

	file, err := os.CreateTemp(vendorPath, "libwebp-*")
	if err != nil {
		return "", err
	}
	path := file.Name()
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(resp.Body, maxWebPArchiveSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(path)
		return "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", closeErr
	}
	if written > maxWebPArchiveSize {
		_ = os.Remove(path)
		return "", i18n.NewError("error.cover.webp_archive_too_large", map[string]any{"Limit": maxWebPArchiveSize}, nil)
	}
	actualChecksum := fmt.Sprintf("%x", hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		_ = os.Remove(path)
		return "", i18n.NewError("error.cover.webp_checksum_mismatch", map[string]any{
			"Expected": expectedChecksum, "Actual": actualChecksum,
		}, nil)
	}
	return path, nil
}

func extractWebPExecutableContext(ctx context.Context, archivePath, target, name string, isZip bool) error {
	if isZip {
		archive, err := zip.OpenReader(archivePath)
		if err != nil {
			return err
		}
		defer archive.Close()
		for _, file := range archive.File {
			if filepath.Base(file.Name) != name || !file.FileInfo().Mode().IsRegular() {
				continue
			}
			reader, err := file.Open()
			if err != nil {
				return err
			}
			defer reader.Close()
			return writeWebPExecutableContext(ctx, reader, target)
		}
		return i18n.NewError("error.cover.webp_executable_missing", map[string]any{"Name": name}, nil)
	}

	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := tarReader.Next()
		if err == io.EOF {
			return i18n.NewError("error.cover.webp_executable_missing", map[string]any{"Name": name}, nil)
		}
		if err != nil {
			return err
		}
		isRegular := header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA
		if filepath.Base(header.Name) == name && isRegular {
			return writeWebPExecutableContext(ctx, tarReader, target)
		}
	}
}

func writeWebPExecutableContext(ctx context.Context, source io.Reader, target string) error {
	file, err := os.CreateTemp(filepath.Dir(target), "cwebp-*")
	if err != nil {
		return err
	}
	path := file.Name()
	defer os.Remove(path)

	written, err := io.Copy(file, io.LimitReader(&webPContextReader{ctx: ctx, reader: source}, maxWebPExecutableSize+1))
	if err != nil {
		_ = file.Close()
		return err
	}
	if written > maxWebPExecutableSize {
		_ = file.Close()
		return i18n.NewError("error.cover.webp_executable_too_large", map[string]any{"Limit": maxWebPExecutableSize}, nil)
	}
	if err := file.Chmod(0o755); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Rename(path, target)
}

type webPContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *webPContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
