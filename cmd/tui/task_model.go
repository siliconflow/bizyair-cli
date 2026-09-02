package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
)

// initCurrentParamInput 按参数类型初始化输入控件：
// 枚举选项或布尔值 → 选择列表；图片/视频 → URL 输入框；其余 → 多行文本域。
func (m *mainModel) initCurrentParamInput() tea.Cmd {
	if m.taskModel.detail == nil || m.currentParamIdx() >= len(m.taskModel.detail.InputParams) {
		return nil
	}
	p := m.taskModel.detail.InputParams[m.currentParamIdx()]
	m.taskModel.paramInputs = make(map[string]textinput.Model)
	m.taskModel.usingTA = false
	m.taskModel.usingSelect = false

	switch {
	case p.FieldOptions != nil && len(p.FieldOptions.EnumValues()) > 0:
		// 有限枚举选项优先，image/video/boolean 均适用
		m.initParamSelectList(p, p.FieldOptions.EnumValues())
	case p.VariableType == "boolean":
		m.initParamSelectList(p, []any{true, false})
	case p.VariableType == "image" || p.VariableType == "video":
		ti := textinput.New()
		ti.Placeholder = i18n.T("tui.task_model.url_placeholder", map[string]any{"Type": variableTypeDisplayName(p.VariableType)})
		if initial, ok := m.paramInitialValue(p); ok {
			ti.SetValue(initial)
		}
		iw, _ := m.innerSize()
		if w := iw - 10; w > 10 {
			ti.Width = w
		}
		ti.Focus()
		m.taskModel.paramInputs[p.ParamKey()] = ti
	default:
		ta := textarea.New()
		ta.Placeholder = i18n.T("tui.task_model.text_placeholder", map[string]any{"Label": i18n.APITranslate("param_label", p.FieldLabel)})
		if initial, ok := m.paramInitialValue(p); ok {
			ta.SetValue(initial)
		}
		if _, ih := m.innerSize(); ih > 6 {
			ta.SetHeight(ih - 18)
			if ta.Height() < 3 {
				ta.SetHeight(3)
			}
		} else {
			ta.SetHeight(3)
		}
		ta.ShowLineNumbers = false
		ta.Focus()
		m.taskModel.taParam = ta
		m.taskModel.usingTA = true
	}
	m.taskModel.paramRules = lib.DeriveModelzooRules(p)
	m.taskModel.paramError = nil
	return nil
}

// currentParamIdx 返回当前参数的索引。
func (m *mainModel) currentParamIdx() int {
	return m.taskModel.currentParamIdx
}

// currentParam 返回当前正在填写的参数定义（越界时返回 nil）。
func (m *mainModel) currentParam() *lib.ModelzooInputParam {
	if m.taskModel.detail == nil || m.taskModel.currentParamIdx >= len(m.taskModel.detail.InputParams) {
		return nil
	}
	return &m.taskModel.detail.InputParams[m.taskModel.currentParamIdx]
}

// paramInitialValue 返回参数输入框初始值：优先取已填值，其次 API 默认值。
func (m *mainModel) paramInitialValue(p lib.ModelzooInputParam) (string, bool) {
	if v, ok := m.taskModel.params[p.ParamKey()]; ok && v != nil {
		return fieldValueDisplay(v), true
	}
	if p.FieldValue != nil {
		return fieldValueDisplay(p.FieldValue), true
	}
	return "", false
}

