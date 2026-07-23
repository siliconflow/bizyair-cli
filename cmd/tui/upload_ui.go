package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib/format"
)

// 等待上传事件（进度/完成）
func waitForUploadEvent(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

// updatePathCompletion 更新路径补全建议
func (m *mainModel) updatePathCompletion(typedPath string) tea.Cmd {
	// 计算补全建议
	matches, dir, prefix := findPathCompletions(
		typedPath,
		m.filepicker.CurrentDirectory,
		m.filepicker.AllowedTypes,
		m.filepicker.DirAllowed,
		m.filepicker.FileAllowed,
	)

	// 更新补全建议
	if len(matches) > 0 {
		m.act.pathCompletionSuggestion = buildCompletionSuggestion(matches)
		m.act.pathMatchCount = len(matches)

		// 更新 filepicker 过滤（提取最后的文件名部分）
		m.filepicker.SetFilterPrefix(prefix)

		// 如果目录改变，重新读取
		if dir != m.filepicker.CurrentDirectory {
			m.filepicker.CurrentDirectory = dir
			return m.filepicker.Init()
		}
	} else {
		m.act.pathCompletionSuggestion = ""
		m.act.pathMatchCount = 0
		m.filepicker.SetFilterPrefix("")
	}

	return nil
}

// applyPathCompletion 应用路径补全建议
// 返回是否需要更新 filepicker 的命令
func (m *mainModel) applyPathCompletion() tea.Cmd {
	if m.act.pathCompletionSuggestion == "" {
		return nil
	}

	suggestion := m.act.pathCompletionSuggestion

	// 设置新值
	m.inpPath.SetValue(suggestion)

	// 将光标移动到行末
	m.inpPath.SetCursor(len(suggestion))

	// 检查补全的路径是否是目录
	if info, err := os.Stat(suggestion); err == nil && info.IsDir() {
		// 是目录，自动添加路径分隔符（如果没有的话）
		if !strings.HasSuffix(suggestion, string(filepath.Separator)) {
			suggestion = ensureTrailingSep(suggestion)
			m.inpPath.SetValue(suggestion)
			m.inpPath.SetCursor(len(suggestion))
		}

		// 更新 filepicker 到这个目录
		m.filepicker.CurrentDirectory = suggestion
		m.filepicker.SetFilterPrefix("")

		// 重新计算补全建议（清空，因为我们已经进入了目录）
		m.act.pathCompletionSuggestion = ""
		m.act.pathMatchCount = 0

		// 返回初始化 filepicker 的命令
		return m.filepicker.Init()
	}

	// 如果不是目录，重新计算补全（用于连续补全）
	return m.updatePathCompletion(suggestion)
}

// renderPathInputWithCompletion 渲染带补全预览的路径输入框
func (m *mainModel) renderPathInputWithCompletion() string {
	var result strings.Builder
	// 先渲染输入框本身
	result.WriteString(m.inpPath.View())
	result.WriteString("\n") // 换行，在下一行显示补全提示

	// 如果有补全建议，在下一行显示灰色预览和匹配数量
	if m.act.pathCompletionSuggestion != "" {
		suggestion := m.act.pathCompletionSuggestion

		// 渲染灰色预览提示（在独立的一行）
		grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		previewText := i18n.T("tui.upload.suggestion", map[string]any{"Path": suggestion})
		result.WriteString(grayStyle.Render(previewText))

		// 显示匹配数量
		if m.act.pathMatchCount > 1 {
			countHint := i18n.T("tui.upload.matches", map[string]any{"Count": m.act.pathMatchCount})
			hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
			result.WriteString(hintStyle.Render(countHint))
		}
	}

	return result.String()
}

// 清除filepicker错误
func clearFilePickerErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg { return clearFilePickerErrorMsg{} })
}

// 重置上传状态
func (m *mainModel) resetUploadState() {
	m.running = false
	m.uploadCh = nil
	m.uploadProg = uploadProgMsg{}
	m.cancelFn = nil
	iw, _ := m.innerSize()
	m.progress.Width = iw - 6
	if m.progress.Width < 10 {
		m.progress.Width = 10
	}
	m.verProgress = nil
	m.verConsumed = nil
	m.verTotal = nil
	m.lastUploadTime = time.Time{}
	m.lastUploadBytes = 0
	m.currentUploadSpeed = 0
	m.verLastTime = nil
	m.verLastBytes = nil
	m.verSpeed = nil
	m.act = actionInputs{}
	m.upStep = stepType

	m.inpName.SetValue("")
	m.inpVersion.SetValue("")
	m.inpCover.SetValue("")
	m.taIntro.SetValue("")
	m.selectedFile = ""
	m.filepicker.Path = ""
	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		m.filepicker.CurrentDirectory = homeDir
		m.inpPath.SetValue(homeDir + "/")
	} else {
		m.inpPath.SetValue("")
	}
	m.coverUrlInputFocused = false
	m.coverPathInputFocused = false
	m.act.pathInputFocused = false
	m.act.useFilePicker = false
	m.act.filePickerErr = nil
	m.act.coverUploadMethod = ""
	m.act.introInputMethod = ""
	m.act.introPathInputFocused = false
	m.act.pathCompletionSuggestion = ""
	m.act.pathMatchCount = 0
	m.filepicker.SetFilterPrefix("")
	m.coverStatus = ""
	m.coverStatusWarning = false
}

// 路径校验与设置
func (m *mainModel) validateAndSetPath(path string) error {
	supportedExts := []string{".safetensors", ".pth", ".bin", ".pt", ".ckpt", ".gguf", ".sft", ".onnx"}
	isSupported := false
	for _, ext := range supportedExts {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			isSupported = true
			break
		}
	}
	if !isSupported {
		return i18n.NewError("tui.error.format_unsupported", map[string]any{"Supported": strings.Join(supportedExts, ", ")}, nil)
	}
	if err := validatePath(path); err != nil {
		return err
	}
	m.act.cur.path = absPath(path)
	m.selectedFile = path
	m.act.filePickerErr = nil
	return nil
}

