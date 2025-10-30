package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		m.act.pathCompletionSuggestion = buildCompletionSuggestion(typedPath, matches)
		m.act.pathMatchCount = len(matches)

		// 更新 filepicker 过滤（提取最后的文件名部分）
		m.filepicker.FilterPrefix = prefix

		// 如果目录改变，重新读取
		if dir != m.filepicker.CurrentDirectory {
			m.filepicker.CurrentDirectory = dir
			return m.filepicker.Init()
		}
	} else {
		m.act.pathCompletionSuggestion = ""
		m.act.pathMatchCount = 0
		m.filepicker.FilterPrefix = ""
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
		m.filepicker.FilterPrefix = ""

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
		previewText := fmt.Sprintf("建议: %s", suggestion)
		result.WriteString(grayStyle.Render(previewText))

		// 显示匹配数量
		if m.act.pathMatchCount > 1 {
			countHint := fmt.Sprintf(" (%d 个匹配)", m.act.pathMatchCount)
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
	m.filepicker.FilterPrefix = ""
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
		return fmt.Errorf("不支持的文件格式，支持的格式：%s", strings.Join(supportedExts, ", "))
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
	summaryBuilder.WriteString(fmt.Sprintf("- type: %s\n- name: %s\n", dash(m.act.u.typ), dash(m.act.u.name)))
	for i, v := range m.act.versions {
		summaryBuilder.WriteString(fmt.Sprintf("  [%d] version=%s base=%s cover=%s path=%s intro=%s\n", i+1, dash(v.version), dash(v.base), dash(v.cover), dash(v.path), dash(truncateToLines(v.intro, 2))))
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
			progressSection.WriteString(fmt.Sprintf("当前: (%s) %s\n", m.uploadProg.fileIndex, m.uploadProg.fileName))
		} else {
			// 准备阶段，显示封面状态或默认提示
			if m.coverStatus != "" {
				if m.coverStatusWarning {
					// 警告样式（黄色）
					warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
					progressSection.WriteString(warningStyle.Render(m.coverStatus) + "\n")
				} else {
					progressSection.WriteString(m.coverStatus + "\n")
				}
			} else {
				progressSection.WriteString("准备上传…\n")
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
			progressSection.WriteString(fmt.Sprintf("%s[%d/%d] 版本=%s\n", prefix, i+1, len(m.verProgress), dash(versionLabel)))
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
				fileLine = "准备上传…"
			}
			progLine = m.progress.View()
			speedLine = ""
		}
		progressSection.WriteString(fileLine + "\n" + progLine)
		if speedLine != "" {
			progressSection.WriteString("\n" + speedLine)
		}
	}

	var hint string
	if m.canceling {
		hint = m.hintStyle.Render("正在取消上传，请稍候...（已上传部分会保存，支持断点续传）")
	} else {
		hint = m.hintStyle.Render("按 Ctrl+C 取消上传（已上传部分会保存，支持断点续传）")
	}

	return m.titleStyle.Render("上传中 · 请稍候") + "\n\n" + summary + "\n\n" + progressSection.String() + "\n\n" + hint
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
				return checkModelExists(m.apiKey, name, m.act.u.typ)
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
					method := it.title
					if strings.Contains(method, "URL") {
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
						m.filepicker.FilterPrefix = ""
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
				m.filepicker.FilterPrefix = ""
				m.upStep = stepCoverMethod
				return nil
			}
		}

		// 根据上传方式分别处理
		if m.act.coverUploadMethod == "url" {
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
						m.act.filePickerErr = errors.New("检测到分号，仅保留第一个 URL")
						warned = true
					}
					if _, err := os.Stat(first); err == nil {
						m.act.filePickerErr = fmt.Errorf("检测到本地路径，请返回上一步选择本地上传")
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					if !IsHTTPURL(first) {
						m.act.filePickerErr = fmt.Errorf("请输入封面的 URL（以 http/https 开头）")
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					check := first
					if q := strings.Index(check, "?"); q >= 0 {
						check = check[:q]
					}
					if !isSupportedCoverFormat(check) {
						m.act.filePickerErr = fmt.Errorf("URL 格式不支持: %s\n支持的格式: %s", first, getSupportedCoverFormats())
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
		} else if m.act.coverUploadMethod == "local" {
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
							m.act.filePickerErr = fmt.Errorf("路径不存在: %s", p)
							return clearFilePickerErrorAfter(3 * time.Second)
						}
						if info.IsDir() {
							m.filepicker.CurrentDirectory = p
							m.filepicker.FilterPrefix = ""
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
						m.filepicker.FilterPrefix = ""
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
						m.filepicker.FilterPrefix = ""
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
					m.filepicker.FilterPrefix = ""
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
						m.filepicker.FilterPrefix = ""
						m.upStep = stepIntroMethod
						return nil
					}
				}
				if didSelect, p := m.filepicker.DidSelectDisabledFile(msg); didSelect {
					if info, err := os.Stat(p); err == nil && !info.IsDir() {
						m.act.filePickerErr = errors.New(p + " 文件格式不支持")
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
					method := it.title
					if strings.Contains(method, "文件导入") {
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
						m.filepicker.FilterPrefix = ""
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
				if m.act.coverUploadMethod == "url" {
					return m.inpCover.Focus()
				} else if m.act.coverUploadMethod == "local" {
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
					m.filepicker.FilterPrefix = ""
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
							m.act.filePickerErr = fmt.Errorf("路径不存在: %s", p)
							return clearFilePickerErrorAfter(3 * time.Second)
						}
						if info.IsDir() {
							m.filepicker.CurrentDirectory = p
							m.filepicker.FilterPrefix = ""
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
							m.act.filePickerErr = fmt.Errorf("模型介绍（intro）是必填项，文件内容不能为空")
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
						m.filepicker.FilterPrefix = ""
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
					m.filepicker.FilterPrefix = ""
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
							m.act.filePickerErr = fmt.Errorf("模型介绍（intro）是必填项，文件内容不能为空")
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
						m.act.filePickerErr = errors.New(p + " 文件格式不支持")
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
						m.err = fmt.Errorf("模型介绍（intro）是必填项，请提供介绍文本或通过文件导入")
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
					m.filepicker.FilterPrefix = ""
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
				m.filepicker.FilterPrefix = ""
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
						m.act.filePickerErr = fmt.Errorf("路径无效或不是目录: %s", path)
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
						m.act.filePickerErr = fmt.Errorf("路径不存在: %s", path)
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					if info.IsDir() {
						m.filepicker.CurrentDirectory = path
						m.filepicker.FilterPrefix = ""
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
					m.filepicker.FilterPrefix = ""
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
				m.filepicker.FilterPrefix = ""
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
					m.act.filePickerErr = errors.New(path + " 文件格式不支持")
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
					title := it.title
					// 默认第一个是"否，保持私有"，第二个是"是，公开模型"
					if strings.HasPrefix(title, "是") {
						m.act.cur.public = true
					} else {
						m.act.cur.public = false
					}
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
					title := it.title
					if strings.HasPrefix(title, "是") {
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
				return runUploadActionMulti(m.act.u, m.act.versions)
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
		return m.titleStyle.Render("上传 · Step 1/9 · 选择模型类型") + "\n\n" + m.typeList.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc")
	case stepName:
		return m.titleStyle.Render("上传 · Step 2/9 · 模型名称") + "\n\n" + m.inpName.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc")
	case stepVersion:
		return m.titleStyle.Render("上传 · Step 3/9 · 版本名称（默认 v1.0）") + "\n\n" + m.inpVersion.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc")
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
			return m.titleStyle.Render("上传 · Step 4/9 · Base Model（必选）") + "\n\n" + m.sp.View() + " 正在加载基础模型类型列表…\n" + m.hintStyle.Render("返回：Esc")
		}
		// 如果列表为空（加载失败），显示提示
		if len(m.baseModelTypes) == 0 {
			return m.titleStyle.Render("上传 · Step 4/9 · Base Model（必选）") + "\n\n" + m.baseList.View() + "\n" + m.hintStyle.Render("（使用本地列表）选择后 Enter，返回：Esc")
		}
		return m.titleStyle.Render("上传 · Step 4/9 · Base Model（必选）") + "\n\n" + m.baseList.View() + "\n" + m.hintStyle.Render("选择后 Enter，返回：Esc")
	case stepCoverMethod:
		if _, ih := m.innerSize(); ih > 0 {
			h := ih - 12
			if h < 5 {
				h = 5
			}
			m.coverMethodList.SetHeight(h)
		}
		return m.titleStyle.Render("上传 · Step 5/9 · 选择封面上传方式") + "\n\n" + m.coverMethodList.View() + "\n" + m.hintStyle.Render("选择后 Enter，返回：Esc")
	case stepCover:
		var content strings.Builder

		if m.act.coverUploadMethod == "url" {
			content.WriteString(m.titleStyle.Render("上传 · Step 6/9 · 输入封面URL"))
			content.WriteString("\n\n")
			content.WriteString("封面 URL（必填，仅 1 个图片或视频链接）：\n")
			content.WriteString(m.inpCover.View() + "\n\n")
			if m.act.filePickerErr != nil {
				content.WriteString(m.filepicker.Styles.DisabledFile.Render(m.act.filePickerErr.Error()) + "\n\n")
			}
			content.WriteString(m.hintStyle.Render("输入封面 URL（图片或视频，视频限 100MB），回车确认并进入下一步；Esc 返回选择上传方式"))
		} else if m.act.coverUploadMethod == "local" {
			content.WriteString(m.titleStyle.Render("上传 · Step 6/9 · 选择本地封面文件"))
			content.WriteString("\n\n")
			pathLabel := "本地文件路径输入："
			if m.coverPathInputFocused {
				pathLabel = m.titleStyle.Render("► " + pathLabel + "（当前焦点，按 Ctrl+P 切换）")
			} else {
				pathLabel = m.hintStyle.Render(pathLabel)
			}
			content.WriteString(pathLabel + "\n")
			if m.coverPathInputFocused {
				content.WriteString(m.renderPathInputWithCompletion())
				content.WriteString("\n") // 添加额外换行
			} else {
				content.WriteString(m.inpPath.View())
				content.WriteString("\n\n")
			}

			pickerLabel := "文件选择器："
			if !m.coverPathInputFocused {
				pickerLabel = m.titleStyle.Render("► 文件选择器：（当前焦点，按 Ctrl+P 切换）")
			} else {
				pickerLabel = m.hintStyle.Render(pickerLabel)
			}
			content.WriteString(pickerLabel + "\n")
			if m.act.filePickerErr != nil {
				content.WriteString(m.filepicker.Styles.DisabledFile.Render(m.act.filePickerErr.Error()) + "\n")
			}
			content.WriteString(m.filepicker.View() + "\n")

			if m.coverPathInputFocused {
				content.WriteString(m.hintStyle.Render("输入本地文件路径（视频限 100MB），Enter 确认；Ctrl+P 切换焦点；Esc 返回选择上传方式"))
			} else {
				content.WriteString(m.hintStyle.Render("方向键导航，Enter 选择封面文件（视频限 100MB）；Ctrl+P 切换焦点；Esc 返回选择上传方式"))
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
		return m.titleStyle.Render("上传 · Step 7/10 · 选择介绍输入方式") + "\n\n" + m.introMethodList.View() + "\n" + m.hintStyle.Render("选择后 Enter，返回：Esc")
	case stepIntro:
		if m.act.introInputMethod == "file" {
			// 文件导入模式渲染
			var content strings.Builder
			content.WriteString(m.titleStyle.Render("上传 · Step 8/10 · 从文件导入介绍内容"))
			content.WriteString("\n\n")
			pathLabel := "本地文件路径输入："
			if m.act.introPathInputFocused {
				pathLabel = m.titleStyle.Render("► " + pathLabel + "（当前焦点，按 Ctrl+P 切换）")
			} else {
				pathLabel = m.hintStyle.Render(pathLabel)
			}
			content.WriteString(pathLabel + "\n")
			if m.act.introPathInputFocused {
				content.WriteString(m.renderPathInputWithCompletion())
				content.WriteString("\n") // 添加额外换行
			} else {
				content.WriteString(m.inpPath.View())
				content.WriteString("\n\n")
			}

			pickerLabel := "文件选择器："
			if !m.act.introPathInputFocused {
				pickerLabel = m.titleStyle.Render("► 文件选择器：（当前焦点，按 Ctrl+P 切换）")
			} else {
				pickerLabel = m.hintStyle.Render(pickerLabel)
			}
			content.WriteString(pickerLabel + "\n")
			if m.act.filePickerErr != nil {
				content.WriteString(m.filepicker.Styles.DisabledFile.Render(m.act.filePickerErr.Error()) + "\n")
			}
			content.WriteString(m.filepicker.View() + "\n")

			if m.act.introPathInputFocused {
				content.WriteString(m.hintStyle.Render("输入 .txt 或 .md 文件路径，Enter 确认；Ctrl+P 切换焦点；Esc 返回选择输入方式"))
			} else {
				content.WriteString(m.hintStyle.Render("方向键导航，Enter 选择介绍文件（.txt 或 .md，自动截断到 5000 字）；Ctrl+P 切换焦点；Esc 返回选择输入方式"))
			}
			return content.String()
		} else {
			// 直接输入模式渲染
			charCount := len([]rune(m.taIntro.Value()))
			charInfo := fmt.Sprintf("（%d/5000 字）", charCount)
			return m.titleStyle.Render("上传 · Step 8/10 · 模型介绍") + " " + m.hintStyle.Render(charInfo) + "\n\n" + m.taIntro.View() + "\n" + m.hintStyle.Render("支持 Markdown 格式；提交：Ctrl+S，返回：Esc")
		}
	case stepPath:
		var content strings.Builder
		content.WriteString(m.titleStyle.Render("上传 · Step 9/11 · 选择文件") + "\n\n")
		pathInputLabel := "路径输入："
		if m.act.pathInputFocused {
			pathInputLabel = m.titleStyle.Render("► 路径输入：（当前焦点，按Ctrl+P切换至文件选择器）")
		} else {
			pathInputLabel = m.hintStyle.Render("路径输入：")
		}
		content.WriteString(pathInputLabel + "\n")
		if m.act.pathInputFocused {
			content.WriteString(m.renderPathInputWithCompletion())
			content.WriteString("\n") // 添加额外换行
		} else {
			content.WriteString(m.inpPath.View())
			content.WriteString("\n\n")
		}

		filePickerLabel := "文件选择器："
		if !m.act.pathInputFocused {
			filePickerLabel = m.titleStyle.Render("► 文件选择器：（当前焦点，按Ctrl+P切换至路径输入框）")
		} else {
			filePickerLabel = m.hintStyle.Render("文件选择器：")
		}
		content.WriteString(filePickerLabel + "\n")
		if m.act.filePickerErr != nil {
			content.WriteString(m.filepicker.Styles.DisabledFile.Render(m.act.filePickerErr.Error()))
		} else if m.selectedFile == "" {
			content.WriteString("选择一个文件:")
		} else {
			content.WriteString("已选择文件: " + m.filepicker.Styles.Selected.Render(m.selectedFile))
		}
		content.WriteString("\n" + m.filepicker.View() + "\n")
		if m.act.pathInputFocused {
			content.WriteString(m.hintStyle.Render("输入有效目录将自动同步下方文件列表；Enter确认文件或切换目录；Ctrl+P切换焦点；Esc返回"))
		} else {
			content.WriteString(m.hintStyle.Render("方向键导航，Enter选择文件，Ctrl+P切换输入（输入框实时同步），Esc返回"))
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
		return m.titleStyle.Render("上传 · Step 10/11 · 是否公开此版本？") + "\n\n" + m.publicList.View() + "\n" + m.hintStyle.Render("Enter 确认选择，Esc 返回上一页")
	case stepAskMore:
		var b strings.Builder
		b.WriteString(m.titleStyle.Render("上传 · Step 11/11 · 是否继续添加版本？") + "\n\n")
		if len(m.act.versions) > 0 {
			b.WriteString("已添加版本：\n")
			for i, v := range m.act.versions {
				b.WriteString(fmt.Sprintf("  - [%d] %s  base=%s  cover=%s  path=%s\n", i+1, dash(v.version), dash(v.base), dash(v.cover), dash(v.path)))
			}
			b.WriteString("\n")
		}
		cur := m.act.cur
		b.WriteString("当前版本：\n")
		b.WriteString(fmt.Sprintf("  - %s  base=%s  cover=%s  path=%s\n\n", dash(cur.version), dash(cur.base), dash(cur.cover), dash(cur.path)))
		if _, ih := m.innerSize(); ih > 0 {
			h := ih - 12
			if h < 5 {
				h = 5
			}
			m.moreList.SetHeight(h)
		}
		return b.String() + "\n" + m.moreList.View() + "\n" + m.hintStyle.Render("Enter 确认选择，Esc 返回上一页")
	case stepConfirm:
		var b strings.Builder
		b.WriteString(m.titleStyle.Render("上传 · 确认所有版本") + "\n\n")
		b.WriteString(fmt.Sprintf("模型名称：%s\n类型：%s\n\n", m.act.u.name, m.act.u.typ))
		for i, v := range m.act.versions {
			b.WriteString(fmt.Sprintf("[%d] 版本=%s  base=%s\n", i+1, dash(v.version), dash(v.base)))
			b.WriteString(fmt.Sprintf("cover=%s\npath=%s\nintro=%s\n\n", dash(v.cover), dash(v.path), dash(truncateToLines(v.intro, 2))))
		}
		b.WriteString(m.hintStyle.Render("按 Enter 开始上传；Esc 返回上一步"))
		return b.String()
	}
	return ""
}