// fieldValueDisplay 将参数默认值规范化为可编辑的纯文本。
func fieldValueDisplay(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []string:
		return strings.Join(val, ", ")
	case []any:
		parts := make([]string, 0, len(val))
		for _, e := range val {
			parts = append(parts, fieldValueDisplay(e))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// variableTypeDisplayName 返回变量类型的显示名称。
func variableTypeDisplayName(t string) string {
	switch t {
	case "image":
		return i18n.T("tui.task_model.type_image", nil)
	case "video":
		return i18n.T("tui.task_model.type_video", nil)
	default:
		return t
	}
}

// initParamSelectList 为参数构建枚举选择列表，并预选已填值/API 默认值。
func (m *mainModel) initParamSelectList(p lib.ModelzooInputParam, values []any) {
	m.taskModel.usingSelect = true
	items := make([]list.Item, 0, len(values))
	for _, v := range values {
		items = append(items, listItem{title: fmt.Sprintf("%v", v), value: fmt.Sprintf("%v", v)})
	}
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	cSel := lipgloss.Color("#C4B5FD")
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(cSel).BorderLeftForeground(cSel)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(cSel)
	l := list.New(items, d, 0, 0)
	l.Title = i18n.APITranslate("param_label", p.FieldLabel)
	localizeList(&l)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	// 默认选中首项，保证未按键时 Enter 也能拿到有效选项
	l.Select(0)
	// 预选：优先已填参数，其次 API 默认值
	if defaultVal, ok := m.paramInitialValue(p); ok && defaultVal != "" {
		for idx, item := range items {
			if li, ok := item.(listItem); ok && li.value == defaultVal {
				l.Select(idx)
				break
			}
		}
	}
	m.taskModel.paramSelectList = l
	m.syncListSizes()
}

// paramInputHeight 返回参数选择列表/多行文本域在面板内可用的高度，
// 保证总内容不超过边框内容区域，避免底部边框线条被截断。
func (m *mainModel) paramInputHeight() int {
	_, ih := m.innerSize()
	// 面板开销：renderFrame 中 header(1) + panel 边框与上下内边距(4)
	// 列表上方固定内容：页标题(2) + 参数标题/进度/标签/提示/tooltip/空行(约11)
	// 列表下方：空行(1) + 错误(2) + 已填汇总(2 + N)
	h := ih - 5 - 13 - 5 - len(m.taskModel.params)
	if h < 4 {
		h = 4
	}
	return h
}

// goBackParam 返回上一步参数：非首个参数时回到上一个参数输入
// （已填值会回显）；首个参数或从输出命名步骤回到最后一个参数，
// 无参数可回退时退出到模型广场。
func (m *mainModel) goBackParam() tea.Cmd {
	if m.taskModel.detail == nil {
		return nil
	}
	if m.taskModel.currentParamIdx <= 0 {
		m.step = mainStepModelzoo
		return nil
	}
	m.taskModel.currentParamIdx--
	m.taskModel.step = taskModelParamPoll
	m.taskModel.paramError = nil
	return m.initCurrentParamInput()
}

// saveCurrentParamValue 校验并按类型转换后保存当前参数值；校验失败返回错误。
func (m *mainModel) saveCurrentParamValue() error {
	p := m.currentParam()
	if p == nil {
		return nil
	}
	rules := m.taskModel.paramRules
	if len(rules) == 0 {
		rules = lib.DeriveModelzooRules(*p)
	}
	var raw string
	if m.taskModel.usingSelect {
		if it, ok := m.taskModel.paramSelectList.SelectedItem().(listItem); ok {
			raw = it.value
		}
	} else if m.taskModel.usingTA {
		raw = strings.TrimSpace(m.taskModel.taParam.Value())
	} else if ti, ok := m.taskModel.paramInputs[p.ParamKey()]; ok {
		raw = strings.TrimSpace(ti.Value())
	}
	if err := lib.ValidateModelzooParam(rules, raw); err != nil {
		return err
	}
	if raw == "" {
		return nil
	}
	v, err := lib.CoerceModelzooParamValue(p.VariableType, raw)
	if err != nil {
		return err
	}
	m.taskModel.params[p.ParamKey()] = v
	return nil
}

// validateTaskParams 校验所有必填参数均已填写（有 API 默认值的视为已填）。
func (m *mainModel) validateTaskParams() error {
	if m.taskModel.detail == nil {
		return nil
	}
	missing := lib.MissingModelzooRequiredParams(m.taskModel.detail.InputParams, m.taskModel.params)
	if len(missing) > 0 {
		return fmt.Errorf("%s", i18n.T("tui.task_model.param_required", map[string]any{"Name": i18n.APITranslate("param_label", missing[0].FieldLabel)}))
	}
	return nil
}

// applyFieldDefaults 为未填写的参数补入 API 默认值。
func (m *mainModel) applyFieldDefaults() {
	if m.taskModel.detail == nil {
		return
	}
	lib.ApplyModelzooFieldDefaults(m.taskModel.params, m.taskModel.detail.InputParams)
}

// advanceParam 前进到下一个参数；全部填完后校验必填项、补全默认值并进入输出命名步骤。
func (m *mainModel) advanceParam() tea.Cmd {
	m.taskModel.currentParamIdx++
	if m.taskModel.detail == nil || m.taskModel.currentParamIdx >= len(m.taskModel.detail.InputParams) {
		if err := m.validateTaskParams(); err != nil {
			m.err = err
			return nil
		}
		m.applyFieldDefaults()
		m.taskModel.step = taskModelOutputName
		nameInput := textinput.New()
		nameInput.Prompt = i18n.T("tui.task_model.output_name_prompt", nil) + " "
		nameInput.Placeholder = i18n.T("tui.task_model.output_name_placeholder", nil)
		innerW, _ := m.innerSize()
		if innerW-12 > 0 {
			nameInput.Width = innerW - 12
		}
		nameInput.Focus()
		m.taskModel.outputNameInput = nameInput
		return nil
	}
	return m.initCurrentParamInput()
}

// updateTaskModel 按当前子步骤分发消息。
func updateTaskModel(m *mainModel, msg tea.Msg) (mainModel, tea.Cmd) {
	switch m.taskModel.step {
	case taskModelSelect:
		// 当前流程不经过此步骤，统一从 taskModelParamPoll 开始
		return *m, nil
	case taskModelParamPoll:
		return *m, m.updateParamPoll(msg)
	case taskModelOutputName:
		return *m, m.updateOutputName(msg)
	case taskModelPolling:
		return *m, m.updateTaskPolling(msg)
	case taskModelResult:
		return *m, m.updateTaskResult(msg)
	}
	return *m, nil
}

// updateParamPoll 处理参数填写页按键与输入更新：
// Ctrl+U 清空当前输入、Ctrl+R 重置全部已填参数、Ctrl+S/Enter 保存并前进、Esc 返回上一步。
func (m *mainModel) updateParamPoll(msg tea.Msg) tea.Cmd {
	if m.taskModel.usingSelect {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "esc":
				return m.goBackParam()
			case "ctrl+u":
				m.taskModel.paramSelectList.Select(0)
				return nil
			case "ctrl+r":
				m.taskModel.params = make(map[string]any)
				m.taskModel.currentParamIdx = 0
				return m.initCurrentParamInput()
			case "enter":
				p := m.currentParam()
				if p != nil && p.Required {
					if it, ok := m.taskModel.paramSelectList.SelectedItem().(listItem); !ok || it.value == "" {
						m.err = fmt.Errorf("%s", i18n.T("tui.task_model.param_required", map[string]any{"Name": i18n.APITranslate("param_label", p.FieldLabel)}))
						return nil
					}
				}
				if err := m.saveCurrentParamValue(); err != nil {
					m.err = err
					return nil
				}
				return m.advanceParam()
			}
		}
		var cmd tea.Cmd
		m.taskModel.paramSelectList, cmd = m.taskModel.paramSelectList.Update(msg)
		return cmd
	}

	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			return m.goBackParam()
		case "ctrl+u":
			if m.taskModel.usingTA {
				m.taskModel.taParam.SetValue("")
			} else if p := m.currentParam(); p != nil {
				if ti, ok := m.taskModel.paramInputs[p.ParamKey()]; ok {
					ti.SetValue("")
					m.taskModel.paramInputs[p.ParamKey()] = ti
				}
			}
			return nil
		case "ctrl+r":
			m.taskModel.params = make(map[string]any)
			m.taskModel.currentParamIdx = 0
			return m.initCurrentParamInput()
		case "ctrl+s":
			if m.taskModel.usingTA {
				if err := m.saveCurrentParamValue(); err != nil {
					m.err = err
					return nil
				}
				return m.advanceParam()
			}
		case "enter":
			if !m.taskModel.usingTA {
				if err := m.saveCurrentParamValue(); err != nil {
					m.err = err
					return nil
				}
				return m.advanceParam()
			}
		}
	}

	if m.taskModel.usingTA {
		var cmd tea.Cmd
		m.taskModel.taParam, cmd = m.taskModel.taParam.Update(msg)
		m.taskModel.paramError = lib.ValidateModelzooParam(m.taskModel.paramRules, m.taskModel.taParam.Value())
		return cmd
	}
	p := m.currentParam()
	if p != nil {
		if ti, ok := m.taskModel.paramInputs[p.ParamKey()]; ok {
			var cmd tea.Cmd
			ti, cmd = ti.Update(msg)
			m.taskModel.paramInputs[p.ParamKey()] = ti
			m.taskModel.paramError = lib.ValidateModelzooParam(m.taskModel.paramRules, ti.Value())
			return cmd
		}
	}
	return nil
}