// 上传进行时视图（复用原进度渲染）
func (m *mainModel) renderUploadRunningView() string {
	var summaryBuilder strings.Builder
	summaryBuilder.WriteString(i18n.T("tui.upload.running_summary", map[string]any{"Type": dash(m.act.u.typ), "Name": dash(m.act.u.name)}) + "\n")
	for i, v := range m.act.versions {
		summaryBuilder.WriteString(i18n.T("tui.upload.running_version", map[string]any{
			"Index": i + 1, "Version": dash(v.version), "Base": dash(v.base), "Cover": dash(v.cover), "Path": dash(v.path), "Intro": dash(truncateToLines(v.intro, 2)),
		}) + "\n")
	}
	summary := summaryBuilder.String()

	var progressSection strings.Builder
	if len(m.verProgress) > 0 {
		if m.uploadProg.total > 0 {
			// 进入文件上传阶段，清空封面状态
			if m.coverStatus != "" {
				m.coverStatus = ""
				m.coverStatusWarning = false
			}
			progressSection.WriteString(i18n.T("tui.upload.current_file", map[string]any{"Index": m.uploadProg.fileIndex, "File": m.uploadProg.fileName}) + "\n")
		} else {
			// 准备阶段，显示封面状态或默认提示
			if m.coverStatus != "" {
				if m.coverStatusWarning {
					// 警告样式（黄色）
					warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
					progressSection.WriteString(warningStyle.Render(m.coverStatus))
					progressSection.WriteString("\n")
				} else {
					progressSection.WriteString(m.coverStatus)
					progressSection.WriteString("\n")
				}
			} else {
				progressSection.WriteString(i18n.T("tui.upload.preparing", nil) + "\n")
			}
		}
		for i := range m.verProgress {
			versionLabel := ""
			if i >= 0 && i < len(m.act.versions) {
				versionLabel = m.act.versions[i].version
			}
			consumed := m.verConsumed[i]
			total := m.verTotal[i]
			percent := 0.0
			if total > 0 {
				percent = float64(consumed) / float64(total)
			}
			bar := m.verProgress[i].View()
			prefix := "  "
			if i == m.uploadProg.verIdx {
				prefix = "▶ "
			}
			progressSection.WriteString(i18n.T("tui.upload.version_progress", map[string]any{"Prefix": prefix, "Current": i + 1, "Total": len(m.verProgress), "Version": dash(versionLabel)}) + "\n")
			if total > 0 {
				progressSection.WriteString(fmt.Sprintf("%s%s\n", prefix, bar))
				// 显示进度百分比、已上传/总大小和速率
				speed := int64(0)
				if i < len(m.verSpeed) {
					speed = m.verSpeed[i]
				}
				progressSection.WriteString(fmt.Sprintf("%s%.1f%% (%s/%s) %s/s\n", prefix, percent*100, format.FormatBytes(consumed), format.FormatBytes(total), format.FormatBytes(speed)))
			} else {
				progressSection.WriteString(fmt.Sprintf("%s%s\n", prefix, bar))
			}
		}
	} else {
		var fileLine string
		var progLine string
		var speedLine string
		if m.uploadProg.total > 0 {
			// 进入文件上传阶段，清空封面状态
			if m.coverStatus != "" {
				m.coverStatus = ""
				m.coverStatusWarning = false
			}
			percent := float64(m.uploadProg.consumed) / float64(m.uploadProg.total)
			fileLine = fmt.Sprintf("(%s) %s", m.uploadProg.fileIndex, m.uploadProg.fileName)
			progLine = m.progress.View()
			speedLine = fmt.Sprintf("%.1f%% (%s/%s) %s/s", percent*100, format.FormatBytes(m.uploadProg.consumed), format.FormatBytes(m.uploadProg.total), format.FormatBytes(m.currentUploadSpeed))
		} else {
			// 准备阶段，显示封面状态或默认提示
			if m.coverStatus != "" {
				if m.coverStatusWarning {
					// 警告样式（黄色）
					warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
					fileLine = warningStyle.Render(m.coverStatus)
				} else {
					fileLine = m.coverStatus
				}
			} else {
				fileLine = i18n.T("tui.upload.preparing", nil)
			}
			progLine = m.progress.View()
			speedLine = ""
		}
		progressSection.WriteString(fileLine)
		progressSection.WriteString("\n")
		progressSection.WriteString(progLine)
		if speedLine != "" {
			progressSection.WriteString("\n")
			progressSection.WriteString(speedLine)
		}
	}

	var hint string
	if m.canceling {
		hint = m.hintStyle.Render(i18n.T("tui.upload.canceling", nil))
	} else {
		hint = m.hintStyle.Render(i18n.T("tui.upload.cancel_hint", nil))
	}

	return m.titleStyle.Render(i18n.T("tui.upload.running_title", nil)) + "\n\n" + summary + "\n\n" + progressSection.String() + "\n\n" + hint
}

// 根据当前动作处理输入与触发命令
func (m *mainModel) updateActionInputs(msg tea.Msg) tea.Cmd {
	switch m.currentAction {
	case actionUpload:
		return m.updateUploadInputs(msg)
	default:
		return nil
	}
}

// 渲染动作视图
func (m *mainModel) renderActionView() string {
	switch m.currentAction {
	case actionUpload:
		return m.renderUploadStepsView()
	default:
		return ""
	}
}

