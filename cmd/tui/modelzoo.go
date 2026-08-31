package tui

import (
	"fmt"
	"sort"
	"strings"

	tablev2 "charm.land/bubbles/v2/table"
	lipglossv2 "charm.land/lipgloss/v2"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
)

// updateModelzooTable 从 filtered 列表重建表格。
func (m *mainModel) updateModelzooTable() {
	titles := []string{
		i18n.T("tui.modelzoo.col_name", nil),
		i18n.T("tui.modelzoo.col_endpoint", nil),
		i18n.T("tui.modelzoo.col_manufacturer", nil),
		i18n.T("tui.modelzoo.col_category", nil),
		i18n.T("tui.modelzoo.col_version", nil),
	}
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

	rows := make([]tablev2.Row, 0, len(m.modelzoo.filtered))
	for _, model := range m.modelzoo.filtered {
		cells := []string{
			dash(model.DisplayName),
			dash(model.Endpoint),
			i18n.APITranslate("manufacturer", model.Manufacturer),
			model.Category,
			i18n.APITranslate("version", model.ModelVersion),
		}
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
	h := m.height - 18
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
	m.modelzooTable = t
}

func (m *mainModel) extractModelzooFilterOptions() {
	seriesCount := make(map[string]int)
	mfrCount := make(map[string]int)
	capCount := make(map[string]int)
	verCount := make(map[string]int)

	for _, model := range m.modelzoo.models {
		series := model.Series
		if series == "" {
			series = lib.SeriesFromEndpoint(model.Endpoint)
		}
		if series != "" {
			seriesCount[series]++
		}
		if model.Manufacturer != "" {
			mfrCount[model.Manufacturer]++
		}
		if model.Category != "" {
			capCount[model.Category]++
		}
		if ver := model.ModelVersion; ver != "" {
			verCount[ver]++
		}
	}

	m.modelzoo.seriesOpts = translateFilterOptions(sortedFilterOptions(seriesCount), "series")
	m.modelzoo.manufacturerOpts = translateFilterOptions(sortedFilterOptions(mfrCount), "manufacturer")
	m.modelzoo.versionOpts = translateFilterOptions(sortedFilterOptions(verCount), "version")

	// 能力选项优先使用权威的 categories 接口（类别树，含 api_count），
	// 未加载或为空时回退到按 model.Category 统计。
	capOpts := sortedFilterOptions(capCount)
	if len(m.modelzoo.categories) > 0 {
		catOpts := make([]filterOption, 0, len(m.modelzoo.categories))
		for _, c := range m.modelzoo.categories {
			if c.Category != "" {
				catOpts = append(catOpts, filterOption{label: c.Category, value: c.Category, count: c.APICount})
			}
		}
		if len(catOpts) > 0 {
			capOpts = catOpts
		}
	}
	m.modelzoo.capabilityOpts = translateFilterOptions(capOpts, "capability")
}

func hasModelzooTag(model lib.ModelzooModelFlat, tag string) bool {
	if tag == "" {
		return false
	}
	for _, t := range model.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return strings.EqualFold(model.Category, tag)
}

func sortedFilterOptions(counts map[string]int) []filterOption {
	opts := make([]filterOption, 0, len(counts))
	for val, cnt := range counts {
		opts = append(opts, filterOption{label: val, value: val, count: cnt})
	}
	sort.Slice(opts, func(i, j int) bool {
		if opts[i].count != opts[j].count {
			return opts[i].count > opts[j].count
		}
		return opts[i].label < opts[j].label
	})
	return opts
}

// translateFilterOptions 将过滤选项标签按指定领域翻译为本地化文本。
func translateFilterOptions(opts []filterOption, domain string) []filterOption {
	for i := range opts {
		opts[i].label = i18n.APITranslate(domain, opts[i].label)
	}
	return opts
}

// applyModelzooFilters 按当前过滤条件及搜索关键字筛选模型列表。
func (m *mainModel) applyModelzooFilters() {
	filtered := m.modelzoo.models

	if m.modelzoo.seriesFilter != "" {
		var keep []lib.ModelzooModelFlat
		for _, model := range filtered {
			series := model.Series
			if series == "" {
				series = lib.SeriesFromEndpoint(model.Endpoint)
			}
			if strings.EqualFold(series, m.modelzoo.seriesFilter) {
				keep = append(keep, model)
			}
		}
		filtered = keep
	}

	if m.modelzoo.manufacturerFilter != "" {
		var keep []lib.ModelzooModelFlat
		for _, model := range filtered {
			if strings.EqualFold(model.Manufacturer, m.modelzoo.manufacturerFilter) {
				keep = append(keep, model)
			}
		}
		filtered = keep
	}

	if m.modelzoo.capabilityFilter != "" {
		var keep []lib.ModelzooModelFlat
		for _, model := range filtered {
			if hasModelzooTag(model, m.modelzoo.capabilityFilter) {
				keep = append(keep, model)
			}
		}
		filtered = keep
	}

	if m.modelzoo.versionFilter != "" {
		var keep []lib.ModelzooModelFlat
		for _, model := range filtered {
			if strings.EqualFold(model.ModelVersion, m.modelzoo.versionFilter) {
				keep = append(keep, model)
			}
		}
		filtered = keep
	}

	q := strings.ToLower(m.modelzoo.search.Value())
	if q != "" {
		var keep []lib.ModelzooModelFlat
		for _, model := range filtered {
			if strings.Contains(strings.ToLower(model.DisplayName), q) ||
				strings.Contains(strings.ToLower(model.Endpoint), q) ||
				strings.Contains(strings.ToLower(model.Manufacturer), q) ||
				strings.Contains(strings.ToLower(model.Category), q) ||
				strings.Contains(strings.ToLower(model.Series), q) {
				keep = append(keep, model)
			}
		}
		filtered = keep
	}

	m.modelzoo.filtered = filtered
	m.updateModelzooTable()
}

// updateModelzoo 处理模型广场列表页面的消息与按键。
func updateModelzoo(m *mainModel, msg tea.Msg) (mainModel, tea.Cmd) {
	// Search focus mode
	if m.modelzoo.searchActive && m.step == mainStepModelzoo {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.modelzoo.searchActive = false
				m.modelzoo.search.Blur()
				m.modelzooTable.Focus()
				return *m, nil
			case "enter":
				m.modelzoo.searchActive = false
				m.modelzoo.search.Blur()
				m.modelzooTable.Focus()
				m.applyModelzooFilters()
				return *m, nil
			}
		}
		var cmd tea.Cmd
		m.modelzoo.search, cmd = m.modelzoo.search.Update(msg)
		return *m, cmd
	}

	// Filter picker mode
	if m.modelzoo.filterMode != modelzooFilterNone && m.step == mainStepModelzoo {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.modelzoo.filterMode = modelzooFilterNone
				m.modelzooTable.Focus()
				return *m, nil
			case "enter":
				if it, ok := m.modelzoo.filterList.SelectedItem().(listItem); ok {
					m.applyModelzooFilterSelection(it.value)
				}
				m.modelzoo.filterMode = modelzooFilterNone
				m.modelzooTable.Focus()
				return *m, nil
			}
		}
		var cmd tea.Cmd
		m.modelzoo.filterList, cmd = m.modelzoo.filterList.Update(msg)
		return *m, cmd
	}

	// Normal table mode — handle key shortcuts
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "/":
			m.modelzoo.searchActive = true
			m.modelzooTable.Blur()
			return *m, m.modelzoo.search.Focus()
		case "r":
			m.modelzoo.seriesFilter = ""
			m.modelzoo.manufacturerFilter = ""
			m.modelzoo.capabilityFilter = ""
			m.modelzoo.versionFilter = ""
			m.modelzoo.search.SetValue("")
			m.applyModelzooFilters()
			return *m, nil
		case "s":
			return *m, m.openModelzooFilterPicker(modelzooFilterSeries)
		case "m":
			return *m, m.openModelzooFilterPicker(modelzooFilterManufacturer)
		case "c":
			return *m, m.openModelzooFilterPicker(modelzooFilterCapability)
		case "v":
			return *m, m.openModelzooFilterPicker(modelzooFilterVersion)
		case "enter":
			idx := m.modelzooTable.Cursor()
			if idx >= 0 && idx < len(m.modelzoo.filtered) && m.modelzoo.filtered[idx].Endpoint != "" {
				m.running = true
				return *m, fetchEndpointDetail(m.getAPI(), m.modelzoo.filtered[idx].Endpoint)
			}
		case "p":
			idx := m.modelzooTable.Cursor()
			if idx >= 0 && idx < len(m.modelzoo.filtered) && m.modelzoo.filtered[idx].Endpoint != "" {
				return *m, m.openPriceView(m.modelzoo.filtered[idx].Endpoint)
			}
		case "t":
			idx := m.modelzooTable.Cursor()
			if idx >= 0 && idx < len(m.modelzoo.filtered) && m.modelzoo.filtered[idx].Endpoint != "" {
				m.step = mainStepTaskModel
				m.taskModel = taskModelInputs{
					models:   m.modelzoo.filtered,
					step:     taskModelParamPoll,
					params:   make(map[string]any),
					detail:   nil,
					endpoint: m.modelzoo.filtered[idx].Endpoint,
				}
				m.running = true
				return *m, fetchEndpointDetail(m.getAPI(), m.modelzoo.filtered[idx].Endpoint)
			}
		case "esc":
			m.step = mainStepMenu
			return *m, nil
		}
	}

	// Table navigation
	if m.step == mainStepModelzoo {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				m.modelzooTable.MoveUp(1)
			case "down", "j":
				m.modelzooTable.MoveDown(1)
			case "pgup":
				m.modelzooTable.MoveUp(m.modelzooTable.Height())
			case "pgdown":
				m.modelzooTable.MoveDown(m.modelzooTable.Height())
			case "home":
				m.modelzooTable.GotoTop()
			case "end":
				m.modelzooTable.GotoBottom()
			}
		}
	}

	return *m, nil
}

