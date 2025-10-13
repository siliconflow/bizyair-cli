package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	m.act = actionInputs{}
	m.upStep = stepType

	m.inpName.SetValue("")
	m.inpVersion.SetValue("")
	m.inpCover.SetValue("")
	m.inpIntro.SetValue("")
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
}

// 路径校验与设置
func (m *mainModel) validateAndSetPath(path string) error {
	supportedExts := []string{".safetensors"}
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
		summaryBuilder.WriteString(fmt.Sprintf("  [%d] version=%s base=%s cover=%s path=%s intro=%s\n", i+1, dash(v.version), dash(v.base), dash(v.cover), dash(v.path), dash(v.intro)))
	}
	summary := summaryBuilder.String()

	var progressSection strings.Builder
	if len(m.verProgress) > 0 {
		if m.uploadProg.total > 0 {
			progressSection.WriteString(fmt.Sprintf("当前: (%s) %s\n", m.uploadProg.fileIndex, m.uploadProg.fileName))
		} else {
			progressSection.WriteString("准备上传…\n")
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
				progressSection.WriteString(fmt.Sprintf("%s%s %.1f%% (%s/%s)\n", prefix, bar, percent*100, format.FormatBytes(consumed), format.FormatBytes(total)))
			} else {
				progressSection.WriteString(fmt.Sprintf("%s%s\n", prefix, bar))
			}
		}
	} else {
		var fileLine string
		var progLine string
		if m.uploadProg.total > 0 {
			percent := float64(m.uploadProg.consumed) / float64(m.uploadProg.total)
			fileLine = fmt.Sprintf("(%s) %s", m.uploadProg.fileIndex, m.uploadProg.fileName)
			progLine = fmt.Sprintf("%s %.1f%% (%s/%s)", m.progress.View(), percent*100, format.FormatBytes(m.uploadProg.consumed), format.FormatBytes(m.uploadProg.total))
		} else {
			fileLine = "准备上传…"
			progLine = m.progress.View()
		}
		progressSection.WriteString(fileLine + "\n" + progLine)
	}
	return m.titleStyle.Render("上传中 · 请稍候") + "\n\n" + summary + "\n\n" + progressSection.String()
}

// 根据当前动作处理输入与触发命令（仅上传和列表两个分支）
func (m *mainModel) updateActionInputs(msg tea.Msg) tea.Cmd {
	switch m.currentAction {
	case actionUpload:
		return m.updateUploadInputs(msg)
	case actionLsModel:
		return m.updateListModelsInputs(msg)
	default:
		return nil
	}
}

