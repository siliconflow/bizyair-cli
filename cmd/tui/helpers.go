package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// 空值显示占位符
func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// 校验名称：字母/数字/下划线/短横线
func validateName(s string) error {
	if s == "" {
		return fmt.Errorf("name 不能为空")
	}
	re := regexp.MustCompile(`^[\w-]+$`)
	if !re.MatchString(s) {
		return fmt.Errorf("name 仅支持字母/数字/下划线/短横线")
	}
	return nil
}

// 校验路径：必须存在且为文件
func validatePath(p string) error {
	st, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("路径不存在：%s", p)
	}
	if st.IsDir() {
		return fmt.Errorf("当前仅支持文件上传，不支持目录：%s", p)
	}
	return nil
}

// 绝对路径
func absPath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, p)
}

// 复制 lib.IsHTTPURL 以在 tui 包内使用（避免循环依赖）
func IsHTTPURL(s string) bool {
	ls := strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(ls, "http://") || strings.HasPrefix(ls, "https://")
}

// 检查是否为视频格式
func isVideoFormat(path string) bool {
	lp := strings.ToLower(path)
	videoFormats := []string{".mp4", ".mov", ".avi", ".webm", ".mkv", ".flv"}
	for _, ext := range videoFormats {
		if strings.HasSuffix(lp, ext) {
			return true
		}
	}
	return false
}

// 检查是否为支持的封面文件格式（图片或视频）
func isSupportedCoverFormat(path string) bool {
	lp := strings.ToLower(path)
	// 图片格式
	imageFormats := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"}
	// 视频格式
	videoFormats := []string{".mp4", ".mov", ".avi", ".webm", ".mkv", ".flv"}
	
	for _, ext := range imageFormats {
		if strings.HasSuffix(lp, ext) {
			return true
		}
	}
	for _, ext := range videoFormats {
		if strings.HasSuffix(lp, ext) {
			return true
		}
	}
	return false
}

// 获取支持的封面格式列表（用于提示信息）
func getSupportedCoverFormats() string {
	return "图片: jpg, jpeg, png, gif, bmp, webp; 视频: mp4, mov, avi, webm, mkv, flv"
}

// 校验封面文件（检查格式和大小）
func validateCoverFile(path string) error {
	if !isSupportedCoverFormat(path) {
		return fmt.Errorf("不支持的封面文件类型: %s\n支持的格式: %s", path, getSupportedCoverFormats())
	}
	
	// 如果是视频格式，检查文件大小限制（100MB）
	if isVideoFormat(path) {
		st, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("无法读取文件信息: %v", err)
		}
		const maxVideoSize = 100 * 1024 * 1024 // 100MB
		if st.Size() > maxVideoSize {
			return fmt.Errorf("视频文件大小超过限制: %.2f MB (最大 100 MB)", float64(st.Size())/(1024*1024))
		}
	}
	
	return nil
}
