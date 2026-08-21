package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/meta"
)

func EnsureAbsPath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, p)
}

// ValidateModelName 校验模型名称格式
func ValidateModelName(name string) error {
	if name == "" {
		return i18n.NewError("validation.model_name_required", nil, nil)
	}
	return nil
}

// ValidateModelType 校验模型类型
func ValidateModelType(modelType string) error {
	if modelType == "" {
		return i18n.NewError("validation.model_type_required", nil, nil)
	}
	mt := meta.UploadFileType(modelType)
	if !lo.Contains(meta.ModelTypes, mt) {
		return i18n.NewError("validation.model_type_unsupported", map[string]any{"Type": modelType, "Supported": meta.ModelTypesStr}, nil)
	}
	return nil
}

// ValidatePath 校验文件路径是否存在
func ValidatePath(path string) error {
	if path == "" {
		return i18n.NewError("validation.path_required", nil, nil)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return i18n.NewError("validation.path_not_found", map[string]any{"Path": path}, err)
	}
	return nil
}

// ValidateBaseModel 校验基础模型
// allowedModels: 从API获取的允许的基础模型列表，必须提供
func ValidateBaseModel(baseModel string, allowedModels []string) error {
	if baseModel == "" {
		return i18n.NewError("validation.base_model_required", nil, nil)
	}

	// 如果没有提供允许的模型列表，返回错误
	if len(allowedModels) == 0 {
		return i18n.NewError("validation.base_model_list_missing", nil, nil)
	}

	// 使用提供的列表验证
	if !lo.Contains(allowedModels, baseModel) {
		return i18n.NewError("validation.base_model_unsupported", map[string]any{
			"Model": baseModel, "Supported": strings.Join(allowedModels, ", "),
		}, nil)
	}

	return nil
}

// ValidateCoverFile 校验封面文件格式和大小
func ValidateCoverFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return i18n.NewError("validation.cover_not_found", map[string]any{"Path": path}, err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	// 检查格式
	supportedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp4", ".webm", ".mov"}
	if !lo.Contains(supportedExts, ext) {
		return i18n.NewError("validation.cover_format_unsupported", map[string]any{"Format": ext, "Supported": strings.Join(supportedExts, ", ")}, nil)
	}

	// 视频文件大小限制 100MB
	if ext == ".mp4" || ext == ".webm" || ext == ".mov" {
		maxSize := int64(100 * 1024 * 1024)
		if info.Size() > maxSize {
			return i18n.NewError("validation.cover_too_large", map[string]any{
				"Size": fmt.Sprintf("%.1f", float64(info.Size())/(1024*1024)), "Limit": 100,
			}, nil)
		}
	}

	return nil
}

// IsSupportedCoverFormat 检查 URL 是否为支持的封面格式
func IsSupportedCoverFormat(url string) bool {
	// 移除查询参数
	if idx := strings.Index(url, "?"); idx >= 0 {
		url = url[:idx]
	}

	ext := strings.ToLower(filepath.Ext(url))
	supportedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp4", ".webm", ".mov"}
	return lo.Contains(supportedExts, ext)
}

// GetSupportedCoverFormats 获取支持的封面格式列表
func GetSupportedCoverFormats() string {
	return ".jpg, .jpeg, .png, .gif, .webp, .mp4, .webm, .mov"
}

// ValidateIntroFile 验证介绍文件格式
func ValidateIntroFile(path string) error {
	if path == "" {
		return i18n.NewError("validation.intro_path_required", nil, nil)
	}

	info, err := os.Stat(path)
	if err != nil {
		return i18n.NewError("validation.file_not_found", map[string]any{"Path": path}, err)
	}

	if info.IsDir() {
		return i18n.NewError("validation.file_is_directory", map[string]any{"Path": path}, nil)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".txt" && ext != ".md" {
		return i18n.NewError("validation.intro_format_unsupported", nil, nil)
	}

	return nil
}

// ReadIntroFile 读取介绍文件内容并截断到5000字
func ReadIntroFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", i18n.NewError("validation.file_read_failed", map[string]any{"Path": path}, err)
	}

	text := strings.TrimSpace(string(content))
	runes := []rune(text)
	if len(runes) > 5000 {
		runes = runes[:5000]
	}

	return string(runes), nil
}
