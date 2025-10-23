package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

// findPathCompletions 查找路径补全匹配项
// 参数：
//   - inputPath: 用户输入的路径
//   - currentDir: filepicker 当前目录
//   - allowedTypes: 允许的文件扩展名列表
//   - dirAllowed: 是否允许选择目录
//   - fileAllowed: 是否允许选择文件
//
// 返回：
//   - matches: 所有匹配的文件/目录名（完整路径）
//   - dir: 解析出的目录路径
//   - prefix: 需要匹配的文件名前缀
func findPathCompletions(inputPath, currentDir string, allowedTypes []string, dirAllowed, fileAllowed bool) (matches []string, dir string, prefix string) {
	if inputPath == "" {
		return nil, currentDir, ""
	}

	// 处理路径
	var targetDir, filePrefix string

	// 判断输入是否以路径分隔符结尾（表示用户已经进入某个目录）
	if strings.HasSuffix(inputPath, string(filepath.Separator)) {
		// 输入以 / 结尾，表示已经是一个目录
		targetDir = inputPath
		filePrefix = ""
	} else {
		// 分离目录和文件名部分
		targetDir, filePrefix = filepath.Split(inputPath)
		if targetDir == "" {
			targetDir = currentDir
		}
	}

	// 如果目录不存在，尝试相对于当前目录解析
	if !filepath.IsAbs(targetDir) && targetDir != "" {
		targetDir = filepath.Join(currentDir, targetDir)
	} else if targetDir == "" {
		targetDir = currentDir
	}

	// 清理路径
	targetDir = filepath.Clean(targetDir)

	// 检查目录是否存在
	info, err := os.Stat(targetDir)
	if err != nil || !info.IsDir() {
		return nil, currentDir, ""
	}

	// 读取目录内容
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, targetDir, filePrefix
	}

	// 过滤和匹配
	lowerPrefix := strings.ToLower(filePrefix)
	for _, entry := range entries {
		name := entry.Name()

		// 跳过隐藏文件（以 . 开头）
		if strings.HasPrefix(name, ".") {
			continue
		}

		// 前缀匹配（大小写不敏感）
		if !strings.HasPrefix(strings.ToLower(name), lowerPrefix) {
			continue
		}

		fullPath := filepath.Join(targetDir, name)

		// 如果是目录，始终添加（用于导航）
		if entry.IsDir() {
			if dirAllowed || fileAllowed {
				matches = append(matches, fullPath)
			}
			continue
		}

		// 如果是文件，检查是否允许文件选择
		if !fileAllowed {
			continue
		}

		// 检查文件扩展名
		if len(allowedTypes) > 0 {
			matched := false
			for _, ext := range allowedTypes {
				if strings.HasSuffix(strings.ToLower(name), strings.ToLower(ext)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		matches = append(matches, fullPath)
	}

	return matches, targetDir, filePrefix
}

// getCommonPrefix 计算多个字符串的公共前缀
func getCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			if len(prefix) == 0 {
				return ""
			}
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

// buildCompletionSuggestion 构建补全建议
// 如果只有一个匹配项，返回完整路径
// 如果多个匹配项，返回到公共前缀的路径
func buildCompletionSuggestion(inputPath string, matches []string) string {
	if len(matches) == 0 {
		return ""
	}

	if len(matches) == 1 {
		// 单一匹配，返回完整路径
		match := matches[0]
		// 如果是目录，添加路径分隔符
		if info, err := os.Stat(match); err == nil && info.IsDir() {
			return ensureTrailingSep(match)
		}
		return match
	}

	// 多个匹配，计算公共前缀
	commonPrefix := getCommonPrefix(matches)
	return commonPrefix
}

// getContextualHint 根据当前状态返回上下文相关的操作提示
func (m *mainModel) getContextualHint() string {
	// 退出确认优先级最高
	if m.confirmingExit {
		return "<Enter> 确认\n<Esc> 取消"
	}

	// 正在运行的上传任务
	if m.running && m.currentAction == actionUpload && m.step == mainStepAction {
		if m.canceling {
			return "正在取消..."
		}
		return "<Ctrl+C> 取消上传"
	}

	// 其他运行状态
	if m.running {
		return "请稍候..."
	}

	// 根据主步骤返回提示
	switch m.step {
	case mainStepLogin:
		return "<Enter> 确认\n<Esc> 返回"
	case mainStepMenu:
		return "<↑/k> 上移\n<↓/j> 下移\n</> 筛选\n<Enter> 选择"
	case mainStepAction:
		// 根据上传步骤细分
		switch m.upStep {
		case stepType:
			return "<↑/k> 上移\n<↓/j> 下移\n<Enter> 选择\n<Esc> 返回"
		case stepName:
			return "<Enter> 确认\n<Esc> 返回"
		case stepVersion:
			return "<Enter> 确认\n<Esc> 返回"
		case stepBase:
			return "<↑/k> 上移\n<↓/j> 下移\n<Enter> 选择\n<Esc> 返回"
		case stepCoverMethod:
			return "<↑/k> 上移\n<↓/j> 下移\n<Enter> 选择\n<Esc> 返回"
		case stepCover:
			if m.act.coverUploadMethod == "url" {
				return "<Enter> 确认\n<Esc> 返回"
			}
			// 本地文件上传模式 - 根据焦点显示不同提示
			if m.coverPathInputFocused {
				return "<Tab> 补全\n<Enter> 确认\n<Ctrl+P> 切换至文件选择器\n<Esc> 返回"
			}
			return "<←/→> 导航\n<↑/↓> 选择\n<Enter> 确认\n<Ctrl+P> 切换至路径输入框\n<Esc> 返回"
		case stepIntroMethod:
			return "<↑/k> 上移\n<↓/j> 下移\n<Enter> 选择\n<Esc> 返回"
		case stepIntro:
			if m.act.introInputMethod == "file" {
				// 文件导入模式 - 根据焦点显示不同提示
				if m.act.introPathInputFocused {
					return "<Tab> 补全\n<Enter> 确认\n<Ctrl+P> 切换至文件选择器\n<Esc> 返回"
				}
				return "<←/→> 导航\n<↑/↓> 选择\n<Enter> 确认\n<Ctrl+P> 切换至路径输入框\n<Esc> 返回"
			}
			return "<Ctrl+S> 提交\n<Esc> 返回"
		case stepPath:
			// 文件路径选择 - 根据焦点显示不同提示
			if m.act.pathInputFocused {
				return "<Tab> 补全\n<Enter> 确认\n<Ctrl+P> 切换至文件选择器\n<Esc> 返回"
			}
			return "<←/→> 导航\n<↑/↓> 选择\n<Enter> 确认\n<Ctrl+P> 切换至路径输入框\n<Esc> 返回"
		case stepPublic:
			return "<↑/k> 上移\n<↓/j> 下移\n<Enter> 选择\n<Esc> 返回"
		case stepAskMore:
			return "<↑/k> 上移\n<↓/j> 下移\n<Enter> 选择\n<Esc> 返回"
		case stepConfirm:
			return "<Enter> 开始上传\n<Esc> 返回"
		}
	case mainStepOutput:
		return "<Enter> 返回菜单"
	}

	return ""
}

// renderStyledHint 渲染带样式的提示文本（K9s风格）
// 将 <key> 部分用特殊样式渲染
// 纵向一列最多4行，超过4行则分成多列横向排列
func (m *mainModel) renderStyledHint(hint string) string {
	const maxRowsPerColumn = 4

	// 定义按键样式（黄色/金色 + 粗体）
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FBBF24")).
		Bold(true)

	// 定义普通文本样式（浅灰色）
	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB"))

	// 渲染单行文本（处理 <key> 标记）
	renderLine := func(line string) string {
		var result strings.Builder
		i := 0
		for i < len(line) {
			if line[i] == '<' {
				end := strings.Index(line[i:], ">")
				if end != -1 {
					key := line[i+1 : i+end]
					result.WriteString(keyStyle.Render("<" + key + ">"))
					i += end + 1
					continue
				}
			}
			start := i
			for i < len(line) && line[i] != '<' {
				i++
			}
			if i > start {
				result.WriteString(textStyle.Render(line[start:i]))
			}
		}
		return result.String()
	}

	// 分割成行
	lines := strings.Split(hint, "\n")

	// 如果行数 <= 4，直接纵向排列
	if len(lines) <= maxRowsPerColumn {
		var result strings.Builder
		for i, line := range lines {
			if i > 0 {
				result.WriteString("\n")
			}
			result.WriteString(renderLine(line))
		}
		return result.String()
	}

	// 行数 > 4，需要分成多列
	// 计算需要多少列
	numColumns := (len(lines) + maxRowsPerColumn - 1) / maxRowsPerColumn

	// 创建列
	columns := make([][]string, numColumns)
	for i, line := range lines {
		colIdx := i / maxRowsPerColumn
		columns[colIdx] = append(columns[colIdx], renderLine(line))
	}

	// 找出最长的行宽度（用于对齐）
	maxWidth := 0
	for _, col := range columns {
		for _, line := range col {
			// 计算不含ANSI转义序列的实际宽度
			width := lipgloss.Width(line)
			if width > maxWidth {
				maxWidth = width
			}
		}
	}

	// 为每列添加填充，使其对齐
	for i := range columns {
		for j := range columns[i] {
			currentWidth := lipgloss.Width(columns[i][j])
			if currentWidth < maxWidth {
				columns[i][j] = columns[i][j] + strings.Repeat(" ", maxWidth-currentWidth)
			}
		}
	}

	// 将列合并成行
	var rows []string
	for rowIdx := 0; rowIdx < maxRowsPerColumn; rowIdx++ {
		var rowParts []string
		for colIdx := 0; colIdx < numColumns; colIdx++ {
			if rowIdx < len(columns[colIdx]) {
				rowParts = append(rowParts, columns[colIdx][rowIdx])
			}
		}
		if len(rowParts) > 0 {
			rows = append(rows, strings.Join(rowParts, "  "))
		}
	}

	return strings.Join(rows, "\n")
}