// 上传交互更新（截断原大函数到子方法）
func (m *mainModel) updateUploadInputs(msg tea.Msg) tea.Cmd {
	switch m.upStep {
	case stepType:
		var cmd tea.Cmd
		m.typeList, cmd = m.typeList.Update(msg)
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "enter":
				if it, ok := m.typeList.SelectedItem().(listItem); ok {
					m.act.u.typ = it.title
					m.upStep = stepName
					return m.inpName.Focus()
				}
			case "esc":
				m.step = mainStepMenu
				m.act = actionInputs{}
				return nil
			}
		}
		return cmd
	case stepName:
		var cmd tea.Cmd
		m.inpName, cmd = m.inpName.Update(msg)
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "enter":
				name := strings.TrimSpace(m.inpName.Value())
				if err := validateName(name); err != nil {
					m.err = err
					return nil
				}
				m.act.u.name = name
				// 调用后端校验模型名是否重复
				m.running = true
				return checkModelExists(m.ctx, m.baseDomain, m.apiKey, name, m.act.u.typ)
			case "esc":
				m.upStep = stepType
				return nil
			}
		}
		return cmd
	case stepVersion:
		var cmd tea.Cmd
		m.inpVersion, cmd = m.inpVersion.Update(msg)
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "enter":
				v := strings.TrimSpace(m.inpVersion.Value())
				if v == "" {
					v = "v1.0"
					m.inpVersion.SetValue(v)
				}
				m.act.cur.version = v
				m.upStep = stepBase
				return nil
			case "esc":
				// 判断是否在添加更多版本
				if len(m.act.versions) > 0 {
					// 正在添加更多版本，需要恢复状态并回退到 stepAskMore
					// 取出最后一个版本，恢复为当前版本
					lastIdx := len(m.act.versions) - 1
					m.act.cur = m.act.versions[lastIdx]
					m.act.versions = m.act.versions[:lastIdx]

					// 恢复输入框的值
					m.inpVersion.SetValue(m.act.cur.version)
					m.inpCover.SetValue(m.act.cur.cover)
					m.taIntro.SetValue(m.act.cur.intro)

					m.upStep = stepAskMore
					return nil
				} else {
					// 首次输入版本，回退到 stepName
					m.upStep = stepName
					return m.inpName.Focus()
				}
			}
		}
		return cmd
	case stepBase:
		var cmd tea.Cmd
		m.baseList, cmd = m.baseList.Update(msg)
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "enter":
				if it, ok := m.baseList.SelectedItem().(listItem); ok {
					m.act.cur.base = it.title
					m.upStep = stepCoverMethod
					return nil
				}
			case "esc":
				m.upStep = stepVersion
				return nil
			}
		}
		return cmd
	case stepCoverMethod:
		var cmd tea.Cmd
		m.coverMethodList, cmd = m.coverMethodList.Update(msg)
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "enter":
				if it, ok := m.coverMethodList.SelectedItem().(listItem); ok {
					if it.value == "url" {
						m.act.coverUploadMethod = "url"
						m.coverUrlInputFocused = true
						m.coverPathInputFocused = false
						m.inpCover.SetValue("")
						m.upStep = stepCover
						return m.inpCover.Focus()
					} else {
						m.act.coverUploadMethod = "local"
						m.act.useFilePicker = true
						m.coverUrlInputFocused = false
						m.coverPathInputFocused = true
						m.inpPath.SetValue(ensureTrailingSep(m.filepicker.CurrentDirectory))
						m.filepicker.AllowedTypes = []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp4", ".webm", ".mov"}
						m.filepicker.DirAllowed = true
						m.filepicker.FileAllowed = true
						m.filepicker.Path = ""
						// 清除补全状态
						m.act.pathCompletionSuggestion = ""
						m.act.pathMatchCount = 0
						m.filepicker.SetFilterPrefix("")
						m.upStep = stepCover
						return tea.Batch(m.inpPath.Focus(), m.filepicker.Init())
					}
				}
			case "esc":
				m.upStep = stepBase
				return nil
			}
		}
		return cmd
	case stepCover:
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "esc":
				m.coverUrlInputFocused = false
				m.coverPathInputFocused = false
				m.act.filePickerErr = nil
				m.filepicker.Path = ""
				// 清除补全状态
				m.act.pathCompletionSuggestion = ""
				m.act.pathMatchCount = 0
				m.filepicker.SetFilterPrefix("")
				m.upStep = stepCoverMethod
				return nil
			}
		}

		// 根据上传方式分别处理
		switch m.act.coverUploadMethod {
		case "url":
			// URL 上传模式
			var urlCmd tea.Cmd
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "enter":
					raw := strings.TrimSpace(m.inpCover.Value())
					first := raw
					warned := false
					if i := strings.Index(raw, ";"); i >= 0 {
						first = strings.TrimSpace(raw[:i])
						m.inpCover.SetValue(first)
						m.act.filePickerErr = i18n.NewError("tui.error.cover_multiple_urls", nil, nil)
						warned = true
					}
					if _, err := os.Stat(first); err == nil {
						m.act.filePickerErr = i18n.NewError("tui.error.cover_local_in_url", nil, nil)
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					if !IsHTTPURL(first) {
						m.act.filePickerErr = i18n.NewError("tui.error.cover_url_required", nil, nil)
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					check := first
					if q := strings.Index(check, "?"); q >= 0 {
						check = check[:q]
					}
					if !isSupportedCoverFormat(check) {
						m.act.filePickerErr = i18n.NewError("tui.error.cover_url_format", map[string]any{"URL": first, "Supported": getSupportedCoverFormats()}, nil)
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					m.act.cur.cover = first
					m.upStep = stepIntroMethod
					if warned {
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					return nil
				}
			}
			m.inpCover, urlCmd = m.inpCover.Update(msg)
			return urlCmd
		case "local":
			// 本地文件上传模式
			var pathCmd, fpCmd tea.Cmd
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "tab": // Tab 补全
					if m.coverPathInputFocused && m.act.pathCompletionSuggestion != "" {
						// 应用补全建议（包括光标移动到行末、自动进入目录等）
						fpCmd = m.applyPathCompletion()
					}
				case "ctrl+p": // Ctrl+P 切换焦点
					if m.coverPathInputFocused {
						path := strings.TrimSpace(m.inpPath.Value())
						if path != "" {
							if info, err := os.Stat(path); err == nil && info.IsDir() {
								m.filepicker.CurrentDirectory = path
								m.coverPathInputFocused = false
								m.inpPath.Blur()
								// 清除补全建议，但保留过滤前缀以维持 filepicker 的过滤状态
								m.act.pathCompletionSuggestion = ""
								m.act.pathMatchCount = 0
								return m.filepicker.Init()
							}
						}
						m.coverPathInputFocused = false
						m.inpPath.Blur()
						// 保持 FilterPrefix 不变，让 filepicker 保持过滤状态
						return nil
					} else {
						m.coverPathInputFocused = true
						return m.inpPath.Focus()
					}
				case "enter":
					if m.coverPathInputFocused && m.inpPath.Value() != "" {
						p := strings.TrimSpace(m.inpPath.Value())
						info, err := os.Stat(p)
						if err != nil {
							m.act.filePickerErr = i18n.NewError("tui.error.path_missing", map[string]any{"Path": p}, err)
							return clearFilePickerErrorAfter(3 * time.Second)
						}
						if info.IsDir() {
							m.filepicker.CurrentDirectory = p
							m.filepicker.SetFilterPrefix("")
							m.act.pathCompletionSuggestion = ""
							m.act.pathMatchCount = 0
							return m.filepicker.Init()
						}
						if err := validateCoverFile(p); err != nil {
							m.act.filePickerErr = err
							return clearFilePickerErrorAfter(3 * time.Second)
						}
						ap := absPath(p)
						m.inpCover.SetValue(ap)
						m.act.cur.cover = ap
						// 清除补全状态
						m.act.pathCompletionSuggestion = ""
						m.act.pathMatchCount = 0
						m.filepicker.SetFilterPrefix("")
						m.upStep = stepIntroMethod
						return nil
					}
				}
			}
			if m.coverPathInputFocused {
				m.inpPath, pathCmd = m.inpPath.Update(msg)
				typedPath := strings.TrimSpace(m.inpPath.Value())
				// 实时更新补全建议
				fpCmd = m.updatePathCompletion(typedPath)
				// 保持旧的目录切换逻辑
				if typedPath != "" {
					if info, err := os.Stat(typedPath); err == nil && info.IsDir() {
						if filepath.Clean(m.filepicker.CurrentDirectory) != filepath.Clean(typedPath) {
							m.filepicker.CurrentDirectory = typedPath
							fpCmd = m.filepicker.Init()
						}
					}
				}
			}
			if !m.coverPathInputFocused {
				// 拦截 Back 键：如果有过滤条件，先清除过滤而不是返回上级目录
				if km, ok := msg.(tea.KeyMsg); ok {
					keyStr := km.String()
					if (keyStr == "h" || keyStr == "backspace" || keyStr == "left") && m.filepicker.FilterPrefix != "" {
						// 清除过滤，显示当前目录的所有文件
						m.filepicker.SetFilterPrefix("")
						m.act.pathCompletionSuggestion = ""
						m.act.pathMatchCount = 0
						m.inpPath.SetValue(ensureTrailingSep(m.filepicker.CurrentDirectory))
						return nil
					}
				}

				oldDir := m.filepicker.CurrentDirectory
				var fc tea.Cmd
				m.filepicker, fc = m.filepicker.Update(msg)
				if fc != nil {
					fpCmd = fc
				}
				if m.filepicker.CurrentDirectory != oldDir {
					m.inpPath.SetValue(ensureTrailingSep(m.filepicker.CurrentDirectory))
					// 目录改变时清除补全和过滤
					m.act.pathCompletionSuggestion = ""
					m.act.pathMatchCount = 0
					m.filepicker.SetFilterPrefix("")
				}
				if did, p := m.filepicker.DidSelectFile(msg); did {
					if info, err := os.Stat(p); err == nil && !info.IsDir() {
						if err := validateCoverFile(p); err != nil {
							m.act.filePickerErr = err
							return clearFilePickerErrorAfter(3 * time.Second)
						}
						ap := absPath(p)
						m.inpCover.SetValue(ap)
						m.act.cur.cover = ap
						// 清除补全状态
						m.act.pathCompletionSuggestion = ""
						m.act.pathMatchCount = 0
						m.filepicker.SetFilterPrefix("")
						m.upStep = stepIntroMethod
						return nil
					}
				}
				if didSelect, p := m.filepicker.DidSelectDisabledFile(msg); didSelect {
					if info, err := os.Stat(p); err == nil && !info.IsDir() {
						m.act.filePickerErr = i18n.NewError("tui.error.file_format", map[string]any{"Path": p}, nil)
						return clearFilePickerErrorAfter(3 * time.Second)
					}
				}
			}
			return tea.Batch(pathCmd, fpCmd)
		}
		return nil
	case stepIntroMethod:
		var cmd tea.Cmd
		m.introMethodList, cmd = m.introMethodList.Update(msg)
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "enter":
				if it, ok := m.introMethodList.SelectedItem().(listItem); ok {
					if it.value == "file" {
						m.act.introInputMethod = "file"
						m.act.useFilePicker = true
						m.act.introPathInputFocused = true
						m.inpPath.SetValue(ensureTrailingSep(m.filepicker.CurrentDirectory))
						m.filepicker.AllowedTypes = []string{".txt", ".md"}
						m.filepicker.DirAllowed = true
						m.filepicker.FileAllowed = true
						m.filepicker.Path = ""
						// 清除补全状态
						m.act.pathCompletionSuggestion = ""
						m.act.pathMatchCount = 0
						m.filepicker.SetFilterPrefix("")
						m.upStep = stepIntro
						return tea.Batch(m.inpPath.Focus(), m.filepicker.Init())
					} else {
						m.act.introInputMethod = "direct"
						m.upStep = stepIntro
						return m.taIntro.Focus()
					}
				}
			case "esc":
				m.upStep = stepCover
				switch m.act.coverUploadMethod {
				case "url":
					return m.inpCover.Focus()
				case "local":
					// 恢复封面文件选择的配置
					m.coverPathInputFocused = true
					m.act.useFilePicker = true
					m.filepicker.AllowedTypes = []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp4", ".webm", ".mov"}
					m.filepicker.DirAllowed = true
					m.filepicker.FileAllowed = true
					return tea.Batch(m.inpPath.Focus(), m.filepicker.Init())
				}
				return nil
			}
		}
		return cmd
	case stepIntro:
		if m.act.introInputMethod == "file" {
			// 文件导入模式
			var pathCmd, fpCmd tea.Cmd
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "esc":
					m.act.useFilePicker = false
					m.act.introPathInputFocused = false
					m.act.filePickerErr = nil
					m.filepicker.Path = ""
					m.filepicker.SetFilterPrefix("")
					m.act.pathCompletionSuggestion = ""
					m.act.pathMatchCount = 0
					m.taIntro.SetValue("")
					m.upStep = stepIntroMethod
					return nil
				case "tab": // Tab 补全
					if m.act.introPathInputFocused && m.act.pathCompletionSuggestion != "" {
						// 应用补全建议（包括光标移动到行末、自动进入目录等）
						fpCmd = m.applyPathCompletion()
					}
				case "ctrl+p": // Ctrl+P 切换焦点
					if m.act.introPathInputFocused {
						path := strings.TrimSpace(m.inpPath.Value())
						if path != "" {
							if info, err := os.Stat(path); err == nil && info.IsDir() {
								m.filepicker.CurrentDirectory = path
								m.act.introPathInputFocused = false
								m.inpPath.Blur()
								// 清除补全建议，但保留过滤前缀以维持 filepicker 的过滤状态
								m.act.pathCompletionSuggestion = ""
								m.act.pathMatchCount = 0
								return m.filepicker.Init()
							}
						}
						m.act.introPathInputFocused = false
						m.inpPath.Blur()
						// 保持 FilterPrefix 不变，让 filepicker 保持过滤状态
						return nil
					} else {
						m.act.introPathInputFocused = true
						return m.inpPath.Focus()
					}
				case "enter":
					if m.act.introPathInputFocused && m.inpPath.Value() != "" {
						p := strings.TrimSpace(m.inpPath.Value())
						info, err := os.Stat(p)
						if err != nil {
							m.act.filePickerErr = i18n.NewError("tui.error.path_missing", map[string]any{"Path": p}, err)
							return clearFilePickerErrorAfter(3 * time.Second)
						}
						if info.IsDir() {
							m.filepicker.CurrentDirectory = p
							m.filepicker.SetFilterPrefix("")
							m.act.pathCompletionSuggestion = ""
							m.act.pathMatchCount = 0
							return m.filepicker.Init()
						}
						if err := validateIntroFile(p); err != nil {
							m.act.filePickerErr = err
							return clearFilePickerErrorAfter(3 * time.Second)
						}
						content, err := readIntroFile(p)
						if err != nil {
							m.act.filePickerErr = err
							return clearFilePickerErrorAfter(3 * time.Second)
						}
						if strings.TrimSpace(content) == "" {
							m.act.filePickerErr = i18n.NewError("tui.error.intro_empty", nil, nil)
							return clearFilePickerErrorAfter(3 * time.Second)
						}
						m.taIntro.SetValue(content)
						m.act.introInputMethod = "direct"
						m.act.useFilePicker = false
						m.act.introPathInputFocused = false
						return m.taIntro.Focus()
					}
				}
			}
			if m.act.introPathInputFocused {
				m.inpPath, pathCmd = m.inpPath.Update(msg)
				typedPath := strings.TrimSpace(m.inpPath.Value())
				// 实时更新补全建议
				fpCmd = m.updatePathCompletion(typedPath)
				// 保持旧的目录切换逻辑
				if typedPath != "" {
					if info, err := os.Stat(typedPath); err == nil && info.IsDir() {
						if filepath.Clean(m.filepicker.CurrentDirectory) != filepath.Clean(typedPath) {
							m.filepicker.CurrentDirectory = typedPath
							fpCmd = m.filepicker.Init()
						}
					}
				}
			}
			if !m.act.introPathInputFocused {
				// 拦截 Back 键：如果有过滤条件，先清除过滤而不是返回上级目录
				if km, ok := msg.(tea.KeyMsg); ok {
					keyStr := km.String()
					if (keyStr == "h" || keyStr == "backspace" || keyStr == "left") && m.filepicker.FilterPrefix != "" {
						// 清除过滤，显示当前目录的所有文件
						m.filepicker.SetFilterPrefix("")
						m.act.pathCompletionSuggestion = ""
						m.act.pathMatchCount = 0
						m.inpPath.SetValue(ensureTrailingSep(m.filepicker.CurrentDirectory))
						return nil
					}
				}

				oldDir := m.filepicker.CurrentDirectory
				var fc tea.Cmd
				m.filepicker, fc = m.filepicker.Update(msg)
				if fc != nil {
					fpCmd = fc
				}
				if m.filepicker.CurrentDirectory != oldDir {
					m.inpPath.SetValue(ensureTrailingSep(m.filepicker.CurrentDirectory))
					// 目录改变时清除补全和过滤
					m.act.pathCompletionSuggestion = ""
					m.act.pathMatchCount = 0
					m.filepicker.SetFilterPrefix("")
				}
				if did, p := m.filepicker.DidSelectFile(msg); did {
					if info, err := os.Stat(p); err == nil && !info.IsDir() {
						if err := validateIntroFile(p); err != nil {
							m.act.filePickerErr = err
							return clearFilePickerErrorAfter(3 * time.Second)
						}
						content, err := readIntroFile(p)
						if err != nil {
							m.act.filePickerErr = err
							return clearFilePickerErrorAfter(3 * time.Second)
						}
						if strings.TrimSpace(content) == "" {
							m.act.filePickerErr = i18n.NewError("tui.error.intro_empty", nil, nil)
							return clearFilePickerErrorAfter(3 * time.Second)
						}
						m.taIntro.SetValue(content)
						m.act.introInputMethod = "direct"
						m.act.useFilePicker = false
						m.act.introPathInputFocused = false
						return m.taIntro.Focus()
					}
				}
				if didSelect, p := m.filepicker.DidSelectDisabledFile(msg); didSelect {
					if info, err := os.Stat(p); err == nil && !info.IsDir() {
						m.act.filePickerErr = i18n.NewError("tui.error.file_format", map[string]any{"Path": p}, nil)
						return clearFilePickerErrorAfter(3 * time.Second)
					}
				}
			}
			return tea.Batch(pathCmd, fpCmd)
		} else {
			// 直接输入模式
			var cmd tea.Cmd
			m.taIntro, cmd = m.taIntro.Update(msg)
			if km, ok := msg.(tea.KeyMsg); ok {
				switch km.String() {
				case "ctrl+s":
					intro := strings.TrimSpace(m.taIntro.Value())
					if intro == "" {
						m.err = i18n.NewError("tui.error.intro_required", nil, nil)
						return nil
					}
					if len([]rune(intro)) > 5000 {
						intro = string([]rune(intro)[:5000])
					}
					m.act.cur.intro = intro
					m.act.useFilePicker = true
					m.act.pathInputFocused = true
					m.inpPath.SetValue(ensureTrailingSep(m.filepicker.CurrentDirectory))
					m.filepicker.AllowedTypes = []string{".safetensors", ".pth", ".bin", ".pt", ".ckpt", ".gguf", ".sft", ".onnx"}
					m.filepicker.DirAllowed = true
					m.filepicker.FileAllowed = true
					m.filepicker.Path = ""
					// 清除补全状态
					m.act.pathCompletionSuggestion = ""
					m.act.pathMatchCount = 0
					m.filepicker.SetFilterPrefix("")
					m.upStep = stepPath
					return tea.Batch(m.inpPath.Focus(), m.filepicker.Init())
				case "esc":
					m.upStep = stepIntroMethod
					return nil
				}
			}
			return cmd
		}
	case stepPath:
		var pathCmd, fpCmd tea.Cmd
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "esc":
				m.act.useFilePicker = false
				m.act.pathInputFocused = false
				m.act.filePickerErr = nil
				m.filepicker.Path = ""
				m.filepicker.SetFilterPrefix("")
				m.act.pathCompletionSuggestion = ""
				m.act.pathMatchCount = 0
				m.upStep = stepIntro
				// 根据之前的输入方式恢复状态
				if m.act.introInputMethod == "file" {
					// 恢复文件导入模式
					m.act.useFilePicker = true
					m.act.introPathInputFocused = true
					m.filepicker.AllowedTypes = []string{".txt", ".md"}
					m.filepicker.DirAllowed = true
					m.filepicker.FileAllowed = true
					return tea.Batch(m.inpPath.Focus(), m.filepicker.Init())
				} else {
					// 返回直接输入模式
					return m.taIntro.Focus()
				}
			case "ctrl+r":
				path := strings.TrimSpace(m.inpPath.Value())
				if path != "" {
					if info, err := os.Stat(path); err == nil && info.IsDir() {
						m.filepicker.CurrentDirectory = path
						return m.filepicker.Init()
					} else {
						m.act.filePickerErr = i18n.NewError("tui.error.directory_invalid", map[string]any{"Path": path}, nil)
						return clearFilePickerErrorAfter(3 * time.Second)
					}
				}
				m.act.cur.path = absPath(path)
				m.upStep = stepPublic
				return nil
			case "tab": // Tab 补全
				if m.act.pathInputFocused && m.act.pathCompletionSuggestion != "" {
					// 应用补全建议（包括光标移动到行末、自动进入目录等）
					fpCmd = m.applyPathCompletion()
				}
			case "ctrl+p": // Ctrl+P 切换焦点
				if m.act.pathInputFocused {
					path := strings.TrimSpace(m.inpPath.Value())
					if path != "" {
						if info, err := os.Stat(path); err == nil && info.IsDir() {
							m.filepicker.CurrentDirectory = path
							m.act.pathInputFocused = false
							m.inpPath.Blur()
							// 清除补全建议，但保留过滤前缀以维持 filepicker 的过滤状态
							m.act.pathCompletionSuggestion = ""
							m.act.pathMatchCount = 0
							return m.filepicker.Init()
						}
					}
					m.act.pathInputFocused = false
					m.inpPath.Blur()
					// 保持 FilterPrefix 不变，让 filepicker 保持过滤状态
					return nil
				} else {
					m.act.pathInputFocused = true
					return m.inpPath.Focus()
				}
			case "enter":
				if m.act.pathInputFocused && m.inpPath.Value() != "" {
					path := strings.TrimSpace(m.inpPath.Value())
					info, err := os.Stat(path)
					if err != nil {
						m.act.filePickerErr = i18n.NewError("tui.error.path_missing", map[string]any{"Path": path}, err)
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					if info.IsDir() {
						m.filepicker.CurrentDirectory = path
						m.filepicker.SetFilterPrefix("")
						m.act.pathCompletionSuggestion = ""
						m.act.pathMatchCount = 0
						return m.filepicker.Init()
					}
					if err := m.validateAndSetPath(path); err != nil {
						m.act.filePickerErr = err
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					// 文件选择成功，进入下一步
					m.upStep = stepPublic
					return nil
				}
			}
		}
		if m.act.pathInputFocused {
			m.inpPath, pathCmd = m.inpPath.Update(msg)
			typedPath := strings.TrimSpace(m.inpPath.Value())
			// 实时更新补全建议
			fpCmd = m.updatePathCompletion(typedPath)
			// 保持旧的目录切换逻辑
			if typedPath != "" {
				if info, err := os.Stat(typedPath); err == nil && info.IsDir() {
					if filepath.Clean(m.filepicker.CurrentDirectory) != filepath.Clean(typedPath) {
						m.filepicker.CurrentDirectory = typedPath
						fpCmd = m.filepicker.Init()
					}
				}
			}
		}
		if !m.act.pathInputFocused {
			// 拦截 Back 键：如果有过滤条件，先清除过滤而不是返回上级目录
			if km, ok := msg.(tea.KeyMsg); ok {
				keyStr := km.String()
				if (keyStr == "h" || keyStr == "backspace" || keyStr == "left") && m.filepicker.FilterPrefix != "" {
					// 清除过滤，显示当前目录的所有文件
					m.filepicker.SetFilterPrefix("")
					m.act.pathCompletionSuggestion = ""
					m.act.pathMatchCount = 0
					m.inpPath.SetValue(ensureTrailingSep(m.filepicker.CurrentDirectory))
					return nil
				}
			}

			oldDir := m.filepicker.CurrentDirectory
			var fc tea.Cmd
			m.filepicker, fc = m.filepicker.Update(msg)
			if fc != nil {
				fpCmd = fc
			}
			if m.filepicker.CurrentDirectory != oldDir {
				m.inpPath.SetValue(ensureTrailingSep(m.filepicker.CurrentDirectory))
				// 目录改变时清除补全和过滤
				m.act.pathCompletionSuggestion = ""
				m.act.pathMatchCount = 0
				m.filepicker.SetFilterPrefix("")
			}
			if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
				if info, err := os.Stat(path); err == nil && !info.IsDir() {
					if err := m.validateAndSetPath(path); err != nil {
						m.act.filePickerErr = err
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					m.upStep = stepPublic
					return nil
				}
			}
			if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
				if info, err := os.Stat(path); err == nil && !info.IsDir() {
					m.act.filePickerErr = i18n.NewError("tui.error.file_format", map[string]any{"Path": path}, nil)
					return clearFilePickerErrorAfter(3 * time.Second)
				}
			}
		}
		return tea.Batch(pathCmd, fpCmd)
	case stepPublic:
		var cmd tea.Cmd
		m.publicList, cmd = m.publicList.Update(msg)
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "enter":
				if it, ok := m.publicList.SelectedItem().(listItem); ok {
					m.act.cur.public = it.value == "public"
					m.upStep = stepAskMore
					return nil
				}
			case "esc":
				m.upStep = stepPath
				m.act.useFilePicker = true
				return nil
			}
		}
		return cmd
	case stepAskMore:
		var cmd tea.Cmd
		m.moreList, cmd = m.moreList.Update(msg)
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "enter":
				if it, ok := m.moreList.SelectedItem().(listItem); ok {
					if it.value == "add" {
						m.act.versions = append(m.act.versions, m.act.cur)
						next := fmt.Sprintf("v%d.0", len(m.act.versions)+1)
						m.act.cur = versionItem{}
						m.inpVersion.SetValue(next)
						m.inpCover.SetValue("")
						m.taIntro.SetValue("")
						m.upStep = stepVersion
						return nil
					}
					m.act.versions = append(m.act.versions, m.act.cur)
					m.upStep = stepConfirm
					m.act.confirming = true
					return nil
				}
			case "esc":
				m.upStep = stepPublic
				return nil
			}
		}
		return cmd
	case stepConfirm:
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "enter":
				m.running = true
				return runUploadActionMulti(m.ctx, m.baseDomain, m.act.u, m.act.versions)
			case "esc":
				m.act.confirming = false
				m.upStep = stepAskMore
				return nil
			}
		}
		return nil
	default:
		return nil
	}
}

