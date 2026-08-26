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
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/lib/format"
	"github.com/siliconflow/bizyair-cli/meta"
)

// myModelTitles 我的模型列表表头
var myModelTitles = []string{
	i18n.T("tui.my_models.col_name", nil),
	i18n.T("tui.my_models.col_type", nil),
	i18n.T("tui.my_models.col_versions", nil),
	i18n.T("tui.my_models.col_base_model", nil),
	i18n.T("tui.my_models.col_used", nil),
	i18n.T("tui.my_models.col_liked", nil),
	i18n.T("tui.my_models.col_public", nil),
}

// updateMyModelsTable 根据当前模型数据动态计算列宽并重建表格。
// 数据变更或过滤/排序/搜索状态变更后调用；渲染时仅读取存储的表格。
func (m *mainModel) updateMyModelsTable() {
	models := m.myModels

	ideal := make([]int, len(myModelTitles))
	for i, t := range myModelTitles {
		ideal[i] = lipglossv2.Width(t)
	}

	maxW := m.width - 16
	if maxW < 40 {
		maxW = m.width - 16
	}
	if maxW < 24 {
		maxW = 24
	}
	overhead := len(myModelTitles)*2 + 2
	if maxW > overhead {
		maxW -= overhead
	}

	rows := make([]tablev2.Row, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}
		bm := modelBaseModel(model)
		cells := []string{
			dash(model.Name),
			dash(model.Type),
			fmt.Sprintf("%d", len(model.Versions)),
			dash(bm),
			fmt.Sprintf("%d", model.Counter.UsedCount),
			fmt.Sprintf("%d", model.Counter.LikedCount),
			publicStatusText(modelPublic(model)),
		}
		for i, c := range cells {
			if w := lipglossv2.Width(c); w > ideal[i] {
				ideal[i] = w
			}
		}
		rows = append(rows, cells)
	}

	cols := fitColumnWidthsByContent(ideal, maxW, true)
	for i, t := range myModelTitles {
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
	m.myModelsTable = t
}

// modelBaseModel 返回模型首个版本的基座模型（列表展示用）
func modelBaseModel(model *lib.BizyModelInfo) string {
	if len(model.Versions) > 0 {
		return model.Versions[0].BaseModel
	}
	return ""
}

// modelPublic 返回模型首个版本的公开状态（列表展示用）
func modelPublic(model *lib.BizyModelInfo) bool {
	if len(model.Versions) > 0 {
		return model.Versions[0].Public
	}
	return false
}

