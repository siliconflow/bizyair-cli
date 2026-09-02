package tui

import (
	"fmt"
	"strings"

	tablev2 "charm.land/bubbles/v2/table"
	lipglossv2 "charm.land/lipgloss/v2"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
)

// updateAIAppTable 从 filtered 列表重建 AI 应用表格。
func (m *mainModel) updateAIAppTable() {
	titles := lib.AIApplicationTableColumns()
	ideal := make([]int, len(titles))
	for i, t := range titles {
		ideal[i] = lipglossv2.Width(t)
	}

	maxW := m.width - 16
	if maxW < 24 {
		maxW = 24
	}
	overhead := len(titles)*2 + 2
	if maxW > overhead {
		maxW -= overhead
	}

	rows := make([]tablev2.Row, 0, len(m.aiApp.filtered))
	for _, app := range m.aiApp.filtered {
		cells := lib.AIApplicationTableRow(app)
		for i, c := range cells {
			if w := lipglossv2.Width(c); w > ideal[i] {
				ideal[i] = w
			}
		}
		rows = append(rows, cells)
	}

	cols := fitColumnWidthsByContent(ideal, maxW, true)
	for i, t := range titles {
		cols[i].Title = t
	}

	iw, _ := m.innerSize()
	lw := iw - 6
	if lw < 10 {
		lw = 10
	}
	tw := lw - 4
	if tw < 20 {
		tw = 20
	}
	h := m.height - 20
	if h < 4 {
		h = 4
	}

	t := tablev2.New(
		tablev2.WithColumns(cols),
		tablev2.WithRows(rows),
		tablev2.WithWidth(tw),
		tablev2.WithHeight(h),
		tablev2.WithFocused(true),
	)
	applyTableStyles(&t)
	m.aiAppTable = t
}

// extractAIAppFilterOptions 从列表提取基座模型过滤选项。
func (m *mainModel) extractAIAppFilterOptions() {
	baseModels := lib.AIApplicationBaseModels(m.aiApp.apps)
	counts := make(map[string]int)
	for _, app := range m.aiApp.apps {
		if bm := lib.AIApplicationBaseModel(app); bm != "" {
			counts[bm]++
		}
	}
	opts := make([]filterOption, 0, len(baseModels))
	for _, bm := range baseModels {
		opts = append(opts, filterOption{label: bm, value: bm, count: counts[bm]})
	}
	m.aiApp.baseModelOpts = opts

	sortOpts := make([]filterOption, 0, len(lib.AIAppSortOptions()))
	for _, s := range lib.AIAppSortOptions() {
		sortOpts = append(sortOpts, filterOption{label: i18n.T("tui.ai_app.sort_"+s, nil), value: s})
	}
	m.aiApp.sortOpts = sortOpts
}

// aiAppFilter 组装当前筛选条件。
func (m *mainModel) aiAppFilter() lib.AIApplicationFilter {
	return lib.AIApplicationFilter{
		Sort:      m.aiApp.sortBy,
		BaseModel: m.aiApp.baseModelFilter,
		Keyword:   m.aiApp.search.Value(),
	}
}

// applyAIAppFilters 按当前筛选条件过滤并排序应用列表。
func (m *mainModel) applyAIAppFilters() {
	filtered := lib.FilterAIApplications(m.aiApp.apps, m.aiAppFilter())
	m.aiApp.filtered = filtered
	m.updateAIAppTable()
}