// 渲染上传各步骤视图（从旧 View 拆出）
func (m *mainModel) renderUploadStepsView() string {
	switch m.upStep {
	case stepType:
		if _, ih := m.innerSize(); ih > 0 {
			h := ih - 12
			if h < 5 {
				h = 5
			}
			m.typeList.SetHeight(h)
		}
		return m.titleStyle.Render(uploadTitle(1, "tui.upload.section.type")) + "\n\n" + m.typeList.View() + "\n" + m.hintStyle.Render(i18n.T("tui.hint.confirm_back", nil))
	case stepName:
		return m.titleStyle.Render(uploadTitle(2, "tui.upload.section.name")) + "\n\n" + m.inpName.View() + "\n" + m.hintStyle.Render(i18n.T("tui.hint.confirm_back", nil))
	case stepVersion:
		return m.titleStyle.Render(uploadTitle(3, "tui.upload.section.version")) + "\n\n" + m.inpVersion.View() + "\n" + m.hintStyle.Render(i18n.T("tui.hint.confirm_back", nil))
	case stepBase:
		if _, ih := m.innerSize(); ih > 0 {
			h := ih - 12
			if h < 5 {
				h = 5
			}
			m.baseList.SetHeight(h)
		}
		// 如果基础模型类型还在加载中
		if m.loadingBaseModelTypes {
			return m.titleStyle.Render(uploadTitle(4, "tui.upload.section.base")) + "\n\n" + m.sp.View() + " " + i18n.T("tui.upload.load_base_models", nil) + "\n" + m.hintStyle.Render(i18n.T("tui.hint.confirm_back", nil))
		}
		// 如果列表为空（加载失败），显示提示
		if len(m.baseModelTypes) == 0 {
			return m.titleStyle.Render(uploadTitle(4, "tui.upload.section.base")) + "\n\n" + m.baseList.View() + "\n" + m.hintStyle.Render(i18n.T("tui.upload.local_base_models", nil))
		}
		return m.titleStyle.Render(uploadTitle(4, "tui.upload.section.base")) + "\n\n" + m.baseList.View() + "\n" + m.hintStyle.Render(i18n.T("tui.upload.select_base_model", nil))
	case stepCoverMethod:
		if _, ih := m.innerSize(); ih > 0 {
			h := ih - 12
			if h < 5 {
				h = 5
			}
			m.coverMethodList.SetHeight(h)
		}
		return m.titleStyle.Render(uploadTitle(5, "tui.upload.section.cover_method")) + "\n\n" + m.coverMethodList.View() + "\n" + m.hintStyle.Render(i18n.T("tui.upload.select_then_enter", nil))
	case stepCover:
		var content strings.Builder

		switch m.act.coverUploadMethod {
		case "url":
			content.WriteString(m.titleStyle.Render(uploadTitle(6, "tui.upload.section.cover_url")))
			content.WriteString("\n\n")
			content.WriteString(i18n.T("tui.upload.cover_url_label", nil) + "\n")
			content.WriteString(m.inpCover.View())
			content.WriteString("\n\n")
			if m.act.filePickerErr != nil {
				content.WriteString(m.filepicker.Styles.DisabledFile.Render(m.act.filePickerErr.Error()))
				content.WriteString("\n\n")
			}
			content.WriteString(m.hintStyle.Render(i18n.T("tui.upload.cover_url_hint", nil)))
		case "local":
			content.WriteString(m.titleStyle.Render(uploadTitle(6, "tui.upload.section.cover_local")))
			content.WriteString("\n\n")
			pathLabel := i18n.T("tui.upload.local_path_label", nil)
			if m.coverPathInputFocused {
				pathLabel = m.titleStyle.Render("► " + pathLabel + i18n.T("tui.upload.focused", nil))
			} else {
				pathLabel = m.hintStyle.Render(pathLabel)
			}
			content.WriteString(pathLabel)
			content.WriteString("\n")
			if m.coverPathInputFocused {
				content.WriteString(m.renderPathInputWithCompletion())
				content.WriteString("\n") // 添加额外换行
			} else {
				content.WriteString(m.inpPath.View())
				content.WriteString("\n\n")
			}

			pickerLabel := i18n.T("tui.upload.picker_label", nil)
			if !m.coverPathInputFocused {
				pickerLabel = m.titleStyle.Render("► " + pickerLabel + i18n.T("tui.upload.focused", nil))
			} else {
				pickerLabel = m.hintStyle.Render(pickerLabel)
			}
			content.WriteString(pickerLabel)
			content.WriteString("\n")
			if m.act.filePickerErr != nil {
				content.WriteString(m.filepicker.Styles.DisabledFile.Render(m.act.filePickerErr.Error()))
				content.WriteString("\n")
			}
			content.WriteString(m.filepicker.View())
			content.WriteString("\n")

			if m.coverPathInputFocused {
				content.WriteString(m.hintStyle.Render(i18n.T("tui.upload.cover_path_hint", nil)))
			} else {
				content.WriteString(m.hintStyle.Render(i18n.T("tui.upload.cover_picker_hint", nil)))
			}
		}
		return content.String()
	case stepIntroMethod:
		if _, ih := m.innerSize(); ih > 0 {
			h := ih - 12
			if h < 5 {
				h = 5
			}
			m.introMethodList.SetHeight(h)
		}
		return m.titleStyle.Render(uploadTitle(7, "tui.upload.section.intro_method")) + "\n\n" + m.introMethodList.View() + "\n" + m.hintStyle.Render(i18n.T("tui.upload.select_then_enter", nil))
	case stepIntro:
		if m.act.introInputMethod == "file" {
			// 文件导入模式渲染
			var content strings.Builder
			content.WriteString(m.titleStyle.Render(uploadTitle(8, "tui.upload.section.intro_file")))
			content.WriteString("\n\n")
			pathLabel := i18n.T("tui.upload.local_path_label", nil)
			if m.act.introPathInputFocused {
				pathLabel = m.titleStyle.Render("► " + pathLabel + i18n.T("tui.upload.focused", nil))
			} else {
				pathLabel = m.hintStyle.Render(pathLabel)
			}
			content.WriteString(pathLabel)
			content.WriteString("\n")
			if m.act.introPathInputFocused {
				content.WriteString(m.renderPathInputWithCompletion())
				content.WriteString("\n") // 添加额外换行
			} else {
				content.WriteString(m.inpPath.View())
				content.WriteString("\n\n")
			}

			pickerLabel := i18n.T("tui.upload.picker_label", nil)
			if !m.act.introPathInputFocused {
				pickerLabel = m.titleStyle.Render("► " + pickerLabel + i18n.T("tui.upload.focused", nil))
			} else {
				pickerLabel = m.hintStyle.Render(pickerLabel)
			}
			content.WriteString(pickerLabel)
			content.WriteString("\n")
			if m.act.filePickerErr != nil {
				content.WriteString(m.filepicker.Styles.DisabledFile.Render(m.act.filePickerErr.Error()))
				content.WriteString("\n")
			}
			content.WriteString(m.filepicker.View())
			content.WriteString("\n")

			if m.act.introPathInputFocused {
				content.WriteString(m.hintStyle.Render(i18n.T("tui.upload.intro_path_hint", nil)))
			} else {
				content.WriteString(m.hintStyle.Render(i18n.T("tui.upload.intro_picker_hint", nil)))
			}
			return content.String()
		} else {
			// 直接输入模式渲染
			charCount := len([]rune(m.taIntro.Value()))
			charInfo := i18n.T("tui.upload.intro_count", map[string]any{"Count": charCount})
			return m.titleStyle.Render(uploadTitle(8, "tui.upload.section.intro")) + " " + m.hintStyle.Render(charInfo) + "\n\n" + m.taIntro.View() + "\n" + m.hintStyle.Render(i18n.T("tui.upload.intro_editor_hint", nil))
		}
	case stepPath:
		var content strings.Builder
		content.WriteString(m.titleStyle.Render(uploadTitle(9, "tui.upload.section.model_file")))
		content.WriteString("\n\n")
		pathInputLabel := i18n.T("tui.upload.path_label", nil)
		if m.act.pathInputFocused {
			pathInputLabel = m.titleStyle.Render("► " + pathInputLabel + i18n.T("tui.upload.focused", nil))
		} else {
			pathInputLabel = m.hintStyle.Render(pathInputLabel)
		}
		content.WriteString(pathInputLabel)
		content.WriteString("\n")
		if m.act.pathInputFocused {
			content.WriteString(m.renderPathInputWithCompletion())
			content.WriteString("\n") // 添加额外换行
		} else {
			content.WriteString(m.inpPath.View())
			content.WriteString("\n\n")
		}

		filePickerLabel := i18n.T("tui.upload.picker_label", nil)
		if !m.act.pathInputFocused {
			filePickerLabel = m.titleStyle.Render("► " + filePickerLabel + i18n.T("tui.upload.focused", nil))
		} else {
			filePickerLabel = m.hintStyle.Render(filePickerLabel)
		}
		content.WriteString(filePickerLabel)
		content.WriteString("\n")
		if m.act.filePickerErr != nil {
			content.WriteString(m.filepicker.Styles.DisabledFile.Render(m.act.filePickerErr.Error()))
		} else if m.selectedFile == "" {
			content.WriteString(i18n.T("tui.upload.choose_file", nil))
		} else {
			content.WriteString(i18n.T("tui.upload.selected_file", nil))
			content.WriteString(m.filepicker.Styles.Selected.Render(m.selectedFile))
		}
		content.WriteString("\n")
		content.WriteString(m.filepicker.View())
		content.WriteString("\n")
		if m.act.pathInputFocused {
			content.WriteString(m.hintStyle.Render(i18n.T("tui.upload.model_path_hint", nil)))
		} else {
			content.WriteString(m.hintStyle.Render(i18n.T("tui.upload.model_picker_hint", nil)))
		}
		return content.String()
	case stepPublic:
		if _, ih := m.innerSize(); ih > 0 {
			h := ih - 12
			if h < 5 {
				h = 5
			}
			m.publicList.SetHeight(h)
		}
		return m.titleStyle.Render(uploadTitle(10, "tui.upload.section.public")) + "\n\n" + m.publicList.View() + "\n" + m.hintStyle.Render(i18n.T("tui.upload.confirm_choice", nil))
	case stepAskMore:
		var b strings.Builder
		b.WriteString(m.titleStyle.Render(uploadTitle(11, "tui.upload.section.more")))
		b.WriteString("\n\n")
		if len(m.act.versions) > 0 {
			b.WriteString(i18n.T("tui.upload.added_versions", nil) + "\n")
			for i, v := range m.act.versions {
				b.WriteString(i18n.T("tui.upload.version_summary", map[string]any{
					"Index": i + 1, "Version": dash(v.version), "Base": dash(v.base), "Cover": dash(v.cover), "Path": dash(v.path),
				}) + "\n")
			}
			b.WriteString("\n")
		}
		cur := m.act.cur
		b.WriteString(i18n.T("tui.upload.current_version", nil) + "\n")
		b.WriteString(i18n.T("tui.upload.current_version_summary", map[string]any{
			"Version": dash(cur.version), "Base": dash(cur.base), "Cover": dash(cur.cover), "Path": dash(cur.path),
		}) + "\n\n")
		if _, ih := m.innerSize(); ih > 0 {
			h := ih - 12
			if h < 5 {
				h = 5
			}
			m.moreList.SetHeight(h)
		}
		return b.String() + "\n" + m.moreList.View() + "\n" + m.hintStyle.Render(i18n.T("tui.upload.confirm_choice", nil))
	case stepConfirm:
		var b strings.Builder
		b.WriteString(m.titleStyle.Render(i18n.T("tui.upload.section.confirm", nil)))
		b.WriteString("\n\n")
		b.WriteString(i18n.T("tui.upload.confirm_summary", map[string]any{"Name": m.act.u.name, "Type": m.act.u.typ}) + "\n\n")
		for i, v := range m.act.versions {
			b.WriteString(i18n.T("tui.upload.confirm_version", map[string]any{"Index": i + 1, "Version": dash(v.version), "Base": dash(v.base)}) + "\n")
			b.WriteString(i18n.T("tui.upload.confirm_details", map[string]any{
				"Cover": dash(v.cover), "Path": dash(v.path), "Intro": dash(truncateToLines(v.intro, 2)),
			}) + "\n\n")
		}
		b.WriteString(m.hintStyle.Render(i18n.T("tui.upload.confirm_hint", nil)))
		return b.String()
	}
	return ""
}