// openModelzooFilterPicker 打开指定维度的过滤选择器。
func (m *mainModel) openModelzooFilterPicker(mode modelzooFilterMode) tea.Cmd {
	m.modelzoo.filterMode = mode
	m.modelzooTable.Blur()

	var items []list.Item
	allLabel := fmt.Sprintf("%s (%d)", i18n.T("tui.modelzoo.filter_all", nil), len(m.modelzoo.filtered))
	items = append(items, listItem{title: allLabel, value: ""})

	var opts []filterOption
	var title string
	switch mode {
	case modelzooFilterSeries:
		opts = m.modelzoo.seriesOpts
		title = i18n.T("tui.modelzoo.filter_series_title", nil)
	case modelzooFilterManufacturer:
		opts = m.modelzoo.manufacturerOpts
		title = i18n.T("tui.modelzoo.filter_manufacturer_title", nil)
	case modelzooFilterCapability:
		opts = m.modelzoo.capabilityOpts
		title = i18n.T("tui.modelzoo.filter_capability_title", nil)
	case modelzooFilterVersion:
		opts = m.modelzoo.versionOpts
		title = i18n.T("tui.modelzoo.filter_version_title", nil)
	}

	for _, opt := range opts {
		items = append(items, listItem{
			title: fmt.Sprintf("%s (%d)", opt.label, opt.count),
			value: opt.value,
		})
	}

	d := list.NewDefaultDelegate()
	cSel := lipgloss.Color("#C4B5FD")
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(cSel).BorderLeftForeground(cSel)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(cSel)

	fl := list.New(items, d, 0, 0)
	fl.Title = title
	fl.SetShowStatusBar(false)
	fl.SetShowPagination(false)
	m.modelzoo.filterList = fl
	m.syncListSizes()
	return nil
}

