package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
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

// formatCreditsPrice 将价格数值格式化为美元显示：积分值（千分之一美元）换算。
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

// priceKindLabel 将计价组类型名称（Input/Output）翻译为本地化文本。
func priceKindLabel(name string) string {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "INPUT":
		return i18n.T("tui.modelzoo.price_kind_input", nil)
	case "OUTPUT":
		return i18n.T("tui.modelzoo.price_kind_output", nil)
	default:
		return name
	}
}

// tuiPriceUnitLabel 将计费单位（SECOND/CALL/IMAGE/MTOKEN 等）翻译为本地化文本。
func tuiPriceUnitLabel(name string) string {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "SECOND", "SECONDS", "S", "PER_SECOND", "每秒", "秒":
		return i18n.T("tui.modelzoo.price_unit_per_second", nil)
	case "CALL", "CALLS", "PER_CALL", "每次", "次":
		return i18n.T("tui.modelzoo.price_unit_per_call", nil)
	case "IMAGE", "IMAGES", "IMG", "PER_IMAGE", "每张", "张":
		return i18n.T("tui.modelzoo.price_unit_per_image", nil)
	case "MTOKEN", "MTOK", "MT", "PER_MTOKEN", "PER_MT", "百万TOKEN":
		return i18n.T("tui.modelzoo.price_unit_per_mtoken", nil)
	default:
		return name
	}
}

// renderPriceTableWithHeader 渲染带表头的价格键值表：首行为列头
// （参数组合 │ 价格），其下为分隔线与数据行，与 renderKeyValueTable 对齐。
func renderPriceTableWithHeader(labels, values []string, maxW int) string {
	if len(labels) == 0 || len(labels) != len(values) {
		return ""
	}
	labelColW, sepW, valueColW, labelContentW, valueContentW := keyValueColLayout(maxW, labels)
	sep := " │ "

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FBBF24"))
	leftHeader := headerStyle.Padding(0, 1).Width(labelColW).
		Render(truncateLine(i18n.T("tui.modelzoo.price_param_header", nil), labelContentW))
	rightHeader := headerStyle.Padding(0, 1).Width(valueColW).
		Render(truncateLine(i18n.T("tui.modelzoo.price_value_header", nil), valueContentW))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	sepCell := sepStyle.Render(sep)
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, leftHeader, sepCell, rightHeader)

	lineW := labelColW + sepW + valueColW
	lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	line := lineStyle.Render(strings.Repeat("─", lineW))

	body := renderKeyValueTable(labels, values, maxW)
	return lipgloss.JoinVertical(lipgloss.Left, headerRow, line, body)
}

// buildPriceTableContent 参照用户信息模块的键值表格展示方式渲染价格数据：
// 每个计价组（PricingName + UnitName）为一个灰色圆角框区块，标题为黄色，
// 区块内包含表头行与数据行；多组之间以空行分隔。
func (m *mainModel) buildPriceTableContent(prices []lib.PriceTable) {
	if len(prices) == 0 {
		m.priceTableContent = ""
		return
	}
	var groups []string
	for _, pt := range prices {
		var labels []string
		var values []string
		maxRow := len(pt.PricingValues)
		if len(pt.CellsV2) > maxRow {
			maxRow = len(pt.CellsV2)
		}
		if len(pt.Remarks) > maxRow {
			maxRow = len(pt.Remarks)
		}
		if maxRow == 0 {
			maxRow = 1
		}
		for r := 0; r < maxRow; r++ {
			var params string
			if r < len(pt.CellsV2) {
				params = strings.Join(pt.CellsV2[r], " / ")
			}
			if params == "" {
				params = "-"
			}
			price := "-"
			if r < len(pt.PricingValues) {
				price = formatCreditsPrice(pt.PricingValues[r])
			}
			labels = append(labels, params)
			values = append(values, price)
			if r < len(pt.Remarks) && strings.TrimSpace(pt.Remarks[r]) != "" {
				labels = append(labels, i18n.T("tui.modelzoo.price_remark_header", nil))
				values = append(values, pt.Remarks[r])
			}
		}
		title := priceKindLabel(pt.PricingName)
		if unit := tuiPriceUnitLabel(pt.UnitName); unit != "" {
			if title != "" {
				title += "（" + unit + "）"
			} else {
				title = unit
			}
		}
		if title == "" {
			title = "-"
		}
		table := renderPriceTableWithHeader(labels, values, m.width-24)
		groups = append(groups, sectionCard(title, table, m.width-19))
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