// updateAIApp 处理 AI 应用列表页面的消息与按键。
func updateAIApp(m *mainModel, msg tea.Msg) (mainModel, tea.Cmd) {
	// Search focus mode
	if m.aiApp.searchActive && m.step == mainStepAIApp {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.aiApp.searchActive = false
				m.aiApp.search.Blur()
				m.aiAppTable.Focus()
				return *m, nil
			case "enter":
				m.aiApp.searchActive = false
				m.aiApp.search.Blur()
				m.aiAppTable.Focus()
				m.applyAIAppFilters()
				return *m, nil
			}
		}
		var cmd tea.Cmd
		m.aiApp.search, cmd = m.aiApp.search.Update(msg)
		return *m, cmd
	}

	// Filter picker mode
	if m.aiApp.filterMode != aiAppFilterNone && m.step == mainStepAIApp {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.aiApp.filterMode = aiAppFilterNone
				m.aiAppTable.Focus()
				return *m, nil
			case "enter":
				if it, ok := m.aiApp.filterList.SelectedItem().(listItem); ok {
					m.applyAIAppFilterSelection(it.value)
				}
				m.aiApp.filterMode = aiAppFilterNone
				m.aiAppTable.Focus()
				return *m, nil
			}
		}
		var cmd tea.Cmd
		m.aiApp.filterList, cmd = m.aiApp.filterList.Update(msg)
		return *m, cmd
	}

	// Normal table mode — handle key shortcuts
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "/":
			m.aiApp.searchActive = true
			m.aiAppTable.Blur()
			return *m, m.aiApp.search.Focus()
		case "r":
			m.aiApp.sortBy = lib.AIAppSortRecent
			m.aiApp.baseModelFilter = ""
			m.aiApp.search.SetValue("")
			m.applyAIAppFilters()
			return *m, nil
		case "s":
			return *m, m.openAIAppFilterPicker(aiAppFilterSort)
		case "m":
			if len(m.aiApp.baseModelOpts) == 0 {
				return *m, nil
			}
			return *m, m.openAIAppFilterPicker(aiAppFilterBaseModel)
		case "enter":
			idx := m.aiAppTable.Cursor()
			if idx >= 0 && idx < len(m.aiApp.filtered) {
				app := m.aiApp.filtered[idx]
				if app != nil && len(app.Versions) > 0 && app.Versions[0] != nil {
					m.running = true
					return *m, fetchAIAppDetail(m.getAPI(), app.Versions[0].Id)
				}
			}
		case "esc":
			m.step = mainStepMenu
			return *m, nil
		}
	}

	// Table navigation
	if m.step == mainStepAIApp {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				m.aiAppTable.MoveUp(1)
			case "down", "j":
				m.aiAppTable.MoveDown(1)
			case "pgup":
				m.aiAppTable.MoveUp(m.aiAppTable.Height())
			case "pgdown":
				m.aiAppTable.MoveDown(m.aiAppTable.Height())
			case "home":
				m.aiAppTable.GotoTop()
			case "end":
				m.aiAppTable.GotoBottom()
			}
		}
	}

	return *m, nil
}

// openAIAppFilterPicker 打开指定维度的过滤选择器。
func (m *mainModel) openAIAppFilterPicker(mode aiAppFilterMode) tea.Cmd {
	m.aiApp.filterMode = mode
	m.aiAppTable.Blur()

	var items []list.Item
	var title string
	switch mode {
	case aiAppFilterSort:
		title = i18n.T("tui.ai_app.filter_sort_title", nil)
		for _, opt := range m.aiApp.sortOpts {
			label := opt.label
			if opt.value == m.aiApp.sortBy {
				label = "✓ " + label
			}
			items = append(items, listItem{title: label, value: opt.value})
		}
	case aiAppFilterBaseModel:
		title = i18n.T("tui.ai_app.filter_base_model_title", nil)
		allLabel := i18n.T("tui.ai_app.filter_all", nil)
		if m.aiApp.baseModelFilter == "" {
			allLabel = "✓ " + allLabel
		}
		items = append(items, listItem{title: allLabel, value: ""})
		for _, opt := range m.aiApp.baseModelOpts {
			label := fmt.Sprintf("%s (%d)", opt.label, opt.count)
			if opt.value == m.aiApp.baseModelFilter {
				label = "✓ " + label
			}
			items = append(items, listItem{title: label, value: opt.value})
		}
	}

	d := list.NewDefaultDelegate()
	cSel := lipgloss.Color("#C4B5FD")
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(cSel).BorderLeftForeground(cSel)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(cSel)

	fl := list.New(items, d, 0, 0)
	fl.Title = title
	fl.SetShowStatusBar(false)
	fl.SetShowPagination(false)
	m.aiApp.filterList = fl
	m.syncListSizes()
	return nil
}