func publicStatusText(public bool) string {
	if public {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render(i18n.T("tui.my_models.col_public_yes", nil))
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(i18n.T("tui.my_models.col_public_no", nil))
}

// extractMyModelsFilterOptions 构建筛选选项：
//   - 类型：meta.ModelTypes 全量列表 + 本地计数
//   - 基座模型：API 返回的 m.baseModelTypes + 本地计数
//   - 排序：API sort 参数值（Recently / Most Liked / Most Downloaded / Most Used / Most Forked）
func (m *mainModel) extractMyModelsFilterOptions() {
	typeCount := make(map[string]int)
	bmCount := make(map[string]int)
	for _, model := range m.myModels {
		if model == nil {
			continue
		}
		typeCount[model.Type]++
		if bm := modelBaseModel(model); bm != "" {
			bmCount[bm]++
		}
	}

	// 类型选项 — 全量 meta.ModelTypes + 计数
	typeOpts := make([]filterOption, 0, len(meta.ModelTypes)+1)
	typeOpts = append(typeOpts, filterOption{label: i18n.T("tui.my_models.type_all", nil), value: "", count: m.myModelsTotal})
	for _, t := range meta.ModelTypes {
		ts := string(t)
		typeOpts = append(typeOpts, filterOption{label: ts, value: ts, count: typeCount[ts]})
	}
	m.myModelsInputs.typeOpts = typeOpts

	// 排序选项 — API sort 参数值
	m.myModelsInputs.sortOpts = []filterOption{
		{label: i18n.T("tui.my_models.sort_recently", nil), value: "Recently"},
		{label: i18n.T("tui.my_models.sort_most_liked", nil), value: "Most Liked"},
		{label: i18n.T("tui.my_models.sort_most_downloaded", nil), value: "Most Downloaded"},
		{label: i18n.T("tui.my_models.sort_most_used", nil), value: "Most Used"},
		{label: i18n.T("tui.my_models.sort_most_forked", nil), value: "Most Forked"},
	}

	// 基座模型选项 — API 返回的完整列表 + 计数
	bmOpts := make([]filterOption, 0, len(m.baseModelTypes)+1)
	bmOpts = append(bmOpts, filterOption{label: i18n.T("tui.my_models.type_all", nil), value: "", count: 0})
	for _, bm := range m.baseModelTypes {
		label := bm.Label
		if label == "" {
			label = bm.Value
		}
		bmOpts = append(bmOpts, filterOption{label: label, value: bm.Value, count: bmCount[bm.Value]})
	}
	m.myModelsInputs.bmOpts = bmOpts
}

// openMyModelsFilterPicker 打开指定维度的过滤选择器（list 弹窗），table 失焦。
func (m *mainModel) openMyModelsFilterPicker(mode myModelsFilterMode) tea.Cmd {
	m.myModelsInputs.filterMode = mode
	m.myModelsTable.Blur()

	var items []list.Item
	var opts []filterOption
	var title string
	switch mode {
	case myModelsFilterType:
		opts = m.myModelsInputs.typeOpts
		title = i18n.T("tui.my_models.filter_type_title", nil)
	case myModelsFilterSort:
		opts = m.myModelsInputs.sortOpts
		title = i18n.T("tui.my_models.filter_sort_title", nil)
	case myModelsFilterBaseModel:
		opts = m.myModelsInputs.bmOpts
		title = i18n.T("tui.my_models.filter_base_model_title", nil)
	}
	if len(opts) == 0 {
		m.myModelsInputs.filterMode = myModelsFilterNone
		m.myModelsTable.Focus()
		return nil
	}

	for _, o := range opts {
		itemTitle := o.label
		if mode == myModelsFilterType || mode == myModelsFilterBaseModel {
			itemTitle = fmt.Sprintf("%s (%d)", o.label, o.count)
		}
		items = append(items, listItem{title: itemTitle, value: o.value})
	}

	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#C4B5FD")).
		BorderLeftForeground(lipgloss.Color("#C4B5FD"))
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#C4B5FD"))

	l := list.New(items, d, 0, 0)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	m.myModelsInputs.filterList = l
	m.syncListSizes()
	return nil
}

// applyMyModelsFilterSelection 将选择的过滤值写入对应字段。
func (m *mainModel) applyMyModelsFilterSelection(value string) {
	switch m.myModelsInputs.filterMode {
	case myModelsFilterType:
		m.myModelsInputs.typeFilter = value
	case myModelsFilterSort:
		m.myModelsInputs.sortBy = value
	case myModelsFilterBaseModel:
		m.myModelsInputs.baseModelFilter = value
	}
}

// renderFilterBar 渲染过滤状态条。
func (m mainModel) renderFilterBar() string {
	allLabel := i18n.T("tui.my_models.type_all", nil)

	typeDisplay := allLabel
	if m.myModelsInputs.typeFilter != "" {
		typeDisplay = m.myModelsInputs.typeFilter
	}
	sortDisplay := allLabel
	if m.myModelsInputs.sortBy != "" {
		sortDisplay = m.myModelsInputs.sortBy
		for _, s := range m.myModelsInputs.sortOpts {
			if s.value == m.myModelsInputs.sortBy {
				sortDisplay = s.label
				break
			}
		}
	}
	bmDisplay := allLabel
	if m.myModelsInputs.baseModelFilter != "" {
		bmDisplay = m.myModelsInputs.baseModelFilter
	}

	searchDisplay := allLabel
	if sq := m.myModelsInputs.search.Value(); sq != "" {
		searchDisplay = sq
	}

	items := []filterBarItem{
		{label: i18n.T("tui.my_models.filter_type_title", nil), value: typeDisplay, active: m.myModelsInputs.typeFilter != ""},
		{label: i18n.T("tui.my_models.filter_sort_title", nil), value: sortDisplay, active: m.myModelsInputs.sortBy != ""},
		{label: i18n.T("tui.my_models.filter_base_model_title", nil), value: bmDisplay, active: m.myModelsInputs.baseModelFilter != ""},
		{label: i18n.T("tui.my_models.key_search", nil), value: searchDisplay, active: m.myModelsInputs.search.Value() != ""},
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

func (m *mainModel) renderMyModelsView() string {
	if m.running && len(m.myModels) == 0 {
		return m.renderStyledHint(i18n.T("tui.hint.wait", nil))
	}
	if len(m.myModels) == 0 && !m.myModelsLoaded {
		return m.renderStyledHint(i18n.T("tui.my_models.empty", nil))
	}

	models := m.myModels

	var bar string
	if m.myModelsInputs.searchActive {
		bar = m.myModelsInputs.search.View()
	} else {
		bar = m.renderFilterBar()
	}

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#60A5FA")).
		Bold(true).
		Render(i18n.T("tui.menu.models", nil))
	if m.running {
		title += " " + m.sp.View()
	}

	count := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).
		Render(i18n.T("tui.my_models.showing_count", map[string]any{"Count": len(models)}))

	// 删除确认：在表格上方叠加确认提示
	if m.deleteConfirmModel != nil {
		confirm := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true).Render(
			i18n.T("tui.my_models.delete_confirm", map[string]any{"Name": m.deleteConfirmModel.Name}))
		hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(
			i18n.T("tui.my_models.delete_hint", nil))
		return lipgloss.JoinVertical(lipgloss.Left, title, bar, confirm, hint, renderTable(m.myModelsTable), count)
	}

	if m.myModelsInputs.filterMode != myModelsFilterNone {
		return lipgloss.JoinVertical(lipgloss.Left, title, bar, m.myModelsInputs.filterList.View())
	}
	parts := []string{title, bar, renderTable(m.myModelsTable), count}
	if len(m.myModelsTable.Rows()) == 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(i18n.T("tui.my_models.no_match", nil)))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *mainModel) renderModelDetailView() string {
	d := m.modelDetail
	if d == nil {
		return m.renderStyledHint(i18n.T("tui.hint.wait", nil))
	}

	// 切换公开确认提示
	if m.publicConfirm {
		publicLabel := i18n.T("tui.my_models.col_public_yes", nil)
		if !m.publicConfirmNew {
			publicLabel = i18n.T("tui.my_models.col_public_no", nil)
		}
		confirm := lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Bold(true).Render(
			i18n.T("tui.my_models.public_confirm", map[string]any{"Public": publicLabel}))
		hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(
			i18n.T("tui.my_models.public_hint", nil))
		return lipgloss.JoinVertical(lipgloss.Left, confirm, hint)
	}

	// 删除确认提示叠加在详情内容上方
	if m.deleteConfirmModel != nil {
		confirm := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true).Render(
			i18n.T("tui.my_models.delete_confirm", map[string]any{"Name": m.deleteConfirmModel.Name}))
		hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(
			i18n.T("tui.my_models.delete_hint", nil))
		return lipgloss.JoinVertical(lipgloss.Left, confirm, hint)
	}

	isPublic := false
	if len(d.Versions) > 0 {
		isPublic = d.Versions[0].Public
	}

	labels := []string{
		i18n.T("tui.my_models.detail_name", nil),
		i18n.T("tui.my_models.detail_type", nil),
		i18n.T("tui.my_models.detail_versions", nil),
		i18n.T("tui.my_models.col_used", nil),
		i18n.T("tui.my_models.col_liked", nil),
		i18n.T("tui.my_models.detail_public_status", nil),
		i18n.T("tui.my_models.detail_created", nil),
	}
	values := []string{
		dash(d.Name),
		dash(d.Type),
		fmt.Sprintf("%d", len(d.Versions)),
		fmt.Sprintf("%d", d.Counter.UsedCount),
		fmt.Sprintf("%d", d.Counter.LikedCount),
		publicStatusText(isPublic),
		dash(d.CreatedAt),
	}
	basicTable := renderKeyValueTable(labels, values, m.width-24)

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Bold(true).Render(i18n.T("tui.my_models.detail_title", nil)))
	b.WriteString("\n\n")
	b.WriteString(sectionCard(i18n.T("tui.my_models.detail_title", nil), basicTable, m.width-19))

	if len(d.Versions) > 0 {
		var vBuf strings.Builder
		verLabels := []string{
			i18n.T("tui.my_models.detail_version", nil),
			i18n.T("tui.my_models.version_size", nil),
			i18n.T("tui.my_models.col_base_model", nil),
			i18n.T("tui.my_models.col_public", nil),
		}
		for i, v := range d.Versions {
			vv := []string{
				dash(v.Version),
				format.FormatBytes(v.FileSize),
				dash(v.BaseModel),
				publicStatusText(v.Public),
			}
			vBuf.WriteString(renderKeyValueTable(verLabels, vv, m.width-24))
			if i < len(d.Versions)-1 {
				vBuf.WriteString("\n")
			}
		}
		b.WriteString("\n\n")
		b.WriteString(sectionCard(i18n.T("tui.my_models.detail_versions_header", nil), vBuf.String(), m.width-19))
	}

	return b.String()
}