// updateOutputName 处理输出命名步骤按键：Enter 提交任务、Esc 返回、Ctrl+U 清空。
func (m *mainModel) updateOutputName(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "enter":
			name := strings.TrimSpace(m.taskModel.outputNameInput.Value())
			if name != "" {
				m.taskModel.params["output_name"] = name
			}
			m.taskModel.step = taskModelPolling
			m.running = true
			return createTaskCmd(m.getAPI(), m.taskModel.endpoint, m.taskModel.params)
		case "esc":
			return m.goBackParam()
		case "ctrl+u":
			m.taskModel.outputNameInput.SetValue("")
			return nil
		}
	}
	m.taskModel.outputNameInput, cmd = m.taskModel.outputNameInput.Update(msg)
	return cmd
}

// updateTaskPolling 处理轮询状态按键。
func (m *mainModel) updateTaskPolling(msg tea.Msg) tea.Cmd {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			m.step = mainStepModelzooDetail
			m.running = false
			return nil
		}
	}
	return nil
}

// updateTaskResult 处理结果页按键。
func (m *mainModel) updateTaskResult(msg tea.Msg) tea.Cmd {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "enter", "esc":
			m.step = mainStepModelzoo
			return nil
		}
	}
	return nil
}

// renderTaskModelView 按当前子步骤渲染创建任务页面。
func (m *mainModel) renderTaskModelView() string {
	var b strings.Builder
	b.WriteString(m.titleStyle.Render(i18n.T("tui.task_model.select_title", nil)))
	b.WriteString("\n\n")

	switch m.taskModel.step {
	case taskModelSelect:
		if len(m.taskModel.models) == 0 {
			b.WriteString(i18n.T("tui.task_model.no_models", nil))
		} else {
			summary := i18n.T("tui.task_model.filter_summary", map[string]any{"Total": len(m.taskModel.models)})
			b.WriteString(m.hintStyle.Render(summary))
		}
		b.WriteString("\n")
		b.WriteString(m.hintStyle.Render(i18n.T("tui.hint.select_back", nil)))

	case taskModelParamPoll:
		b.WriteString(m.titleStyle.Render(i18n.T("tui.task_model.param_poll_title", nil)))
		b.WriteString("\n\n")
		if m.taskModel.detail != nil && m.taskModel.currentParamIdx < len(m.taskModel.detail.InputParams) {
			p := m.taskModel.detail.InputParams[m.taskModel.currentParamIdx]
			total := len(m.taskModel.detail.InputParams)

			b.WriteString(m.hintStyle.Render(i18n.T("tui.task_model.param_progress", map[string]any{"Current": m.taskModel.currentParamIdx + 1, "Total": total})))
			b.WriteString("\n\n")

			label := i18n.APITranslate("param_label", p.FieldLabel)
			if p.Required {
				label += i18n.T("tui.task_model.required_marker", nil)
			}
			b.WriteString(m.titleStyle.Render(label))
			b.WriteString("\n")

			if p.VariableType == "image" || p.VariableType == "video" {
				b.WriteString(m.hintStyle.Render(i18n.T("tui.task_model.url_hint", map[string]any{"Type": variableTypeDisplayName(p.VariableType)})))
				b.WriteString("\n")
			} else if m.taskModel.usingSelect {
				b.WriteString(m.hintStyle.Render(i18n.T("tui.task_model.select_hint", nil)))
				b.WriteString("\n")
			} else {
				b.WriteString(m.hintStyle.Render(i18n.T("tui.task_model.text_hint", nil)))
				b.WriteString("\n")
			}

			if p.FieldTooltip != "" {
				b.WriteString(m.hintStyle.Render(p.FieldTooltip))
				b.WriteString("\n")
			}
			b.WriteString("\n")

			// 渲染输入控件（渲染前按可用空间校准高度，避免超出边框）
			if m.taskModel.usingSelect {
				m.taskModel.paramSelectList.SetHeight(m.paramInputHeight())
				b.WriteString(m.taskModel.paramSelectList.View())
			} else if m.taskModel.usingTA {
				m.taskModel.taParam.SetHeight(m.paramInputHeight())
				b.WriteString(m.taskModel.taParam.View())
			} else if ti, ok := m.taskModel.paramInputs[p.ParamKey()]; ok {
				b.WriteString(ti.View())
			}

			if m.taskModel.paramError != nil {
				errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171"))
				b.WriteString("\n")
				b.WriteString(errStyle.Render("✗ " + m.taskModel.paramError.Error()))
				b.WriteString("\n")
			} else {
				b.WriteString("\n\n")
			}
		}

	case taskModelOutputName:
		b.WriteString(m.titleStyle.Render(i18n.T("tui.task_model.output_name_title", nil)))
		b.WriteString("\n\n")
		b.WriteString(m.taskModel.outputNameInput.View())
		b.WriteString("\n\n")
		b.WriteString(m.hintStyle.Render(i18n.T("tui.task_model.output_name_hint", nil)))

	case taskModelPolling:
		spin := m.sp.View()
		b.WriteString(i18n.T("tui.task_model.generating_title", nil))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("%s: %s\n", i18n.T("cli.task.request_id_label", nil), m.taskModel.requestID))
		b.WriteString("\n")
		statusText := lib.ModelzooStatusName(m.taskModel.lastPollStatus)
		if m.taskModel.lastPollStatus == "" {
			statusText = i18n.T("tui.status.waiting_api", nil)
		}
		line := spin + " " + statusText
		if !m.taskModel.startedAt.IsZero() {
			line += " " + taskElapsedText(time.Since(m.taskModel.startedAt))
		}
		b.WriteString(line)
		b.WriteString("\n\n")
		b.WriteString(m.hintStyle.Render(i18n.T("tui.task_model.cancel_hint", nil)))

	case taskModelResult:
		b.WriteString(m.titleStyle.Render(i18n.T("tui.status.done", nil)))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("%s: %s\n", i18n.T("cli.task.request_id_label", nil), m.taskModel.requestID))
		b.WriteString("\n")
		if m.taskModel.lastPollStatus != "" {
			b.WriteString(fmt.Sprintf("%s: %s\n", i18n.T("tui.task_model.status_label", nil), lib.ModelzooStatusName(m.taskModel.lastPollStatus)))
		}
		b.WriteString(m.hintStyle.Render(i18n.T("tui.hint.return_menu", nil)))
	}

	return b.String()
}
