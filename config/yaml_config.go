package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/siliconflow/bizyair-cli/lib"
	"gopkg.in/yaml.v3"
)

// YamlConfig YAML 配置文件的根结构
type YamlConfig struct {
	Models []YamlModel `yaml:"models"`
}

// YamlModel 单个模型的配置
type YamlModel struct {
	Name     string        `yaml:"name"`
	Type     string        `yaml:"type"`
	Versions []YamlVersion `yaml:"versions"`
}

// YamlVersion 单个版本的配置
type YamlVersion struct {
	Name      string `yaml:"name"`       // 可选，默认递增
	BaseModel string `yaml:"base_model"` // 基础模型
	ModelPath string `yaml:"model_path"` // 模型文件路径（必填）
	CoverPath string `yaml:"cover_path"` // 本地封面文件路径（与 CoverUrl 二选一）
	CoverUrl  string `yaml:"cover_url"`  // 封面网络 URL（与 CoverPath 二选一）
	Intro     string `yaml:"intro"`      // 直接文本介绍（与 IntroPath 二选一）
	IntroPath string `yaml:"intro_path"` // 介绍文件路径（与 Intro 二选一）
	Public    *bool  `yaml:"public"`     // 是否公开，指针类型以区分未设置和 false
}

// LoadYamlConfig 从文件加载并解析 YAML 配置
func LoadYamlConfig(filepath string) (*YamlConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("读取 YAML 文件失败: %w", err)
	}

	var config YamlConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析 YAML 文件失败: %w", err)
	}

	return &config, nil
}

// ValidateYamlConfig 验证 YAML 配置的合法性
func ValidateYamlConfig(config *YamlConfig) error {
	if len(config.Models) == 0 {
		return fmt.Errorf("配置文件中至少需要一个模型")
	}

	for i, model := range config.Models {
		// 验证模型名称
		if err := lib.ValidateModelName(model.Name); err != nil {
			return fmt.Errorf("模型 %d (%s): 名称无效: %w", i+1, model.Name, err)
		}

		// 验证模型类型
		if err := lib.ValidateModelType(model.Type); err != nil {
			return fmt.Errorf("模型 %d (%s): 类型无效: %w", i+1, model.Name, err)
		}

		// 验证至少有一个版本
		if len(model.Versions) == 0 {
			return fmt.Errorf("模型 %d (%s): 至少需要一个版本", i+1, model.Name)
		}

		// 验证每个版本
		for j, version := range model.Versions {
			if err := validateYamlVersion(version, model.Name, j+1); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateYamlVersion 验证单个版本的配置
func validateYamlVersion(version YamlVersion, modelName string, versionIndex int) error {
	prefix := fmt.Sprintf("模型 %s, 版本 %d", modelName, versionIndex)

	// 验证 model_path 必填且文件存在
	if version.ModelPath == "" {
		return fmt.Errorf("%s: model_path 不能为空", prefix)
	}
	if err := lib.ValidatePath(version.ModelPath); err != nil {
		return fmt.Errorf("%s: model_path 无效: %w", prefix, err)
	}

	// 验证 cover_path 和 cover_url 二选一（必须有且只有一个）
	hasCoverPath := version.CoverPath != ""
	hasCoverUrl := version.CoverUrl != ""
	if !hasCoverPath && !hasCoverUrl {
		return fmt.Errorf("%s: 必须指定 cover_path 或 cover_url 其中之一", prefix)
	}
	if hasCoverPath && hasCoverUrl {
		return fmt.Errorf("%s: cover_path 和 cover_url 不能同时指定", prefix)
	}

	// 如果是 cover_path，验证文件存在
	if hasCoverPath {
		if err := lib.ValidatePath(version.CoverPath); err != nil {
			return fmt.Errorf("%s: cover_path 无效: %w", prefix, err)
		}
		// 验证封面文件格式
		if err := lib.ValidateCoverFile(version.CoverPath); err != nil {
			return fmt.Errorf("%s: cover_path 格式无效: %w", prefix, err)
		}
	}

	// 验证 intro 和 intro_path 不能同时指定
	hasIntro := version.Intro != ""
	hasIntroPath := version.IntroPath != ""
	if hasIntro && hasIntroPath {
		return fmt.Errorf("%s: intro 和 intro_path 不能同时指定", prefix)
	}

	// 如果是 intro_path，验证文件存在
	if hasIntroPath {
		if err := lib.ValidateIntroFile(version.IntroPath); err != nil {
			return fmt.Errorf("%s: intro_path 无效: %w", prefix, err)
		}
	}

	// 验证 base_model（如果指定）
	if version.BaseModel != "" {
		if err := lib.ValidateBaseModel(version.BaseModel); err != nil {
			return fmt.Errorf("%s: base_model 无效: %w", prefix, err)
		}
	}

	return nil
}

// AutoIncrementVersionNames 自动为未指定 name 的版本生成递增的版本号
// 逻辑：检查已指定的版本号中的最大值，从该值开始递增
func AutoIncrementVersionNames(versions []YamlVersion) []YamlVersion {
	maxVersionNum := 0
	result := make([]YamlVersion, len(versions))

	// 第一遍：复制数据并找到最大版本号
	for i, v := range versions {
		result[i] = v
		if v.Name != "" {
			// 提取版本号，如 "v2.0" -> 2
			if num := extractVersionNumber(v.Name); num > maxVersionNum {
				maxVersionNum = num
			}
		}
	}

	// 第二遍：为空的 name 分配递增版本号
	currentNum := maxVersionNum
	for i := range result {
		if result[i].Name == "" {
			currentNum++
			result[i].Name = fmt.Sprintf("v%d.0", currentNum)
		}
	}

	return result
}

// extractVersionNumber 从版本号字符串中提取主版本号数字
// 例如: "v1.0" -> 1, "v2.5" -> 2, "v10.0" -> 10
// 如果无法提取，返回 0
func extractVersionNumber(version string) int {
	// 匹配 vX.Y 格式的版本号
	re := regexp.MustCompile(`^v?(\d+)\..*`)
	matches := re.FindStringSubmatch(version)
	if len(matches) >= 2 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			return num
		}
	}

	// 匹配 vX 格式的版本号
	re = regexp.MustCompile(`^v?(\d+)$`)
	matches = re.FindStringSubmatch(version)
	if len(matches) >= 2 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			return num
		}
	}

	return 0
}