// applyAIAppFilterSelection 将过滤选择写入对应维度状态并重新应用过滤。
func (m *mainModel) applyAIAppFilterSelection(value string) {
	switch m.aiApp.filterMode {
	case aiAppFilterSort:
		m.aiApp.sortBy = value
	case aiAppFilterBaseModel:
		m.aiApp.baseModelFilter = value
	}
	m.aiAppTable.SetCursor(0)
	m.applyAIAppFilters()
}

// aiAppDetailMaxScroll 返回详情页最大可滚动行数。
func (m *mainModel) aiAppDetailMaxScroll() int {
	_, ih := m.innerSize()
	maxH := ih - 4
	if maxH < 1 {
		maxH = 1
	}
	total := len(strings.Split(m.renderAIAppDetailView(), "\n"))
	if total <= maxH {
		return 0
	}
	return total - maxH
}

// updateAIAppDetail 处理 AI 应用详情页按键。
func updateAIAppDetail(m *mainModel, msg tea.Msg) (mainModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "t", "enter":
			if m.aiApp.detail != nil && m.aiApp.detail.Id > 0 {
				m.step = mainStepAIAppTask
				m.aiAppTask = aiAppTaskInputs{
					detail: m.aiApp.detail,
					fields: lib.WebAppInputParams(m.aiApp.detail),
					step:   aiAppTaskParamPoll,
					params: make(map[string]any),
				}
				// 无输入节点定义时直接提交，由服务端填充默认参数。
				if len(m.aiAppTask.fields) == 0 {
					m.aiAppTask.step = aiAppTaskSubmitted
					m.running = true
					return *m, createAIAppTask(m.getAPI(), lib.WebAppTaskCreateReq{
						WebAppId:    m.aiApp.detail.Id,
						InputValues: m.aiAppTask.params,
					})
				}
				return *m, m.initAIAppTaskParamInput()
			}
		case "esc":
			m.step = mainStepAIApp
			return *m, nil
		case "up", "k":
			if m.aiAppDetailScroll > 0 {
				m.aiAppDetailScroll--
			}
		case "down", "j":
			m.aiAppDetailScroll++
		case "pgup":
			m.aiAppDetailScroll -= 10
		case "pgdown":
			m.aiAppDetailScroll += 10
		}
		max := m.aiAppDetailMaxScroll()
		if m.aiAppDetailScroll > max {
			m.aiAppDetailScroll = max
		}
		if m.aiAppDetailScroll < 0 {
			m.aiAppDetailScroll = 0
		}
	}
	return *m, nil
}

// renderAIAppDetailView 渲染 AI 应用详情页。
func (m *mainModel) renderAIAppDetailView() string {
	var b strings.Builder
	if m.aiApp.detail == nil {
		b.WriteString(i18n.T("tui.ai_app.no_apps", nil))
		return b.String()
	}
	d := m.aiApp.detail
	b.WriteString(m.titleStyle.Render(d.Name))
	b.WriteString("\n\n")

	infoLabelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	infoValueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	b.WriteString(fmt.Sprintf("%s: %s\n",
		infoLabelStyle.Render(i18n.T("tui.ai_app.base_model_label", nil)+": "),
		infoValueStyle.Render(lib.DisplayOrDash(d.BaseModel))))
	owner := d.NickName
	if owner == "" {
		owner = d.Creator
	}
	b.WriteString(fmt.Sprintf("%s: %s\n",
		infoLabelStyle.Render(i18n.T("tui.ai_app.author_label", nil)+": "),
		infoValueStyle.Render(lib.DisplayOrDash(owner))))
	b.WriteString(fmt.Sprintf("%s: %s\n",
		infoLabelStyle.Render(i18n.T("tui.ai_app.created_label", nil)+": "),
		infoValueStyle.Render(formatTimestamp(d.CreatedAt))))

	if d.Description != "" {
		innerW, _ := m.innerSize()
		descW := innerW - 16
		if descW < 10 {
			descW = 10
		}
		wrapped := lipgloss.NewStyle().Width(descW).Render(d.Description)
		b.WriteString("\n\n")
		b.WriteString(sectionCard(i18n.T("tui.ai_app.section_desc", nil), wrapped, m.width-19))
	}

	if len(d.InputNodes) > 0 {
		var pb strings.Builder
		paramStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		for i, n := range d.InputNodes {
			if i > 0 {
				pb.WriteString("\n")
			}
			var t string
			if mt := lib.WebAppNodeMediaType(n); mt != "" {
				t = m.hintStyle.Render("[" + i18n.T("tui.ai_app.type_"+mt, nil) + "]")
			}
			pb.WriteString(paramStyle.Render(fmt.Sprintf("%d. %s%s", i+1, lib.WebAppFieldLabel(n), t)))
		}
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Bold(true).Render(i18n.T("tui.ai_app.params_label", nil)))
		b.WriteString("\n")
		b.WriteString(pb.String())
	}

	b.WriteString("\n\n")
	b.WriteString(m.hintStyle.Render(i18n.T("tui.ai_app.hint_detail", nil)))
	return b.String()
}