// 渲染动作视图
func (m *mainModel) renderActionView() string {
	switch m.currentAction {
	case actionUpload:
		return m.renderUploadStepsView()
	case actionLsModel:
		return m.renderListModelsView()
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
				m.upStep = stepName
				return m.inpName.Focus()
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
					m.upStep = stepCover
					m.act.useFilePicker = true
					m.coverUrlInputFocused = false
					m.coverPathInputFocused = true
					m.inpCover.SetValue("")
					m.inpPath.SetValue(ensureTrailingSep(m.filepicker.CurrentDirectory))
					m.filepicker.AllowedTypes = nil
					m.filepicker.DirAllowed = true
					m.filepicker.FileAllowed = true
					m.filepicker.Path = ""
					return tea.Batch(m.inpPath.Focus(), m.filepicker.Init())
				}
			case "esc":
				m.upStep = stepVersion
				return nil
			}
		}
		return cmd
	case stepCover:
		var urlCmd, pathCmd, fpCmd tea.Cmd
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "esc":
				m.coverUrlInputFocused = false
				m.coverPathInputFocused = false
				m.act.filePickerErr = nil
				m.filepicker.Path = ""
				m.upStep = stepBase
				return nil
			case "tab":
				if m.coverUrlInputFocused {
					m.coverUrlInputFocused = false
					m.coverPathInputFocused = true
					m.inpCover.Blur()
					return m.inpPath.Focus()
				}
				if m.coverPathInputFocused {
					path := strings.TrimSpace(m.inpPath.Value())
					if path != "" {
						if info, err := os.Stat(path); err == nil && info.IsDir() {
							m.filepicker.CurrentDirectory = path
							m.coverPathInputFocused = false
							m.inpPath.Blur()
							return m.filepicker.Init()
						}
					}
					m.coverPathInputFocused = false
					m.inpPath.Blur()
					return nil
				}
				m.coverUrlInputFocused = true
				return m.inpCover.Focus()
			case "enter":
				if m.coverUrlInputFocused {
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
						m.act.filePickerErr = fmt.Errorf("检测到本地路径，请在下方文件选择器中选择：%s", first)
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					if !IsHTTPURL(first) {
						m.act.filePickerErr = fmt.Errorf("请输入图片的 URL（以 http/https 开头）")
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					check := first
					if q := strings.Index(check, "?"); q >= 0 {
						check = check[:q]
					}
					lcheck := strings.ToLower(check)
					if !(strings.HasSuffix(lcheck, ".jpg") || strings.HasSuffix(lcheck, ".jpeg") || strings.HasSuffix(lcheck, ".png") || strings.HasSuffix(lcheck, ".gif") || strings.HasSuffix(lcheck, ".bmp") || strings.HasSuffix(lcheck, ".webp")) {
						m.act.filePickerErr = fmt.Errorf("URL 不是图片链接: %s", first)
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					m.act.cur.cover = first
					m.upStep = stepIntro
					if warned {
						return tea.Batch(m.inpIntro.Focus(), clearFilePickerErrorAfter(3*time.Second))
					}
					return m.inpIntro.Focus()
				}
				if m.coverPathInputFocused && m.inpPath.Value() != "" {
					p := strings.TrimSpace(m.inpPath.Value())
					info, err := os.Stat(p)
					if err != nil {
						m.act.filePickerErr = fmt.Errorf("路径不存在: %s", p)
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					if info.IsDir() {
						m.filepicker.CurrentDirectory = p
						return m.filepicker.Init()
					}
					lp := strings.ToLower(p)
					if !(strings.HasSuffix(lp, ".jpg") || strings.HasSuffix(lp, ".jpeg") || strings.HasSuffix(lp, ".png") || strings.HasSuffix(lp, ".gif") || strings.HasSuffix(lp, ".bmp") || strings.HasSuffix(lp, ".webp")) {
						m.act.filePickerErr = fmt.Errorf("不支持的图片类型: %s", p)
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					ap := absPath(p)
					m.inpCover.SetValue(ap)
					m.act.cur.cover = ap
					m.upStep = stepIntro
					return m.inpIntro.Focus()
				}
			}
		}
		if m.coverUrlInputFocused {
			m.inpCover, urlCmd = m.inpCover.Update(msg)
		}
		if m.coverPathInputFocused {
			m.inpPath, pathCmd = m.inpPath.Update(msg)
			typedPath := strings.TrimSpace(m.inpPath.Value())
			if typedPath != "" {
				if info, err := os.Stat(typedPath); err == nil && info.IsDir() {
					if filepath.Clean(m.filepicker.CurrentDirectory) != filepath.Clean(typedPath) {
						m.filepicker.CurrentDirectory = typedPath
						fpCmd = m.filepicker.Init()
					}
				}
			}
		}
		if !m.coverUrlInputFocused && !m.coverPathInputFocused {
			oldDir := m.filepicker.CurrentDirectory
			var fc tea.Cmd
			m.filepicker, fc = m.filepicker.Update(msg)
			if fc != nil {
				fpCmd = fc
			}
			if m.filepicker.CurrentDirectory != oldDir {
				m.inpPath.SetValue(ensureTrailingSep(m.filepicker.CurrentDirectory))
			}
			if did, p := m.filepicker.DidSelectFile(msg); did {
				if info, err := os.Stat(p); err == nil && !info.IsDir() {
					lp := strings.ToLower(p)
					if !(strings.HasSuffix(lp, ".jpg") || strings.HasSuffix(lp, ".jpeg") || strings.HasSuffix(lp, ".png") || strings.HasSuffix(lp, ".gif") || strings.HasSuffix(lp, ".bmp") || strings.HasSuffix(lp, ".webp")) {
						m.act.filePickerErr = fmt.Errorf("不支持的图片类型: %s", p)
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					ap := absPath(p)
					m.inpCover.SetValue(ap)
					m.act.cur.cover = ap
					m.upStep = stepIntro
					return m.inpIntro.Focus()
				}
			}
			if didSelect, p := m.filepicker.DidSelectDisabledFile(msg); didSelect {
				if info, err := os.Stat(p); err == nil && !info.IsDir() {
					m.act.filePickerErr = errors.New(p + " 文件格式不支持")
					return clearFilePickerErrorAfter(3 * time.Second)
				}
			}
		}
		return tea.Batch(urlCmd, pathCmd, fpCmd)
	case stepIntro:
		var cmd tea.Cmd
		m.inpIntro, cmd = m.inpIntro.Update(msg)
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "enter":
				m.act.cur.intro = strings.TrimSpace(m.inpIntro.Value())
				m.act.useFilePicker = true
				m.act.pathInputFocused = false
				m.inpPath.SetValue(ensureTrailingSep(m.filepicker.CurrentDirectory))
				m.upStep = stepPath
				return m.filepicker.Init()
			case "esc":
				m.upStep = stepCover
				return m.inpCover.Focus()
			}
		}
		return cmd
	case stepPath:
		var pathCmd, fpCmd tea.Cmd
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "esc":
				m.act.useFilePicker = false
				m.act.pathInputFocused = false
				m.act.filePickerErr = nil
				m.filepicker.Path = ""
				m.upStep = stepIntro
				return m.inpIntro.Focus()
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
				m.upStep = stepAskMore
				return nil
			case "tab":
				if m.act.pathInputFocused {
					path := strings.TrimSpace(m.inpPath.Value())
					if path != "" {
						if info, err := os.Stat(path); err == nil && info.IsDir() {
							m.filepicker.CurrentDirectory = path
							m.act.pathInputFocused = false
							m.inpPath.Blur()
							return m.filepicker.Init()
						}
					}
					m.act.pathInputFocused = false
					m.inpPath.Blur()
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
						return m.filepicker.Init()
					}
					if err := m.validateAndSetPath(path); err != nil {
						m.act.filePickerErr = err
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					return nil
				}
			}
		}
		if m.act.pathInputFocused {
			m.inpPath, pathCmd = m.inpPath.Update(msg)
			typedPath := strings.TrimSpace(m.inpPath.Value())
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
			oldDir := m.filepicker.CurrentDirectory
			var fc tea.Cmd
			m.filepicker, fc = m.filepicker.Update(msg)
			if fc != nil {
				fpCmd = fc
			}
			if m.filepicker.CurrentDirectory != oldDir {
				m.inpPath.SetValue(ensureTrailingSep(m.filepicker.CurrentDirectory))
			}
			if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
				if info, err := os.Stat(path); err == nil && !info.IsDir() {
					if err := m.validateAndSetPath(path); err != nil {
						m.act.filePickerErr = err
						return clearFilePickerErrorAfter(3 * time.Second)
					}
					m.upStep = stepAskMore
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
						m.inpIntro.SetValue("")
						m.upStep = stepVersion
						return nil
					}
					m.act.versions = append(m.act.versions, m.act.cur)
					m.upStep = stepConfirm
					m.act.confirming = true
					return nil
				}
			case "esc":
				m.upStep = stepPath
				m.act.useFilePicker = true
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
			h := ih - 10
			if h < 5 {
				h = 5
			}
			m.typeList.SetHeight(h)
		}
		return m.titleStyle.Render("上传 · Step 1/8 · 选择模型类型") + "\n\n" + m.typeList.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
	case stepName:
		return m.titleStyle.Render("上传 · Step 2/8 · 模型名称") + "\n\n" + m.inpName.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
	case stepVersion:
		return m.titleStyle.Render("上传 · Step 3/8 · 版本名称（默认 v1.0）") + "\n\n" + m.inpVersion.View() + "\n" + m.hintStyle.Render("确认：Enter，返回：Esc，退出：q")
	case stepBase:
		if _, ih := m.innerSize(); ih > 0 {
			h := ih - 10
			if h < 5 {
				h = 5
			}
			m.baseList.SetHeight(h)
		}
		return m.titleStyle.Render("上传 · Step 4/8 · Base Model（必选）") + "\n\n" + m.baseList.View() + "\n" + m.hintStyle.Render("选择后 Enter，返回：Esc，退出：q")
	case stepCover:
		var content strings.Builder
		content.WriteString(m.titleStyle.Render("上传 · Step 5/8 · 选择封面（默认文件选择器，Tab 在 URL/路径/文件选择器之间切换）"))
		content.WriteString("\n\n")
		urlLabel := "封面 URL（仅 1 个图片链接；本地图片请在下方选择）："
		if m.coverUrlInputFocused {
			urlLabel = m.titleStyle.Render("► " + urlLabel + "（当前焦点，按 Tab 切换）")
		} else {
			urlLabel = m.hintStyle.Render(urlLabel)
		}
		content.WriteString(urlLabel + "\n" + m.inpCover.View() + "\n\n")
		pathLabel := "本地文件路径输入（按 Enter 确认并进入下一步）："
		if m.coverPathInputFocused {
			pathLabel = m.titleStyle.Render("► " + pathLabel + "（当前焦点，按 Tab 切换）")
		} else {
			pathLabel = m.hintStyle.Render(pathLabel)
		}
		content.WriteString(pathLabel + "\n" + m.inpPath.View() + "\n\n")
		pickerLabel := "文件选择器："
		if !m.coverUrlInputFocused && !m.coverPathInputFocused {
			pickerLabel = m.titleStyle.Render("► 文件选择器：（当前焦点，按 Tab 切换）")
		} else {
			pickerLabel = m.hintStyle.Render(pickerLabel)
		}
		content.WriteString(pickerLabel + "\n")
		if m.act.filePickerErr != nil {
			content.WriteString(m.filepicker.Styles.DisabledFile.Render(m.act.filePickerErr.Error()) + "\n")
		}
		content.WriteString(m.filepicker.View() + "\n")
		if m.coverUrlInputFocused {
			content.WriteString(m.hintStyle.Render("输入图片 URL，回车确认并进入下一步；Tab 切换焦点；Esc 返回"))
		} else if m.coverPathInputFocused {
			content.WriteString(m.hintStyle.Render("输入本地文件路径，按 Enter 确认并进入下一步；Tab 切换焦点；Esc 返回"))
		} else {
			content.WriteString(m.hintStyle.Render("方向键导航，Enter 选择图片并进入下一步；Tab 切换焦点；Esc 返回"))
		}
		return content.String()
	case stepIntro:
		return m.titleStyle.Render("上传 · Step 6/8 · 模型介绍（可为空，回车跳过）") + "\n\n" + m.inpIntro.View() + "\n" + m.hintStyle.Render("确认：Enter（可空），返回：Esc，退出：q")
	case stepPath:
		var content strings.Builder
		content.WriteString(m.titleStyle.Render("上传 · Step 7/8 · 选择文件") + "\n\n")
		pathInputLabel := "路径输入："
		if m.act.pathInputFocused {
			pathInputLabel = m.titleStyle.Render("► 路径输入：（当前焦点，按Tab切换至文件选择器）")
		} else {
			pathInputLabel = m.hintStyle.Render("路径输入：")
		}
		content.WriteString(pathInputLabel + "\n" + m.inpPath.View() + "\n\n")
		filePickerLabel := "文件选择器："
		if !m.act.pathInputFocused {
			filePickerLabel = m.titleStyle.Render("► 文件选择器：（当前焦点，按Tab切换至路径输入框）")
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
			content.WriteString(m.hintStyle.Render("输入有效目录将自动同步下方文件列表；Enter确认文件或切换目录；Tab切换焦点；Esc返回"))
		} else {
			content.WriteString(m.hintStyle.Render("方向键导航，Enter选择文件，Tab切换输入（输入框实时同步），Esc返回"))
		}
		return content.String()
	case stepAskMore:
		var b strings.Builder
		b.WriteString(m.titleStyle.Render("上传 · Step 8/8 · 是否继续添加版本？") + "\n\n")
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
			b.WriteString(fmt.Sprintf("cover=%s\npath=%s\nintro=%s\n\n", dash(v.cover), dash(v.path), dash(v.intro)))
		}
		b.WriteString(m.hintStyle.Render("按 Enter 开始上传；Esc 返回上一步；q 退出"))
		return b.String()
	}
	return ""
}
