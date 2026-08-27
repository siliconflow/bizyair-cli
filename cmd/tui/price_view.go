package tui

import (
	"fmt"
	"strconv"
	"strings"

	tablev2 "charm.land/bubbles/v2/table"
	lipglossv2 "charm.land/lipgloss/v2"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
)

// modelzooPriceTableFor 从内存中的模型列表里查找指定端点的缓存价格表。
func modelzooPriceTableFor(models []lib.ModelzooModelFlat, endpoint string) *lib.PriceTable {
	for i := range models {
		if models[i].Endpoint == endpoint {
			return models[i].PriceTable
		}
	}
	return nil
}

// formatCreditsPrice 将价格单元格的值格式化为美元显示。
func formatCreditsPrice(val string) string {
	trimmed := strings.TrimSpace(val)
	if v, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return usdAmount(v)
	}
	parts := strings.SplitN(val, "*", 2)
	if len(parts) == 2 {
		if v, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
			return usdAmount(v) + " *" + parts[1]
		}
	}
	return val
}

// usdAmount 将积分值（千分之一美元）格式化为美元金额字符串。
func usdAmount(credits float64) string {
	return fmt.Sprintf("%s%g", i18n.T("common.currency_usd", nil), credits/1000.0)
}

// priceColumnUnit 返回第 col 列中首个带计费单位的单元格的单位名。
func priceColumnUnit(cells [][]lib.PriceTableCell, col int) string {
	for _, row := range cells {
		if col < len(row) && row[col].UnitName != "" {
			return row[col].UnitName
		}
	}
	return ""
}

// priceColumnTitle 根据价格列单元格的计费单位返回表头文案。
func priceColumnTitle(unit string) string {
	switch unit {
	case "SECOND":
		return i18n.T("tui.modelzoo.price_unit_per_second", nil)
	case "CALL", "call":
		return i18n.T("tui.modelzoo.price_unit_per_call", nil)
	case "IMAGE":
		return i18n.T("tui.modelzoo.price_unit_per_image", nil)
	case "MToken":
		return i18n.T("tui.modelzoo.price_unit_per_mtoken", nil)
	default:
		return i18n.T("tui.modelzoo.price_short", nil)
	}
}

// buildPriceTableView 根据价格表数据构建表格视图。
func (m *mainModel) buildPriceTableView(pt *lib.PriceTable) {
	m.priceTable = pt
	if pt == nil || len(pt.Columns) == 0 {
		m.priceTableView = tablev2.New()
		return
	}

	innerW, _ := m.innerSize()
	maxW := innerW - 8
	if maxW < 20 {
		maxW = 20
	}

	titles := make([]string, len(pt.Columns))
	ideal := make([]int, len(pt.Columns))
	for i, c := range pt.Columns {
		titles[i] = c.FieldLabel
		if unit := priceColumnUnit(pt.Cells, i); unit != "" {
			titles[i] = priceColumnTitle(unit)
		}
		ideal[i] = lipglossv2.Width(titles[i])
	}

	overhead := len(titles)*2 + 2
	if maxW > overhead {
		maxW -= overhead
	}

	rows := make([]tablev2.Row, 0, len(pt.Cells))
	for _, row := range pt.Cells {
		cells := make([]string, len(row))
		for j, cell := range row {
			val := cell.ValueStr
			if val == "" {
				val = fmt.Sprintf("%.4f", cell.Amount)
			}
			if cell.UnitName != "" {
				val = formatCreditsPrice(val)
			}
			cells[j] = val
			if w := lipglossv2.Width(val); w > ideal[j] {
				ideal[j] = w
			}
		}
		rows = append(rows, cells)
	}

	cols := fitColumnWidthsByContent(ideal, maxW, true)
	for i, t := range titles {
		cols[i].Title = t
	}

	h := m.height - 18
	if h < 4 {
		h = 4
	}

	t := tablev2.New(
		tablev2.WithColumns(cols),
		tablev2.WithRows(rows),
		tablev2.WithWidth(innerW-8),
		tablev2.WithHeight(h),
		tablev2.WithFocused(true),
	)
	applyTableStyles(&t)
	m.priceTableView = t
}

// updatePriceView 将消息转发给价格表格组件。
func updatePriceView(m *mainModel, msg tea.Msg) tea.Cmd {
	// Handle navigation keys for the price table
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.priceTableView.MoveUp(1)
		case "down", "j":
			m.priceTableView.MoveDown(1)
		case "pgup":
			m.priceTableView.MoveUp(m.priceTableView.Height())
		case "pgdown":
			m.priceTableView.MoveDown(m.priceTableView.Height())
		case "home":
			m.priceTableView.GotoTop()
		case "end":
			m.priceTableView.GotoBottom()
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

// renderPriceView 渲染价格表视图。
func (m *mainModel) renderPriceView() string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#60A5FA")).
		Bold(true).
		Render(i18n.T("tui.modelzoo.price_title", nil))

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")

	if m.priceTable == nil || len(m.priceTable.Cells) == 0 {
		b.WriteString(i18n.T("tui.modelzoo.no_price", nil))
	} else {
		b.WriteString(renderTable(m.priceTableView))
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).
		Render(i18n.T("tui.hint.return_menu", nil)))
	return b.String()
}
