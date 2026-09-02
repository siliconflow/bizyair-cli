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

// initAIAppTaskParamInput 按参数类型初始化输入控件（与模型广场任务流程一致）：
// 图片/视频 → URL 输入框（优先，即使带枚举选项也按 URL 处理）；
// 布尔值 → 选择列表；枚举选项 → 选择列表；其余 → 多行文本域。
func (m *mainModel) initAIAppTaskParamInput() tea.Cmd {
	p := m.currentAIAppTaskParam()
	if p == nil {
		return nil
	}
	m.aiAppTask.paramInputs = make(map[string]textinput.Model)
	m.aiAppTask.usingTA = false
	m.aiAppTask.usingSelect = false

	switch {
	case p.VariableType == "image" || p.VariableType == "video":
		ti := textinput.New()
		ti.Placeholder = i18n.T("tui.task_model.url_placeholder", map[string]any{"Type": variableTypeDisplayName(p.VariableType)})
		if initial, ok := m.aiAppTaskParamInitialValue(p); ok {
			ti.SetValue(initial)
		}
		iw, _ := m.innerSize()
		if w := iw - 10; w > 10 {
			ti.Width = w
		}
		ti.Focus()
		m.aiAppTask.paramInputs[p.ParamKey()] = ti
	case p.VariableType == "boolean":
		m.initAIAppTaskParamSelect(p, []any{true, false})
	case p.FieldOptions != nil && len(p.FieldOptions.EnumValues()) > 0:
		m.initAIAppTaskParamSelect(p, p.FieldOptions.EnumValues())
	default:
		ta := textarea.New()
		ta.Placeholder = i18n.T("tui.task_model.text_placeholder", map[string]any{"Label": aiAppFieldLabel(p)})
		if initial, ok := m.aiAppTaskParamInitialValue(p); ok {
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
		m.aiAppTask.taParam = ta
		m.aiAppTask.usingTA = true
	}
	m.aiAppTask.paramRules = lib.DeriveModelzooRules(*p)
	m.aiAppTask.paramError = nil
	return nil
}

// currentAIAppTaskParam 返回当前正在填写的参数（越界返回 nil）。
func (m *mainModel) currentAIAppTaskParam() *lib.ModelzooInputParam {
	if m.aiAppTask.detail == nil || m.aiAppTask.currentParamIdx >= len(m.aiAppTask.fields) {
		return nil
	}
	return &m.aiAppTask.fields[m.aiAppTask.currentParamIdx]
}

// aiAppTaskParamInitialValue 返回参数初始值（优先已填值，其次 API 默认值）。
func (m *mainModel) aiAppTaskParamInitialValue(p *lib.ModelzooInputParam) (string, bool) {
	if v, ok := m.aiAppTask.params[p.ParamKey()]; ok && v != nil {
		return fieldValueDisplay(v), true
	}
	if p.FieldValue != nil {
		return fieldValueDisplay(p.FieldValue), true
	}
	return "", false
}

// initAIAppTaskParamSelect 为参数构建枚举选择列表，并预选已填值/API 默认值。
func (m *mainModel) initAIAppTaskParamSelect(p *lib.ModelzooInputParam, values []any) {
	m.aiAppTask.usingSelect = true
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
	l.Title = aiAppFieldLabel(p)
	localizeList(&l)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.Select(0)
	if defaultVal, ok := m.aiAppTaskParamInitialValue(p); ok && defaultVal != "" {
		for idx, item := range items {
			if li, ok := item.(listItem); ok && li.value == defaultVal {
				l.Select(idx)
				break
			}
		}
	}
	m.aiAppTask.paramSelectList = l
	m.syncListSizes()
}

// aiAppTaskParamHeight 返回参数选择列表/多行文本域在面板内可用的高度。
func (m *mainModel) aiAppTaskParamHeight() int {
	_, ih := m.innerSize()
	h := ih - 5 - 13 - 5 - len(m.aiAppTask.params)
	if h < 4 {
		h = 4
	}
	return h
}

// aiAppTaskGoBack 返回上一个参数；无参数可回退时返回应用列表页。
func (m *mainModel) aiAppTaskGoBack() tea.Cmd {
	if m.aiAppTask.currentParamIdx <= 0 {
		m.step = mainStepAIAppDetail
		return nil
	}
	m.aiAppTask.currentParamIdx--
	m.aiAppTask.step = aiAppTaskParamPoll
	m.aiAppTask.paramError = nil
	return m.initAIAppTaskParamInput()
}

// aiAppTaskSaveCurrent 校验并按类型转换后保存当前参数值；校验失败返回错误。
func (m *mainModel) aiAppTaskSaveCurrent() error {
	p := m.currentAIAppTaskParam()
	if p == nil {
		return nil
	}
	rules := m.aiAppTask.paramRules
	if len(rules) == 0 {
		rules = lib.DeriveModelzooRules(*p)
	}
	var raw string
	if m.aiAppTask.usingSelect {
		if it, ok := m.aiAppTask.paramSelectList.SelectedItem().(listItem); ok {
			raw = it.value
		}
	} else if m.aiAppTask.usingTA {
		raw = strings.TrimSpace(m.aiAppTask.taParam.Value())
	} else if ti, ok := m.aiAppTask.paramInputs[p.ParamKey()]; ok {
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
	m.aiAppTask.params[p.ParamKey()] = v
	return nil
}

// aiAppTaskValidate 校验所有必填输入节点均已填写（含 API 默认值）。
func (m *mainModel) aiAppTaskValidate() error {
	if m.aiAppTask.detail == nil {
		return nil
	}
	missing := lib.MissingWebAppRequiredParams(m.aiAppTask.detail, m.aiAppTask.params)
	if len(missing) > 0 {
		return fmt.Errorf("%s", i18n.T("tui.task_model.param_required", map[string]any{"Name": i18n.APITranslate("param_label", missing[0].FieldLabel)}))
	}
	return nil
}

// aiAppTaskApplyDefaults 为未填写的输入节点补入 API 默认值。
func (m *mainModel) aiAppTaskApplyDefaults() {
	lib.ApplyWebAppFieldDefaults(m.aiAppTask.params, m.aiAppTask.detail)
}

// aiAppTaskAdvance 前进到下一个参数；全部填完后校验、补默认值并创建任务。
func (m *mainModel) aiAppTaskAdvance() tea.Cmd {
	m.aiAppTask.currentParamIdx++
	if m.aiAppTask.detail == nil || m.aiAppTask.currentParamIdx >= len(m.aiAppTask.fields) {
		if err := m.aiAppTaskValidate(); err != nil {
			m.err = err
			return nil
		}
		m.aiAppTaskApplyDefaults()
		m.aiAppTask.step = aiAppTaskSubmitted
		m.running = true
		return createAIAppTask(m.getAPI(), lib.WebAppTaskCreateReq{
			WebAppId:    m.aiAppTask.detail.Id,
			InputValues: m.aiAppTask.params,
		})
	}
	return m.initAIAppTaskParamInput()
}

// aiAppFieldLabel 返回本地化参数标签。
func aiAppFieldLabel(p *lib.ModelzooInputParam) string {
	n := lib.WebAppInputNode{FieldLabel: p.FieldLabel, VariableName: p.VariableName}
	return lib.WebAppFieldLabel(n)
}

// updateAIAppTask 按当前子步骤分发消息。
func updateAIAppTask(m *mainModel, msg tea.Msg) (mainModel, tea.Cmd) {
	switch m.aiAppTask.step {
	case aiAppTaskParamPoll:
		return *m, m.updateAIAppTaskParamPoll(msg)
	case aiAppTaskSubmitted:
		return *m, nil
	case aiAppTaskPolling:
		return *m, m.updateAIAppTaskPolling(msg)
	case aiAppTaskResult:
		return *m, m.updateAIAppTaskResult(msg)
	}
	return *m, nil
}

// updateAIAppTaskParamPoll 处理参数填写页按键与输入更新。
func (m *mainModel) updateAIAppTaskParamPoll(msg tea.Msg) tea.Cmd {
	if m.aiAppTask.usingSelect {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "esc":
				return m.aiAppTaskGoBack()
			case "ctrl+u":
				m.aiAppTask.paramSelectList.Select(0)
				return nil
			case "ctrl+r":
				m.aiAppTask.params = make(map[string]any)
				m.aiAppTask.currentParamIdx = 0
				return m.initAIAppTaskParamInput()
			case "enter":
				p := m.currentAIAppTaskParam()
				if p != nil && p.Required {
					if it, ok := m.aiAppTask.paramSelectList.SelectedItem().(listItem); !ok || it.value == "" {
						m.err = fmt.Errorf("%s", i18n.T("tui.task_model.param_required", map[string]any{"Name": aiAppFieldLabel(p)}))
						return nil
					}
				}
				if err := m.aiAppTaskSaveCurrent(); err != nil {
					m.err = err
					return nil
				}
				return m.aiAppTaskAdvance()
			}
		}
		var cmd tea.Cmd
		m.aiAppTask.paramSelectList, cmd = m.aiAppTask.paramSelectList.Update(msg)
		return cmd
	}

	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			return m.aiAppTaskGoBack()
		case "ctrl+u":
			if m.aiAppTask.usingTA {
				m.aiAppTask.taParam.SetValue("")
			} else if p := m.currentAIAppTaskParam(); p != nil {
				if ti, ok := m.aiAppTask.paramInputs[p.ParamKey()]; ok {
					ti.SetValue("")
					m.aiAppTask.paramInputs[p.ParamKey()] = ti
				}
			}
			return nil
		case "ctrl+r":
			m.aiAppTask.params = make(map[string]any)
			m.aiAppTask.currentParamIdx = 0
			return m.initAIAppTaskParamInput()
		case "ctrl+s":
			if m.aiAppTask.usingTA {
				if err := m.aiAppTaskSaveCurrent(); err != nil {
					m.err = err
					return nil
				}
				return m.aiAppTaskAdvance()
			}
		case "enter":
			if !m.aiAppTask.usingTA {
				if err := m.aiAppTaskSaveCurrent(); err != nil {
					m.err = err
					return nil
				}
				return m.aiAppTaskAdvance()
			}
		}
	}

	if m.aiAppTask.usingTA {
		var cmd tea.Cmd
		m.aiAppTask.taParam, cmd = m.aiAppTask.taParam.Update(msg)
		m.aiAppTask.paramError = lib.ValidateModelzooParam(m.aiAppTask.paramRules, m.aiAppTask.taParam.Value())
		return cmd
	}
	p := m.currentAIAppTaskParam()
	if p != nil {
		if ti, ok := m.aiAppTask.paramInputs[p.ParamKey()]; ok {
			var cmd tea.Cmd
			ti, cmd = ti.Update(msg)
			m.aiAppTask.paramInputs[p.ParamKey()] = ti
			m.aiAppTask.paramError = lib.ValidateModelzooParam(m.aiAppTask.paramRules, ti.Value())
			return cmd
		}
	}
	return nil
}

