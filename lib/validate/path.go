package validate

import (
	"fmt"
	"os"
	"path/filepath"
)

// ValidateFilePath ensures the path exists and is a file.
func ValidateFilePath(p string) error {
	st, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("路径不存在：%s", p)
	}
	if st.IsDir() {
		return fmt.Errorf("当前仅支持文件上传，不支持目录：%s", p)
	}
	return nil
}

// EnsureAbsPath returns an absolute path for p relative to current working directory.
func EnsureAbsPath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, p)
}