// renderAIAppView 渲染 AI 应用列表页面。
func (m *mainModel) renderAIAppView() string {
	if m.running && len(m.aiApp.apps) == 0 {
		return m.renderStyledHint(i18n.T("tui.hint.wait", nil))
	}
	if len(m.aiApp.apps) == 0 && !m.aiAppLoaded {
		return m.renderStyledHint(i18n.T("tui.ai_app.no_apps", nil))
	}

	var bar string
	if m.aiApp.searchActive {
		bar = m.aiApp.search.View()
	} else {
		bar = m.renderAIAppFilterBar()
	}

	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Bold(true).Render(i18n.T("tui.menu.ai_app", nil))
	if m.running {
		title += " " + m.sp.View()
	}

	if m.aiApp.filterMode != aiAppFilterNone {
		return lipgloss.JoinVertical(lipgloss.Left, title, bar, m.aiApp.filterList.View())
	}

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(bar)
	b.WriteString("\n")

	total := len(m.aiApp.filtered)
	count := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).
		Render(i18n.T("tui.ai_app.filter_summary", map[string]any{"Total": total}))
	b.WriteString(count)
	b.WriteString("\n")

	if total == 0 {
		b.WriteString(i18n.T("tui.ai_app.no_apps", nil))
	} else {
		b.WriteString(renderTable(m.aiAppTable))
	}
	return b.String()
}

// renderAIAppFilterBar 渲染过滤条。
func (m *mainModel) renderAIAppFilterBar() string {
	allLabel := i18n.T("tui.ai_app.filter_all", nil)

	sortDisplay := i18n.T("tui.ai_app.sort_"+m.aiApp.sortBy, nil)
	bmDisplay := allLabel
	if m.aiApp.baseModelFilter != "" {
		bmDisplay = m.aiApp.baseModelFilter
	}
	searchDisplay := allLabel
	if sq := m.aiApp.search.Value(); sq != "" {
		searchDisplay = sq
	}

	items := []filterBarItem{
		{label: i18n.T("tui.ai_app.key_sort", nil), value: sortDisplay, active: m.aiApp.sortBy != "" && m.aiApp.sortBy != lib.AIAppSortRecent},
		{label: i18n.T("tui.ai_app.key_base_model", nil), value: bmDisplay, active: m.aiApp.baseModelFilter != ""},
		{label: i18n.T("tui.ai_app.key_search", nil), value: searchDisplay, active: m.aiApp.search.Value() != ""},
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
	var b strings.Builder
	for i, item := range items {
		valStyle := style
		if item.active {
			valStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))
		}
		b.WriteString(style.Render(item.label + ": "))
		b.WriteString(valStyle.Render(item.value))
		if i < len(items)-1 {
			b.WriteString(style.Render(" | "))
		}
	}
	return b.String()
}