// GetCoverInput 获取统一的封面输入（cover_path 或 cover_url）
func (v *YamlVersion) GetCoverInput() string {
	if v.CoverPath != "" {
		return v.CoverPath
	}
	return v.CoverUrl
}

// GetIntroduction 获取版本介绍（从 intro 或 intro_path）
func (v *YamlVersion) GetIntroduction() (string, error) {
	if v.IntroPath != "" {
		// 从文件读取
		content, err := lib.ReadIntroFile(v.IntroPath)
		if err != nil {
			return "", fmt.Errorf("读取 intro 文件失败: %w", err)
		}
		return content, nil
	}
	return v.Intro, nil
}

// GetPublic 获取 public 设置（默认 false）
func (v *YamlVersion) GetPublic() bool {
	if v.Public != nil {
		return *v.Public
	}
	return false
}

// NormalizeModelPaths 将模型配置中的相对路径转换为基于 YAML 文件所在目录的绝对路径
func NormalizeModelPaths(config *YamlConfig, yamlDir string) error {
	for i := range config.Models {
		for j := range config.Models[i].Versions {
			ver := &config.Models[i].Versions[j]

			// 规范化 model_path
			if ver.ModelPath != "" && !strings.HasPrefix(ver.ModelPath, "/") && !strings.HasPrefix(ver.ModelPath, "http://") && !strings.HasPrefix(ver.ModelPath, "https://") {
				ver.ModelPath = normalizeRelativePath(yamlDir, ver.ModelPath)
			}

			// 规范化 cover_path（如果是本地路径）
			if ver.CoverPath != "" && !strings.HasPrefix(ver.CoverPath, "/") && !strings.HasPrefix(ver.CoverPath, "http://") && !strings.HasPrefix(ver.CoverPath, "https://") {
				ver.CoverPath = normalizeRelativePath(yamlDir, ver.CoverPath)
			}

			// 规范化 intro_path
			if ver.IntroPath != "" && !strings.HasPrefix(ver.IntroPath, "/") {
				ver.IntroPath = normalizeRelativePath(yamlDir, ver.IntroPath)
			}
		}
	}
	return nil
}

// normalizeRelativePath 将相对路径转换为基于指定目录的绝对路径
func normalizeRelativePath(baseDir, relativePath string) string {
	if baseDir == "" {
		return relativePath
	}
	return strings.TrimSpace(baseDir + "/" + relativePath)
}

