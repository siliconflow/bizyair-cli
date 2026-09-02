package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tablev2 "charm.land/bubbles/v2/table"
	lipglossv2 "charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/cmd/tui/filepicker"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
)

// 空值显示占位符
func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// formatTimestamp 将 RFC3339 时间串格式化为本地可读格式；解析失败时原样返回。
func formatTimestamp(ts string) string {
	if ts == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Format("2006-01-02 15:04:05")
}

// taskElapsedText 返回任务已执行时间文本（与 CLI 轮询样式一致）。
func taskElapsedText(d time.Duration) string {
	s := int(d.Seconds())
	if s < 1 {
		s = 1
	}
	return fmt.Sprintf("(%d s)", s)
}

// 校验名称：字母/数字/下划线/短横线（使用 lib 层的实现）
func validateName(s string) error {
	return lib.ValidateModelName(s)
}

// 校验路径：必须存在且为文件
func validatePath(p string) error {
	st, err := os.Stat(p)
	if err != nil {
		return i18n.NewError("tui.error.path_missing", map[string]any{"Path": p}, err)
	}
	if st.IsDir() {
		return i18n.NewError("tui.error.path_is_directory", map[string]any{"Path": p}, nil)
	}
	return nil
}

func localizeList(m *list.Model) {
	m.FilterInput.Prompt = i18n.T("tui.list.filter_prompt", nil)
	m.SetStatusBarItemName(i18n.T("tui.list.item", nil), i18n.T("tui.list.items", nil))
	m.KeyMap.CursorUp.SetHelp("↑/k", i18n.T("tui.list.up", nil))
	m.KeyMap.CursorDown.SetHelp("↓/j", i18n.T("tui.list.down", nil))
	m.KeyMap.PrevPage.SetHelp("←/h/pgup", i18n.T("tui.list.previous_page", nil))
	m.KeyMap.NextPage.SetHelp("→/l/pgdn", i18n.T("tui.list.next_page", nil))
	m.KeyMap.GoToStart.SetHelp("g/home", i18n.T("tui.list.start", nil))
	m.KeyMap.GoToEnd.SetHelp("G/end", i18n.T("tui.list.end", nil))
	m.KeyMap.Filter.SetHelp("/", i18n.T("tui.list.filter", nil))
	m.KeyMap.ClearFilter.SetHelp("esc", i18n.T("tui.list.clear_filter", nil))
	m.KeyMap.CancelWhileFiltering.SetHelp("esc", i18n.T("tui.list.cancel", nil))
	m.KeyMap.AcceptWhileFiltering.SetHelp("enter", i18n.T("tui.list.apply_filter", nil))
	m.KeyMap.ShowFullHelp.SetHelp("?", i18n.T("tui.list.more_help", nil))
	m.KeyMap.CloseFullHelp.SetHelp("?", i18n.T("tui.list.close_help", nil))
	m.KeyMap.Quit.SetHelp("q", i18n.T("tui.list.quit", nil))
}

func localizeFilePicker(m *filepicker.Model) {
	m.KeyMap.GoToTop.SetHelp("g", i18n.T("tui.filepicker.first", nil))
	m.KeyMap.GoToLast.SetHelp("G", i18n.T("tui.filepicker.last", nil))
	m.KeyMap.Down.SetHelp("j", i18n.T("tui.filepicker.down", nil))
	m.KeyMap.Up.SetHelp("k", i18n.T("tui.filepicker.up", nil))
	m.KeyMap.PageUp.SetHelp("pgup", i18n.T("tui.filepicker.page_up", nil))
	m.KeyMap.PageDown.SetHelp("pgdown", i18n.T("tui.filepicker.page_down", nil))
	m.KeyMap.Back.SetHelp("h", i18n.T("tui.filepicker.back", nil))
	m.KeyMap.Open.SetHelp("l", i18n.T("tui.filepicker.open", nil))
	m.KeyMap.Select.SetHelp("enter", i18n.T("tui.filepicker.select", nil))
}