func (m *mainModel) handleMyModelsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Filter picker handling must come FIRST so ESC/Enter don't fall through
	// to the main switch (which would go back to menu or open detail).
	if m.myModelsInputs.filterMode != myModelsFilterNone && m.step == mainStepMyModelsList {
		switch msg.String() {
		case "esc":
			m.myModelsInputs.filterMode = myModelsFilterNone
			m.myModelsTable.Focus()
			return m, nil
		case "enter":
			if it, ok := m.myModelsInputs.filterList.SelectedItem().(listItem); ok {
				m.applyMyModelsFilterSelection(it.value)
			}
			m.myModelsInputs.filterMode = myModelsFilterNone
			m.myModelsTable.Focus()
			m.running = true
			return m, m.fetchMyModels()
		}
		var cmd tea.Cmd
		m.myModelsInputs.filterList, cmd = m.myModelsInputs.filterList.Update(msg)
		return m, cmd
	}

	// 搜索激活时只有 esc/enter 穿透到主 switch，其余按键交给 textinput
	if m.myModelsInputs.searchActive && m.step == mainStepMyModelsList {
		switch msg.String() {
		case "esc", "enter":
		default:
			var cmd tea.Cmd
			m.myModelsInputs.search, cmd = m.myModelsInputs.search.Update(msg)
			return m, cmd
		}
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		if m.publicConfirm {
			m.publicConfirm = false
			return m, nil
		}
		if m.deleteConfirmModel != nil {
			m.deleteConfirmModel = nil
			return m, nil
		}
		if m.step == mainStepModelDetail {
			m.modelDetail = nil
			m.myModelsInputs.typeFilter = ""
			m.myModelsInputs.sortBy = ""
			m.myModelsInputs.baseModelFilter = ""
			m.myModelsInputs.search.SetValue("")
			m.myModelsInputs.searchActive = false
			m.running = true
			m.step = mainStepMyModelsList
			return m, m.fetchMyModels()
		}
		if m.myModelsInputs.searchActive {
			m.myModelsInputs.searchActive = false
			m.myModelsInputs.search.Blur()
			m.running = true
			return m, m.fetchMyModels()
		}
		m.step = mainStepMenu
		return m, nil
	case "enter":
		if m.publicConfirm {
			m.publicConfirm = false
			m.running = true
			return m, m.toggleModelPublicCmd(m.publicConfirmIDs, m.publicConfirmNew)
		}
		if m.deleteConfirmModel != nil {
			return m.confirmDelete()
		}
		if m.myModelsInputs.searchActive {
			m.myModelsInputs.searchActive = false
			m.myModelsInputs.search.Blur()
			m.running = true
			return m, m.fetchMyModels()
		}
		if m.step == mainStepMyModelsList && len(m.myModelsTable.Rows()) > 0 {
			idx := m.myModelsTable.Cursor()
			models := m.myModels
			if idx >= 0 && idx < len(models) {
				m.step = mainStepModelDetail
				return m, m.fetchModelDetail(models[idx].Id)
			}
		}
	case "d":
		if m.deleteConfirmModel != nil {
			return m.confirmDelete()
		}
		if m.step == mainStepModelDetail && m.modelDetail != nil {
			m.deleteConfirmModel = &lib.BizyModelInfo{Id: m.modelDetail.Id, Name: m.modelDetail.Name}
			return m, nil
		}
	case "p":
		if m.publicConfirm {
			m.publicConfirm = false
			m.running = true
			return m, m.toggleModelPublicCmd(m.publicConfirmIDs, m.publicConfirmNew)
		}
		if m.step == mainStepModelDetail && m.modelDetail != nil && len(m.modelDetail.Versions) > 0 {
			m.publicConfirmIDs = actions.ExtractVersionIDs(m.modelDetail)
			m.publicConfirmNew = !m.modelDetail.Versions[0].Public
			m.publicConfirm = true
			return m, nil
		}
	case "/":
		if m.step == mainStepMyModelsList {
			m.myModelsInputs.searchActive = true
			m.myModelsInputs.search.Focus()
			return m, nil
		}
	case "t":
		if m.step == mainStepMyModelsList {
			return m, m.openMyModelsFilterPicker(myModelsFilterType)
		}
	case "s":
		if m.step == mainStepMyModelsList {
			return m, m.openMyModelsFilterPicker(myModelsFilterSort)
		}
	case "b":
		if m.step == mainStepMyModelsList {
			return m, m.openMyModelsFilterPicker(myModelsFilterBaseModel)
		}
	case "r":
		if m.step == mainStepMyModelsList {
			m.myModelsInputs.typeFilter = ""
			m.myModelsInputs.sortBy = ""
			m.myModelsInputs.baseModelFilter = ""
			m.myModelsInputs.search.SetValue("")
			m.myModelsInputs.searchActive = false
			m.running = true
			return m, m.fetchMyModels()
		}
	}

	// Table navigation — 手动路由到 v2 table 导航方法；
	// 不走 Update(msg) 因为 v1 KeyMsg 与 v2 tea.KeyPressMsg 类型不兼容。
	if m.step == mainStepMyModelsList {
		switch msg.String() {
		case "up", "k":
			m.myModelsTable.MoveUp(1)
		case "down", "j":
			m.myModelsTable.MoveDown(1)
		case "pgup":
			m.myModelsTable.MoveUp(m.myModelsTable.Height())
		case "pgdown":
			m.myModelsTable.MoveDown(m.myModelsTable.Height())
		case "home":
			m.myModelsTable.GotoTop()
		case "end":
			m.myModelsTable.GotoBottom()
		}
		return m, nil
	}

	return m, nil
}

func (m *mainModel) confirmDelete() (tea.Model, tea.Cmd) {
	modelID := m.deleteConfirmModel.Id
	m.deleteConfirmModel = nil
	m.running = true
	return m, m.deleteModelCmd(modelID)
}

func (m *mainModel) fetchMyModels() tea.Cmd {
	return func() tea.Msg {
		return fetchMyModelsList(m.getAPI(), m.apiKey, m.baseDomain, m.myModelsInputs)
	}
}

func (m *mainModel) fetchModelDetail(modelId int64) tea.Cmd {
	return func() tea.Msg {
		return fetchModelDetail(m.getAPI(), modelId)
	}
}

func (m *mainModel) deleteModelCmd(modelId int64) tea.Cmd {
	return func() tea.Msg {
		return deleteModelCmd(m.getAPI(), modelId)
	}
}

func (m *mainModel) toggleModelPublicCmd(versionIDs []int64, public bool) tea.Cmd {
	return func() tea.Msg {
		return toggleModelPublicCmd(m.getAPI(), versionIDs, public)
	}
}