// applyModelzooFilterSelection 将过滤选择写入对应维度状态并重新应用过滤。
func (m *mainModel) applyModelzooFilterSelection(value string) {
	switch m.modelzoo.filterMode {
	case modelzooFilterSeries:
		m.modelzoo.seriesFilter = value
	case modelzooFilterManufacturer:
		m.modelzoo.manufacturerFilter = value
	case modelzooFilterCapability:
		m.modelzoo.capabilityFilter = value
	case modelzooFilterVersion:
		m.modelzoo.versionFilter = value
	}
	m.applyModelzooFilters()
}

// modelzooDetailMaxScroll 返回详情页最大可滚动行数。
func (m *mainModel) modelzooDetailMaxScroll() int {
	_, ih := m.innerSize()
	maxH := ih - 4
	if maxH < 1 {
		maxH = 1
	}
	total := len(strings.Split(m.renderModelzooDetailView(), "\n"))
	if total <= maxH {
		return 0
	}
	return total - maxH
}

// updateModelzooDetail 处理模型详情页按键。
func updateModelzooDetail(m *mainModel, msg tea.Msg) (mainModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "t":
			if m.endpointDetail != nil {
				m.step = mainStepTaskModel
				m.taskModel = taskModelInputs{
					endpoint: m.endpointDetail.Endpoint,
					detail:   m.endpointDetail,
					step:     taskModelParamPoll,
					params:   make(map[string]any),
				}
				return *m, m.initCurrentParamInput()
			}
		case "p":
			if m.endpointDetail != nil {
				return *m, m.openPriceView(m.endpointDetail.Endpoint)
			}
		case "esc":
			m.step = mainStepModelzoo
			return *m, nil
		case "up", "k":
			if m.modelzooDetailScroll > 0 {
				m.modelzooDetailScroll--
			}
		case "down", "j":
			m.modelzooDetailScroll++
		case "pgup":
			m.modelzooDetailScroll -= 10
		case "pgdown":
			m.modelzooDetailScroll += 10
		}
		max := m.modelzooDetailMaxScroll()
		if m.modelzooDetailScroll > max {
			m.modelzooDetailScroll = max
		}
		if m.modelzooDetailScroll < 0 {
			m.modelzooDetailScroll = 0
		}
	}
	return *m, nil
}

