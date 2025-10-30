package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// 列表项（与 bubbles/list 兼容）
type listItem struct{ title, desc string }

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.desc }
func (i listItem) FilterValue() string { return i.title }

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
