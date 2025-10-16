package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/siliconflow/bizyair-cli/lib"
)

// 空值显示占位符
func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// 校验名称：字母/数字/下划线/短横线（使用 lib 层的实现）
func validateName(s string) error {
	return lib.ValidateModelName(s)
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

// IsHTTPURL 使用 lib 层的实现
func IsHTTPURL(s string) bool {
	return lib.IsHTTPURL(s)
}

// isSupportedCoverFormat 使用 lib 层的实现
func isSupportedCoverFormat(path string) bool {
	return lib.IsSupportedCoverFormat(path)
}

// getSupportedCoverFormats 使用 lib 层的实现
func getSupportedCoverFormats() string {
	return lib.GetSupportedCoverFormats()
}

// validateCoverFile 使用 lib 层的实现
func validateCoverFile(path string) error {
	return lib.ValidateCoverFile(path)
}

// validateIntroFile 验证介绍文件格式
func validateIntroFile(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".txt" && ext != ".md" {
		return fmt.Errorf("不支持的文件格式，仅支持 .txt 和 .md 文件")
	}
	return validatePath(path)
}

// readIntroFile 读取介绍文件内容并截断到5000字
func readIntroFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	text := strings.TrimSpace(string(content))
	runes := []rune(text)
	if len(runes) > 5000 {
		runes = runes[:5000]
	}

	return string(runes), nil
}
