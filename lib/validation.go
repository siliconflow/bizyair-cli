package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/samber/lo"
	"github.com/siliconflow/bizyair-cli/meta"
)

// ValidateModelName 校验模型名称格式
func ValidateModelName(name string) error {
	if name == "" {
		return fmt.Errorf("模型名称不能为空")
	}
	re := regexp.MustCompile(`^[\w-]+$`)
	if !re.MatchString(name) {
		return fmt.Errorf("模型名称只能包含字母、数字、下划线和短横线")
	}
	return nil
}

// ValidateModelType 校验模型类型
func ValidateModelType(modelType string) error {
	if modelType == "" {
		return fmt.Errorf("模型类型不能为空")
	}
	mt := meta.UploadFileType(modelType)
	if !lo.Contains[meta.UploadFileType](meta.ModelTypes, mt) {
		return fmt.Errorf("不支持的模型类型 [%s]，仅支持 %s", modelType, meta.ModelTypesStr)
	}
	return nil
}

// ValidatePath 校验文件路径是否存在
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("文件路径不能为空")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("路径不存在: %s", path)
	}
	return nil
}

// ValidateBaseModel 校验基础模型
func ValidateBaseModel(baseModel string) error {
	if baseModel == "" {
		return fmt.Errorf("基础模型不能为空")
	}
	valid, exists := meta.SupportedBaseModels[baseModel]
	if !exists {
		return fmt.Errorf("不支持的基础模型: %s", baseModel)
	}
	if !valid {
		return fmt.Errorf("基础模型无效: %s", baseModel)
	}
	return nil
}

// ValidateCoverFile 校验封面文件格式和大小
func ValidateCoverFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("封面文件不存在: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	// 检查格式
	supportedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp4", ".webm", ".mov"}
	if !lo.Contains(supportedExts, ext) {
		return fmt.Errorf("不支持的封面格式: %s（支持: %s）", ext, strings.Join(supportedExts, ", "))
	}

	// 视频文件大小限制 100MB
	if ext == ".mp4" || ext == ".webm" || ext == ".mov" {
		maxSize := int64(100 * 1024 * 1024)
		if info.Size() > maxSize {
			return fmt.Errorf("视频封面大小超过限制（%.1f MB > 100 MB）",
				float64(info.Size())/(1024*1024))
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
		return fmt.Errorf("intro 文件路径不能为空")
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("文件不存在: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("路径是目录，需要文件: %s", path)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".txt" && ext != ".md" {
		return fmt.Errorf("不支持的文件格式，仅支持 .txt 和 .md 文件")
	}

	return nil
}

// ReadIntroFile 读取介绍文件内容并截断到5000字
func ReadIntroFile(path string) (string, error) {
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