// updateAIAppTaskPolling 处理轮询状态按键。
func (m *mainModel) updateAIAppTaskPolling(msg tea.Msg) tea.Cmd {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			m.step = mainStepAIAppDetail
			m.running = false
			return nil
		}
	}
	return nil
}

// updateAIAppTaskResult 处理结果页按键（含复制快捷键）。
func (m *mainModel) updateAIAppTaskResult(msg tea.Msg) tea.Cmd {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "enter", "esc":
			m.step = mainStepAIApp
			return nil
		case "ctrl+y":
			if len(m.outputURLs) > 0 {
				return m.copyResultURLs()
			}
		case "ctrl+u":
			if m.aiAppTask.requestID != "" {
				m.copyFeedback = writeClipboard(m.aiAppTask.requestID, i18n.T("tui.task_model.copied_request_id", nil))
				return m.feedbackClearCmd()
			}
		}
	}
	return nil
}

// renderAIAppTaskView 渲染 AI 应用任务页面。
func (m *mainModel) renderAIAppTaskView() string {
	var b strings.Builder
	if m.aiAppTask.step != aiAppTaskResult {
		b.WriteString(m.titleStyle.Render(i18n.T("tui.ai_app.running_title", nil)))
		b.WriteString("\n\n")
	}

	switch m.aiAppTask.step {
	case aiAppTaskParamPoll:
		if p := m.currentAIAppTaskParam(); p != nil {
			total := len(m.aiAppTask.fields)
			label := i18n.APITranslate("param_label", p.FieldLabel)
			if p.Required {
				label += i18n.T("tui.task_model.required_marker", nil)
			}
			b.WriteString(m.titleStyle.Render(i18n.T("tui.ai_app.task_nav_title", map[string]any{
				"Current": m.aiAppTask.currentParamIdx + 1, "Total": total, "Name": label,
			})))
			b.WriteString("\n\n")

			if p.VariableType == "image" || p.VariableType == "video" {
				b.WriteString(m.hintStyle.Render(i18n.T("tui.task_model.url_hint", map[string]any{"Type": variableTypeDisplayName(p.VariableType)})))
				b.WriteString("\n")
			} else if m.aiAppTask.usingSelect {
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

			if m.aiAppTask.usingSelect {
				m.aiAppTask.paramSelectList.SetHeight(m.aiAppTaskParamHeight())
				b.WriteString(m.aiAppTask.paramSelectList.View())
			} else if m.aiAppTask.usingTA {
				m.aiAppTask.taParam.SetHeight(m.aiAppTaskParamHeight())
				b.WriteString(m.aiAppTask.taParam.View())
			} else if ti, ok := m.aiAppTask.paramInputs[p.ParamKey()]; ok {
				b.WriteString(ti.View())
			}

			if m.aiAppTask.paramError != nil {
				errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171"))
				b.WriteString("\n")
				b.WriteString(errStyle.Render("✗ " + m.aiAppTask.paramError.Error()))
				b.WriteString("\n")
			} else {
				b.WriteString("\n\n")
			}
			b.WriteString(m.hintStyle.Render(i18n.T("tui.task_model.param_hint_input", nil)))
		}

	case aiAppTaskSubmitted:
		b.WriteString(m.renderAIAppTaskReview())
		b.WriteString("\n\n")
		b.WriteString(m.hintStyle.Render(i18n.T("tui.ai_app.hint_task_poll", nil)))

	case aiAppTaskPolling:
		spin := m.sp.View()
		b.WriteString(i18n.T("tui.ai_app.generating_title", nil))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("%s %s\n", i18n.T("cli.app.run.request_id_label", nil), m.aiAppTask.requestID))
		b.WriteString("\n")
		statusText := lib.WebAppTaskStatusName(m.aiAppTask.lastPollStatus)
		if m.aiAppTask.lastPollStatus == "" {
			statusText = i18n.T("tui.status.waiting_api", nil)
		}
		line := spin + " " + statusText
		if !m.aiAppTask.startedAt.IsZero() {
			line += " " + taskElapsedText(time.Since(m.aiAppTask.startedAt))
		}
		b.WriteString(line)
		b.WriteString("\n\n")
		b.WriteString(m.hintStyle.Render(i18n.T("tui.ai_app.hint_task_poll", nil)))

	case aiAppTaskResult:
		b.WriteString(m.titleStyle.Render(i18n.T("tui.ai_app.task_result_nav", nil)))
		b.WriteString("\n\n")
		statusColor := "#34D399"
		statusIcon := "✓ "
		switch m.aiAppTask.lastPollStatus {
		case lib.TaskStatusFailed:
			statusColor = "#F87171"
			statusIcon = "✗ "
		case lib.TaskStatusCancelled:
			statusColor = "#FBBF24"
			statusIcon = "✗ "
		}
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(statusColor)).Render(statusIcon + lib.WebAppTaskStatusName(m.aiAppTask.lastPollStatus)))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("%s %s\n", i18n.T("cli.app.run.request_id_label", nil), m.aiAppTask.requestID))
		b.WriteString("\n")
		if len(m.outputURLs) > 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399")).Render(i18n.T("tui.task_model.result_url_label", nil)))
			b.WriteString("\n")
			b.WriteString(strings.Join(m.outputURLs, "\n"))
			b.WriteString("\n")
		}
		if m.copyFeedback != "" {
			b.WriteString("\n")
			b.WriteString(m.renderStyledHint(m.copyFeedback))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(m.hintStyle.Render(i18n.T("tui.ai_app.hint_task_result", nil)))
	}

	return b.String()
}

// renderAIAppTaskReview 渲染提交前的参数摘要。
func (m *mainModel) renderAIAppTaskReview() string {
	var b strings.Builder
	b.WriteString(m.titleStyle.Render(i18n.T("tui.ai_app.task_review_title", nil)))
	b.WriteString("\n\n")
	if len(m.aiAppTask.params) == 0 {
		b.WriteString(m.hintStyle.Render(i18n.T("tui.ai_app.task_no_params", nil)))
		return b.String()
	}
	paramStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	for _, f := range m.aiAppTask.fields {
		key := f.ParamKey()
		v, ok := m.aiAppTask.params[key]
		if !ok || v == nil {
			continue
		}
		b.WriteString(paramStyle.Render(fmt.Sprintf("%s: %v", aiAppFieldLabel(&f), fieldValueDisplay(v))))
		b.WriteString("\n")
	}
	return b.String()
}

// aiAppTaskOutputURLs 提取任务输出对象中的 URL 列表。
func aiAppTaskOutputURLs(outputs []lib.WebAppTaskOutput) []string {
	urls := make([]string, 0, len(outputs))
	for _, o := range outputs {
		if o.ObjectURL != "" {
			urls = append(urls, o.ObjectURL)
		}
	}
	return urls
}
