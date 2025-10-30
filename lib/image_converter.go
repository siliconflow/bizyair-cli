package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nickalie/go-webpbin"
)

// getWebPVendorPath 获取WebP工具的存储路径（相对于可执行文件）
func getWebPVendorPath() string {
	// 获取可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		// 如果获取失败，使用当前工作目录
		return ".bin/webp"
	}

	// 返回可执行文件同目录下的 .bin/webp
	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, ".bin", "webp")
}

// ConvertImageToWebP 将图片转换为WebP格式
// 如果不需要转换（已经是webp或视频文件），返回原路径
// 返回值：转换后的文件路径、清理函数、错误
func ConvertImageToWebP(sourcePath string) (string, func(), error) {
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
		return "", nil, fmt.Errorf("无法创建临时文件: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close() // CWebP需要文件路径，先关闭文件

	// 定义清理函数
	cleanup := func() {
		os.Remove(tmpPath)
	}

	// 设置WebP工具路径（相对于可执行文件）
	vendorPath := getWebPVendorPath()

	// 临时抑制stdout和stderr输出，避免下载日志干扰TUI界面
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)
	if err == nil {
		os.Stdout = devNull
		os.Stderr = devNull
		defer func() {
			os.Stdout = oldStdout
			os.Stderr = oldStderr
			devNull.Close()
		}()
	}

	// 使用CWebP转换图片，设置质量为75
	// 通过SetVendorPath选项，让工具下载/读取到可执行文件同目录
	err = webpbin.NewCWebP(webpbin.SetVendorPath(vendorPath)).
		Quality(75).
		InputFile(sourcePath).
		OutputFile(tmpPath).
		Run()
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("无法转换为WebP格式: %w", err)
	}

	return tmpPath, cleanup, nil
}