func uploadTitle(current int, sectionID string) string {
	return i18n.T("tui.upload.step_title", map[string]any{
		"Current": current,
		"Total":   11,
		"Name":    i18n.T(sectionID, nil),
	})
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
func buildCompletionSuggestion(matches []string) string {
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
		return i18n.T("tui.hint.confirm_cancel", nil)
	}

	// 正在运行的上传任务
	if m.running && m.currentAction == actionUpload && m.step == mainStepAction {
		if m.canceling {
			return i18n.T("tui.hint.canceling", nil)
		}
		return i18n.T("tui.hint.cancel_upload", nil)
	}

	// 其他运行状态
	if m.running {
		return i18n.T("tui.hint.wait", nil)
	}

	// 根据主步骤返回提示
	switch m.step {
	case mainStepLogin:
		return i18n.T("tui.hint.confirm_back", nil)
	case mainStepMenu:
		return i18n.T("tui.hint.menu", nil)
	case mainStepAction:
		// 根据上传步骤细分
		switch m.upStep {
		case stepType:
			return i18n.T("tui.hint.select_back", nil)
		case stepName:
			return i18n.T("tui.hint.confirm_back", nil)
		case stepVersion:
			return i18n.T("tui.hint.confirm_back", nil)
		case stepBase:
			return i18n.T("tui.hint.select_back", nil)
		case stepCoverMethod:
			return i18n.T("tui.hint.select_back", nil)
		case stepCover:
			if m.act.coverUploadMethod == "url" {
				return i18n.T("tui.hint.confirm_back", nil)
			}
			// 本地文件上传模式 - 根据焦点显示不同提示
			if m.coverPathInputFocused {
				return i18n.T("tui.hint.path_input", nil)
			}
			return i18n.T("tui.hint.file_picker", nil)
		case stepIntroMethod:
			return i18n.T("tui.hint.select_back", nil)
		case stepIntro:
			if m.act.introInputMethod == "file" {
				// 文件导入模式 - 根据焦点显示不同提示
				if m.act.introPathInputFocused {
					return i18n.T("tui.hint.path_input", nil)
				}
				return i18n.T("tui.hint.file_picker", nil)
			}
			return i18n.T("tui.hint.editor", nil)
		case stepPath:
			// 文件路径选择 - 根据焦点显示不同提示
			if m.act.pathInputFocused {
				return i18n.T("tui.hint.path_input", nil)
			}
			return i18n.T("tui.hint.file_picker", nil)
		case stepPublic:
			return i18n.T("tui.hint.select_back", nil)
		case stepAskMore:
			return i18n.T("tui.hint.select_back", nil)
		case stepConfirm:
			return i18n.T("tui.hint.start_upload", nil)
		}
	case mainStepMyModelsList:
		return i18n.T("tui.my_models.hint_list", nil)
	case mainStepModelDetail:
		return i18n.T("tui.my_models.hint_detail", nil)
	case mainStepModelzoo:
		return i18n.T("tui.modelzoo.hint_list", nil)
	case mainStepModelzooDetail:
		return i18n.T("tui.modelzoo.hint_detail", nil)
	case mainStepAIApp:
		return i18n.T("tui.ai_app.hint_list", nil)
	case mainStepAIAppDetail:
		return i18n.T("tui.ai_app.hint_detail", nil)
	case mainStepAIAppTask:
		switch m.aiAppTask.step {
		case aiAppTaskParamPoll:
			return i18n.T("tui.task_model.param_hint_input", nil)
		case aiAppTaskSubmitted, aiAppTaskPolling:
			return i18n.T("tui.ai_app.hint_task_poll", nil)
		case aiAppTaskResult:
			return i18n.T("tui.ai_app.hint_task_result", nil)
		}
	case mainStepPriceView:
		return i18n.T("tui.hint.return_menu", nil)
	case mainStepTaskModel:
		return i18n.T("tui.task_model.hint", nil)
	case mainStepOutput:
		return i18n.T("tui.hint.return_menu", nil)
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

// keyValueColLayout 计算"标签 │ 值"两列表格的列宽布局。
// 返回值：标签列宽、分隔符宽、值列宽、标签内容宽、值内容宽。
func keyValueColLayout(maxW int, labels []string) (labelColW, sepW, valueColW, labelContentW, valueContentW int) {
	maxLabelW := 0
	for _, l := range labels {
		if w := lipgloss.Width(l); w > maxLabelW {
			maxLabelW = w
		}
	}

	sep := " │ "
	sepW = lipgloss.Width(sep)
	hPad := 2

	availW := maxW
	labelColW = maxLabelW + hPad
	maxLabelColW := availW / 3
	if maxLabelColW < 12 {
		maxLabelColW = 12
	}
	if labelColW > maxLabelColW {
		labelColW = maxLabelColW
	}

	valueColW = availW - labelColW - sepW
	if valueColW < 12 {
		valueColW = 12
	}

	labelContentW = labelColW - hPad
	valueContentW = valueColW - hPad
	return
}

// renderKeyValueTable 渲染"标签 │ 值"两列表格：
// 标签列按内容自适应并限制在 maxW 的三分之一内，值列占剩余宽度，
// 行间以横线分隔，超宽内容由 truncateLine 截断。
func renderKeyValueTable(labels, values []string, maxW int) string {
	if len(labels) == 0 || len(labels) != len(values) {
		return ""
	}

	labelColW, sepW, valueColW, labelContentW, valueContentW := keyValueColLayout(maxW, labels)
	sep := " │ "

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#22D3EE")).
		Bold(true).
		Padding(0, 1).
		Width(labelColW)

	valueStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(valueColW)

	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	lineW := labelColW + sepW + valueColW
	lineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))
	line := lineStyle.Render(strings.Repeat("─", lineW))

	var parts []string
	for i := range labels {
		label := truncateLine(labels[i], labelContentW)
		value := truncateLine(values[i], valueContentW)

		leftCell := labelStyle.Render(label)
		rightCell := valueStyle.Render(value)
		sepCell := sepStyle.Render(sep)

		row := lipgloss.JoinHorizontal(lipgloss.Top, leftCell, sepCell, rightCell)
		parts = append(parts, row)

		if i < len(labels)-1 {
			parts = append(parts, line)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// --- v2 table helpers ---

var tableFrame = lipglossv2.NewStyle().
	BorderStyle(lipglossv2.NormalBorder()).
	BorderForeground(lipglossv2.Color("240"))

// fitColumnWidthsByContent 根据每列理想宽度与可用宽度计算实际列宽：
// 空间充足时均分余量（fill=true），不足时按比例压缩（保底 4 列宽）。
func fitColumnWidthsByContent(ideal []int, maxW int, fill bool) []tablev2.Column {
	if len(ideal) == 0 {
		return nil
	}
	const minW = 4
	sum := 0
	for _, w := range ideal {
		sum += w
	}
	n := len(ideal)
	result := make([]tablev2.Column, n)
	if sum <= maxW {
		if fill {
			leftover := maxW - sum
			base := leftover / n
			rem := leftover % n
			for i, w := range ideal {
				extra := base
				if i < rem {
					extra++
				}
				result[i] = tablev2.Column{Width: w + extra}
			}
			return result
		}
		for i, w := range ideal {
			result[i] = tablev2.Column{Width: w}
		}
		return result
	}
	remaining := maxW - n*minW
	if remaining < 0 {
		remaining = 0
	}
	used := 0
	for i, w := range ideal {
		aw := minW
		if sum > 0 && remaining > 0 {
			extra := remaining * w / sum
			aw += extra
		}
		if aw < minW {
			aw = minW
		}
		result[i] = tablev2.Column{Width: aw}
		used += aw
	}
	overflow := used - maxW
	for i := len(result) - 1; i >= 0 && overflow > 0; i-- {
		if result[i].Width > minW {
			cut := result[i].Width - minW
			if cut > overflow {
				cut = overflow
			}
			result[i].Width -= cut
			overflow -= cut
		}
	}
	return result
}

// applyTableStyles 设置 v2 table 的样式：灰色 Header 边框（240）、
// 深蓝选中底（57）、统一 cell padding(0,1)。
func applyTableStyles(t *tablev2.Model) {
	s := tablev2.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipglossv2.NormalBorder()).
		BorderForeground(lipglossv2.Color("240")).
		BorderBottom(true).
		Bold(false).
		Padding(0, 1)
	s.Selected = s.Selected.
		Foreground(lipglossv2.Color("229")).
		Background(lipglossv2.Color("57")).
		Bold(false)
	s.Cell = s.Cell.Padding(0, 1)
	t.SetStyles(s)
}

// applyPriceTableStyles 与 applyTableStyles 共用表头样式但关闭选中高亮：
// 价格表展示的是信息而非可选项。注意 Selected 只能是中性样式：
// 若给它加 Padding，被包裹的整行会多出左右空格，破坏各列对齐。
func applyPriceTableStyles(t *tablev2.Model) {
	s := tablev2.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipglossv2.NormalBorder()).
		BorderForeground(lipglossv2.Color("240")).
		BorderBottom(true).
		Bold(false).
		Padding(0, 1)
	s.Selected = lipglossv2.NewStyle()
	s.Cell = s.Cell.Padding(0, 1)
	t.SetStyles(s)
}

// renderTable 给表格视图套上灰色外框后输出。
func renderTable(t tablev2.Model) string {
	return tableFrame.Render(t.View())
}
