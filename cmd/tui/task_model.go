package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
)

// initCurrentParamInput 初始化当前参数的输入控件。
func (m *mainModel) initCurrentParamInput() tea.Cmd {
	if m.taskModel.detail == nil || m.currentParamIdx() >= len(m.taskModel.detail.InputParams) {
		return nil
	}
	p := m.taskModel.detail.InputParams[m.currentParamIdx()]

	switch {
	case p.FieldOptions != nil && len(p.FieldOptions.EnumValues()) > 0:
		m.initParamSelectList(p, p.FieldOptions.EnumValues())
	case p.VariableType == "boolean":
		m.initParamSelectList(p, []any{true, false})
	case p.VariableType == "image" || p.VariableType == "video":
		ti := textinput.New()
		ti.Placeholder = i18n.T("tui.task_model.url_placeholder", map[string]any{"Type": variableTypeDisplayName(p.VariableType)})
		if initial, ok := m.paramInitialValue(p); ok {
			ti.SetValue(initial)
		}
		ti.Focus()
		m.taskModel.paramText = ti
	default:
		ti := textinput.New()
		ti.Placeholder = i18n.T("tui.task_model.text_placeholder", map[string]any{"Label": i18n.APITranslate("param_label", p.FieldLabel)})
		if initial, ok := m.paramInitialValue(p); ok {
			ti.SetValue(initial)
		}
		ti.Focus()
		m.taskModel.paramText = ti
	}
	return nil
}

// currentParamIdx returns the current parameter index.
func (m *mainModel) currentParamIdx() int {
	return m.taskModel.currentParamIdx
}

// currentParam 返回当前正在填写的参数定义。
func (m *mainModel) currentParam() *lib.ModelzooInputParam {
	if m.taskModel.detail == nil || m.taskModel.currentParamIdx >= len(m.taskModel.detail.InputParams) {
		return nil
	}
	return &m.taskModel.detail.InputParams[m.taskModel.currentParamIdx]
}

// paramInitialValue 返回参数输入框初始值。
func (m *mainModel) paramInitialValue(p lib.ModelzooInputParam) (string, bool) {
	if v, ok := m.taskModel.params[p.ParamKey()]; ok && v != nil {
		return fieldValueDisplay(v), true
	}
	if p.FieldValue != nil {
		return fieldValueDisplay(p.FieldValue), true
	}
	return "", false
}

// fieldValueDisplay 将 API 下发的参数字段默认值规范化为可编辑的纯文本。
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

// initParamSelectList 为参数构建枚举选择列表。
func (m *mainModel) initParamSelectList(p lib.ModelzooInputParam, values []any) {
	items := make([]list.Item, 0, len(values))
	for _, v := range values {
		items = append(items, listItem{title: fmt.Sprintf("%v", v), value: fmt.Sprintf("%v", v)})
	}
	d := list.NewDefaultDelegate()
	cSel := lipgloss.Color("#C4B5FD")
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(cSel).BorderLeftForeground(cSel)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(cSel)
	l := list.New(items, d, 0, 0)
	l.Title = i18n.APITranslate("param_label", p.FieldLabel)
	localizeList(&l)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	m.taskModel.paramSelectList = l
	m.syncListSizes()
}

// saveCurrentParamValue 校验并保存当前参数值到 params。
func (m *mainModel) saveCurrentParamValue() bool {
	p := m.currentParam()
	if p == nil {
		return true
	}

	var raw string
	if m.taskModel.paramSelectList.Items() != nil && len(m.taskModel.paramSelectList.Items()) > 0 {
		// Check if we're in select mode by checking if paramSelectList has items
		if it, ok := m.taskModel.paramSelectList.SelectedItem().(listItem); ok {
			raw = it.value
		}
	}
	if raw == "" {
		raw = strings.TrimSpace(m.taskModel.paramText.Value())
	}

	if raw == "" {
		if p.Required {
			return false
		}
		return true
	}

	switch p.VariableType {
	case "number", "float":
		f, _ := strconv.ParseFloat(raw, 64)
		m.taskModel.params[p.ParamKey()] = f
	case "integer":
		i, _ := strconv.ParseInt(raw, 10, 64)
		m.taskModel.params[p.ParamKey()] = i
	case "boolean":
		b, _ := strconv.ParseBool(raw)
		m.taskModel.params[p.ParamKey()] = b
	default:
		m.taskModel.params[p.ParamKey()] = raw
	}
	return true
}

// validateTaskParams 校验所有必填参数均已填写。
func (m *mainModel) validateTaskParams() error {
	if m.taskModel.detail == nil {
		return nil
	}
	for _, p := range m.taskModel.detail.InputParams {
		if !p.Required {
			continue
		}
		key := p.ParamKey()
		val, exists := m.taskModel.params[key]
		if !exists || val == nil || val == "" {
			if p.FieldValue != nil {
				continue
			}
			return fmt.Errorf("%s", i18n.T("tui.task_model.param_required", map[string]any{"Name": i18n.APITranslate("param_label", p.FieldLabel)}))
		}
	}
	return nil
}

// applyFieldDefaults 为未填写的参数补入 API 默认值。
func (m *mainModel) applyFieldDefaults() {
	if m.taskModel.detail == nil {
		return
	}
	for _, p := range m.taskModel.detail.InputParams {
		key := p.ParamKey()
		if _, exists := m.taskModel.params[key]; !exists && p.FieldValue != nil {
			m.taskModel.params[key] = p.FieldValue
		}
	}
}