// renderModelzooDetailView 渲染模型详情页。
func (m *mainModel) renderModelzooDetailView() string {
	var b strings.Builder
	if m.endpointDetail == nil {
		b.WriteString(i18n.T("tui.modelzoo.no_models", nil))
		return b.String()
	}
	d := m.endpointDetail
	b.WriteString(m.titleStyle.Render(d.DisplayName))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Render(d.Endpoint))
	b.WriteString("\n\n")

	infoLabelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	infoValueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	b.WriteString(fmt.Sprintf("%s: %s\n",
		infoLabelStyle.Render(i18n.T("tui.modelzoo.manufacturer_label", nil)+": "),
		infoValueStyle.Render(i18n.APITranslate("manufacturer", d.Manufacturer))))
	b.WriteString(fmt.Sprintf("%s: %s\n",
		infoLabelStyle.Render(i18n.T("tui.modelzoo.category_label", nil)+": "),
		infoValueStyle.Render(d.Category)))
	b.WriteString(fmt.Sprintf("%s: %s\n",
		infoLabelStyle.Render(i18n.T("tui.modelzoo.version_label", nil)+": "),
		infoValueStyle.Render(i18n.APITranslate("version", d.Edition))))
	b.WriteString(fmt.Sprintf("%s: %s\n",
		infoLabelStyle.Render(i18n.T("tui.modelzoo.billing_unit_label", nil)+": "),
		infoValueStyle.Render(i18n.APITranslate("billing_unit", d.BillingUnit))))
	b.WriteString(fmt.Sprintf("%s: %s\n",
		infoLabelStyle.Render(i18n.T("tui.modelzoo.min_credits_label", nil)+": "),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#FACC15")).Bold(true).Render(creditsToUSD(d.MinCredits))))

	if d.Description != "" {
		innerW, _ := m.innerSize()
		descW := innerW - 16
		if descW < 10 {
			descW = 10
		}
		wrapped := lipgloss.NewStyle().Width(descW).Render(d.Description)
		b.WriteString("\n\n")
		b.WriteString(sectionCard(i18n.T("tui.modelzoo.section_intro", nil), wrapped, m.width-19))
	}

	if len(d.InputParams) > 0 {
		var pb strings.Builder
		paramStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		for i, p := range d.InputParams {
			if i > 0 {
				pb.WriteString("\n")
			}
			pb.WriteString(paramStyle.Render(fmt.Sprintf("%d. %s", i+1, i18n.APITranslate("param_label", p.FieldLabel))))
		}
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA")).
			Bold(true).
			Render(i18n.T("tui.modelzoo.params_label", nil)))
		b.WriteString("\n")
		b.WriteString(pb.String())
	}

	return b.String()
}

