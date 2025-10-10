package validate

import (
	"fmt"
	"regexp"

	"github.com/siliconflow/bizyair-cli/meta"
)

// ValidateUploadType ensures the type is one of supported meta.ModelTypes.
func ValidateUploadType(typ string) error {
	if typ == "" {
		return fmt.Errorf("type 不能为空")
	}
	// meta.ModelTypes 是 []UploadFileType，逐一比较字符串
	ok := false
	for _, t := range meta.ModelTypes {
		if string(t) == typ {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("不支持的类型 [%s]，仅支持: %s", typ, meta.ModelTypesStr)
	}
	return nil
}

// ValidateModelName checks name rules: letters/digits/_/-
func ValidateModelName(name string) error {
	if name == "" {
		return fmt.Errorf("name 不能为空")
	}
	re := regexp.MustCompile(`^[\w-]+$`)
	if !re.MatchString(name) {
		return fmt.Errorf("name 仅支持字母/数字/下划线/短横线")
	}
	return nil
}