// advanceParam 前进到下一个参数；全部填完后校验必填项并进入轮询步骤。
func (m *mainModel) advanceParam() tea.Cmd {
	m.taskModel.currentParamIdx++
	if m.taskModel.detail == nil || m.taskModel.currentParamIdx >= len(m.taskModel.detail.InputParams) {
		if err := m.validateTaskParams(); err != nil {
			m.err = err
			return nil
		}
		m.applyFieldDefaults()
		m.taskModel.step = taskModelPolling
		m.running = true
		return createTaskCmd(m.getAPI(), m.taskModel.endpoint, m.taskModel.params)
	}
	return m.initCurrentParamInput()
}

// updateTaskModel 按当前子步骤分发消息。
func updateTaskModel(m *mainModel, msg tea.Msg) (mainModel, tea.Cmd) {
	switch m.taskModel.step {
	case taskModelSelect:
		// Not used in current flow — task model always starts at taskModelParamPoll
		return *m, nil
	case taskModelParamPoll:
		return updateParamPoll(m, msg)
	case taskModelPolling:
		return updateTaskPolling(m, msg)
	case taskModelResult:
		return updateTaskResult(m, msg)
	}
	return *m, nil
}

// updateParamPoll 处理参数填写页按键与输入更新。
func updateParamPoll(m *mainModel, msg tea.Msg) (mainModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return *m, tea.Quit
		case "esc":
			m.step = mainStepModelzoo
			return *m, nil
		case "ctrl+u":
			m.taskModel.paramText.SetValue("")
			return *m, nil
		case "ctrl+r":
			m.taskModel.params = make(map[string]any)
			m.taskModel.currentParamIdx = 0
			return *m, m.initCurrentParamInput()
		case "enter":
			if !m.saveCurrentParamValue() {
				p := m.currentParam()
				if p != nil {
					m.err = fmt.Errorf("%s", i18n.T("tui.task_model.param_required", map[string]any{"Name": i18n.APITranslate("param_label", p.FieldLabel)}))
				}
				return *m, nil
			}
			return *m, m.advanceParam()
		}
	}

	// Update input controls
	if m.taskModel.paramSelectList.Items() != nil && len(m.taskModel.paramSelectList.Items()) > 0 {
		var cmd tea.Cmd
		m.taskModel.paramSelectList, cmd = m.taskModel.paramSelectList.Update(msg)
		return *m, cmd
	}

	var cmd tea.Cmd
	m.taskModel.paramText, cmd = m.taskModel.paramText.Update(msg)
	return *m, cmd
}

// updateTaskPolling 处理轮询状态按键。
func updateTaskPolling(m *mainModel, msg tea.Msg) (mainModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "x":
			if m.taskModel.requestID != "" {
				return *m, cancelTaskCmd(m.getAPI(), m.taskModel.requestID)
			}
		}
	}
	return *m, nil
}

// updateTaskResult 处理结果页按键。
func updateTaskResult(m *mainModel, msg tea.Msg) (mainModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			m.step = mainStepModelzoo
			return *m, nil
		}
	}
	return *m, nil
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
			}

			if p.FieldTooltip != "" {
				b.WriteString(m.hintStyle.Render(p.FieldTooltip))
				b.WriteString("\n")
			}
			b.WriteString("\n")

			// Render input control
			if m.taskModel.paramSelectList.Items() != nil && len(m.taskModel.paramSelectList.Items()) > 0 {
				b.WriteString(m.taskModel.paramSelectList.View())
			} else {
				b.WriteString(m.taskModel.paramText.View())
			}

			// Show filled params
			if len(m.taskModel.params) > 0 {
				b.WriteString("\n\n")
				b.WriteString(m.hintStyle.Render(i18n.T("tui.task_model.filled_params", nil)))
				b.WriteString("\n")
				filledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
				for _, prevP := range m.taskModel.detail.InputParams {
					if val, ok := m.taskModel.params[prevP.ParamKey()]; ok {
						prevLabel := i18n.APITranslate("param_label", prevP.FieldLabel)
						line := fmt.Sprintf("  ✓ %s: %v\n", prevLabel, val)
						b.WriteString(filledStyle.Render(line))
					}
				}
			}
		}

	case taskModelPolling:
		spin := m.sp.View()
		b.WriteString(i18n.T("tui.task_model.generating_title", nil))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("%s: %s\n", i18n.T("cli.task.request_id_label", nil), m.taskModel.requestID))
		b.WriteString("\n")
		if m.taskModel.lastPollStatus != "" {
			b.WriteString(spin + " " + m.taskModel.lastPollStatus)
		} else {
			b.WriteString(spin + " " + i18n.T("tui.status.waiting_api", nil))
		}
		b.WriteString("\n\n")
		b.WriteString(m.hintStyle.Render(i18n.T("tui.task_model.cancel_hint", nil)))

	case taskModelResult:
		b.WriteString(m.titleStyle.Render(i18n.T("tui.status.done", nil)))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("%s: %s\n", i18n.T("cli.task.request_id_label", nil), m.taskModel.requestID))
		b.WriteString("\n")
		b.WriteString(m.hintStyle.Render(i18n.T("tui.hint.return_menu", nil)))
	}

	return b.String()
}
