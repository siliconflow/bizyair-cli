package tui

import (
	"strings"

	tablev2 "charm.land/bubbles/v2/table"
	lipglossv2 "charm.land/lipgloss/v2"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/format"
)

// modelzooPriceTablesFor 从内存里的模型列表里查找指定端点的缓存价格表。
func modelzooPriceTablesFor(models []lib.ModelzooModelFlat, endpoint string) []lib.PriceTable {
	for i := range models {
		if models[i].Endpoint == endpoint {
			return models[i].PriceTables
		}
	}
	return nil
}

// openPriceView 打开价格视图：优先使用内存缓存的 price_table_flat 价格表，
// 缓存为空时按端点拉取每端点价格表（与 CLI modelzoo price 一致）。
func (m *mainModel) openPriceView(endpoint string) tea.Cmd {
	if pts := modelzooPriceTablesFor(m.modelzoo.models, endpoint); len(pts) > 0 {
		m.priceTables = pts
		m.buildPriceTableContent(pts)
		m.step = mainStepPriceView
		return nil
	}
	m.running = true
	return fetchPriceTable(m.getAPI(), endpoint)
}

// priceTableWidth 价格表在简介区块内的可用宽度。
func (m *mainModel) priceTableWidth() int {
	w := m.width - 33
	if w < 20 {
		w = 20
	}
	return w
}

// renderPriceTableContent 将单个价格表渲染为多列表格：每个参数一列，
// 最后一列为价格（有备注时追加备注列）。列数据与单元格构造复用 lib/format。
// 列宽按内容自适应并填满可用宽度。
func (m *mainModel) renderPriceTableContent(pt lib.PriceTable) string {
	titles := format.PriceTableColumns(pt, false)
	rowData := format.PriceTableRows(pt, false)
	rows := make([]tablev2.Row, 0, len(rowData))
	for _, r := range rowData {
		rows = append(rows, r)
	}

	ideal := make([]int, len(titles))
	for i, t := range titles {
		ideal[i] = lipglossv2.Width(t)
	}
	for _, row := range rows {
		for i, c := range row {
			if w := lipglossv2.Width(c); w > ideal[i] {
				ideal[i] = w
			}
		}
	}

	// 每个单元格有左右各 1 格内边距，列内容总宽需再扣除 2*nCols，
	// 否则整行会超出视口宽度并导致末列被截断、各列错位。
	nCols := len(titles)
	if nCols == 0 {
		return ""
	}
	contentW := m.priceTableWidth() - nCols*2
	if contentW < nCols*4 {
		contentW = nCols * 4
	}
	cols := fitColumnWidthsByContent(ideal, contentW, true)
	for i, t := range titles {
		cols[i].Title = t
	}

	tableW := contentW + nCols*2
	height := len(rows) + 3
	if height < 4 {
		height = 4
	}
	tbl := tablev2.New(
		tablev2.WithColumns(cols),
		tablev2.WithRows(rows),
		tablev2.WithWidth(tableW),
		tablev2.WithHeight(height),
		tablev2.WithFocused(false),
	)
	applyPriceTableStyles(&tbl)
	return renderTable(tbl)
}

// buildPriceTableContent 渲染价格数据：
// 每个计价组（PricingName + UnitName）为一个灰色圆角框区块，标题为黄色，
// 区块内包含多列表格（每个参数一列）；多组之间以空行分隔。
func (m *mainModel) buildPriceTableContent(prices []lib.PriceTable) {
	if len(prices) == 0 {
		m.priceTableContent = ""
		return
	}
	var groups []string
	for _, pt := range prices {
		groups = append(groups, sectionCard(format.PriceGroupHeading(pt.PricingName, pt.UnitName), m.renderPriceTableContent(pt), m.width-19))
	}
	m.priceTableContent = strings.Join(groups, "\n\n")
}

// updatePriceView 处理价格视图按键（键值表为静态文本，仅保留返回按键）。
func updatePriceView(m *mainModel, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.step = mainStepModelzoo
			return nil
		case "enter":
			m.step = mainStepModelzoo
			return nil
		}
	}
	return nil
}

// renderPriceView 渲染价格视图。
func (m *mainModel) renderPriceView() string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#60A5FA")).
		Bold(true).
		Render(i18n.T("tui.modelzoo.price_title", nil))

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")

	if len(m.priceTables) == 0 {
		b.WriteString(i18n.T("tui.modelzoo.no_price", nil))
	} else if m.priceTableContent != "" {
		b.WriteString(m.priceTableContent)
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).
		Render(i18n.T("tui.hint.return_menu", nil)))
	return b.String()
}