// renderModelzooView 渲染模型广场列表页面。
func (m *mainModel) renderModelzooView() string {
	if m.running && len(m.modelzoo.models) == 0 {
		return m.renderStyledHint(i18n.T("tui.hint.wait", nil))
	}
	if len(m.modelzoo.models) == 0 && !m.modelzooLoaded {
		return m.renderStyledHint(i18n.T("tui.modelzoo.no_models", nil))
	}

	var bar string
	if m.modelzoo.searchActive {
		bar = m.modelzoo.search.View()
	} else {
		bar = m.renderModelzooFilterBar()
	}

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#60A5FA")).
		Bold(true).
		Render(i18n.T("tui.menu.modelzoo", nil))
	if m.running {
		title += " " + m.sp.View()
	}

	// 过滤选择器激活时只渲染 title + filterbar + list，与 my_models 保持一致
	if m.modelzoo.filterMode != modelzooFilterNone {
		return lipgloss.JoinVertical(lipgloss.Left, title, bar, m.modelzoo.filterList.View())
	}

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(bar)
	b.WriteString("\n")

	// Count
	total := len(m.modelzoo.filtered)
	count := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).
		Render(i18n.T("tui.modelzoo.filter_summary", map[string]any{"Total": total}))
	b.WriteString(count)
	b.WriteString("\n")

	// Content
	if total == 0 {
		b.WriteString(i18n.T("tui.modelzoo.no_models", nil))
	} else {
		b.WriteString(renderTable(m.modelzooTable))
	}
	return b.String()
}

// renderModelzooFilterBar 渲染过滤条。
func (m *mainModel) renderModelzooFilterBar() string {
	allLabel := i18n.T("tui.modelzoo.filter_all", nil)

	seriesDisplay := allLabel
	if m.modelzoo.seriesFilter != "" {
		seriesDisplay = i18n.APITranslate("series", m.modelzoo.seriesFilter)
	}
	mfrDisplay := allLabel
	if m.modelzoo.manufacturerFilter != "" {
		mfrDisplay = i18n.APITranslate("manufacturer", m.modelzoo.manufacturerFilter)
	}
	capDisplay := allLabel
	if m.modelzoo.capabilityFilter != "" {
		capDisplay = i18n.APITranslate("category", m.modelzoo.capabilityFilter)
	}
	verDisplay := allLabel
	if m.modelzoo.versionFilter != "" {
		verDisplay = i18n.APITranslate("version", m.modelzoo.versionFilter)
	}
	searchDisplay := allLabel
	if sq := m.modelzoo.search.Value(); sq != "" {
		searchDisplay = sq
	}

	items := []filterBarItem{
		{label: i18n.T("tui.modelzoo.filter_series", nil), value: seriesDisplay, active: m.modelzoo.seriesFilter != ""},
		{label: i18n.T("tui.modelzoo.filter_manufacturer", nil), value: mfrDisplay, active: m.modelzoo.manufacturerFilter != ""},
		{label: i18n.T("tui.modelzoo.filter_capability", nil), value: capDisplay, active: m.modelzoo.capabilityFilter != ""},
		{label: i18n.T("tui.modelzoo.filter_version", nil), value: verDisplay, active: m.modelzoo.versionFilter != ""},
		{label: i18n.T("tui.modelzoo.key_search", nil), value: searchDisplay, active: m.modelzoo.search.Value() != ""},
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
