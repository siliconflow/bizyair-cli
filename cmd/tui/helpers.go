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

// validateIntroFile 验证介绍文件格式（复用 lib 层）
func validateIntroFile(path string) error {
	return lib.ValidateIntroFile(path)
}

// readIntroFile 读取介绍文件内容并截断到5000字（复用 lib 层）
func readIntroFile(path string) (string, error) {
	return lib.ReadIntroFile(path)
}

// truncateToLines 将文本截断为指定行数的预览
// maxLines: 最大行数，默认2行
// maxLineLength: 单行最大字符数，默认80
func truncateToLines(text string, maxLines int) string {
	if text == "" {
		return ""
	}

	const maxLineLength = 80

	// 按换行符分割文本
	lines := strings.Split(text, "\n")

	// 如果文本行数超过限制
	if len(lines) > maxLines {
		// 取前 maxLines 行
		truncated := strings.Join(lines[:maxLines], "\n")
		return truncated + "..."
	}

	// 如果行数不超过，但需要检查单行是否过长
	var result strings.Builder
	needsTruncation := false

	for i, line := range lines {
		if i >= maxLines {
			needsTruncation = true
			break
		}

		lineRunes := []rune(strings.TrimSpace(line))
		if len(lineRunes) > maxLineLength {
			// 单行过长，截断
			result.WriteString(string(lineRunes[:maxLineLength]))
			needsTruncation = true
			break
		}

		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(string(lineRunes))
	}

	if needsTruncation {
		return result.String() + "..."
	}

	return result.String()
}

// getContextualHint 根据当前状态返回上下文相关的操作提示
func (m *mainModel) getContextualHint() string {
	// 退出确认优先级最高
	if m.confirmingExit {
		return "确认：Enter\n取消：Esc"
	}

	// 正在运行的上传任务
	if m.running && m.currentAction == actionUpload && m.step == mainStepAction {
		if m.canceling {
			return "正在取消..."
		}
		return "取消：Ctrl+C"
	}

	// 其他运行状态
	if m.running {
		return "请稍候..."
	}

	// 根据主步骤返回提示
	switch m.step {
	case mainStepLogin:
		return "确认：Enter\n返回：Esc"
	case mainStepMenu:
		return "↑/k 上 • ↓/j 下\n/ 筛选 • Enter 选择"
	case mainStepAction:
		// 根据上传步骤细分
		switch m.upStep {
		case stepType:
			return "↑/k 上 • ↓/j 下\nEnter 选择 • Esc 返回"
		case stepName:
			return "输入模型名称\nEnter 确认 • Esc 返回"
		case stepVersion:
			return "输入版本名称\nEnter 确认 • Esc 返回"
		case stepBase:
			return "↑/k 上 • ↓/j 下\nEnter 选择 • Esc 返回"
		case stepCoverMethod:
			return "↑/k 上 • ↓/j 下\nEnter 选择 • Esc 返回"
		case stepCover:
			if m.act.coverUploadMethod == "url" {
				return "输入封面 URL\nEnter 确认 • Esc 返回"
			}
			// 本地文件上传模式 - 根据焦点显示不同提示
			if m.coverPathInputFocused {
				return "当前焦点：路径输入框\n输入文件路径，Enter 确认\nTab 切换至文件选择器 • Esc 返回"
			}
			return "当前焦点：文件选择器\n← → 进入/退出文件夹 • ↑ ↓ 选择\nEnter 确认 • Tab 切换至路径输入框 • Esc 返回"
		case stepIntroMethod:
			return "↑/k 上 • ↓/j 下\nEnter 选择 • Esc 返回"
		case stepIntro:
			if m.act.introInputMethod == "file" {
				// 文件导入模式 - 根据焦点显示不同提示
				if m.act.introPathInputFocused {
					return "当前焦点：路径输入框\n输入 .txt 或 .md 文件路径\nEnter 确认 • Tab 切换至文件选择器 • Esc 返回"
				}
				return "当前焦点：文件选择器\n← → 进入/退出文件夹 • ↑ ↓ 选择\nEnter 确认 • Tab 切换至路径输入框 • Esc 返回"
			}
			return "支持 Markdown 格式\nCtrl+S 提交 • Esc 返回"
		case stepPath:
			// 文件路径选择 - 根据焦点显示不同提示
			if m.act.pathInputFocused {
				return "当前焦点：路径输入框\n输入文件路径，Enter 确认\nTab 切换至文件选择器 • Esc 返回"
			}
			return "当前焦点：文件选择器\n← → 进入/退出文件夹 • ↑ ↓ 选择\nEnter 确认 • Tab 切换至路径输入框 • Esc 返回"
		case stepPublic:
			return "↑/k 上 • ↓/j 下\nEnter 选择 • Esc 返回"
		case stepAskMore:
			return "↑/k 上 • ↓/j 下\nEnter 选择 • Esc 返回"
		case stepConfirm:
			return "Enter 开始上传\nEsc 返回"
		}
	case mainStepOutput:
		return "Enter 返回菜单"
	}

	return ""
}
