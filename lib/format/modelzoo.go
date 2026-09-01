package format

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
)

// PriceValue 将价格数值（积分，千分之一美元）格式化为美元显示，如 "$1.234"。
// 支持 "数值*倍数" 形式（如 "16*4"），仅转换数值部分。
func PriceValue(val string) string {
	trimmed := strings.TrimSpace(val)
	if v, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return USDPrice(v)
	}
	parts := strings.SplitN(val, "*", 2)
	if len(parts) == 2 {
		if v, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
			return USDPrice(v) + " *" + parts[1]
		}
	}
	return val
}

// USDPrice 将积分值（千分之一美元）格式化为美元金额字符串。
func USDPrice(credits float64) string {
	return fmt.Sprintf("%s%g", i18n.T("common.currency_usd", nil), credits/1000.0)
}

// PriceUnitLabel 将计费单位（SECOND/CALL/IMAGE/MTOKEN 等）翻译为本地化文本。
func PriceUnitLabel(name string) string {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "SECOND", "SECONDS", "S", "PER_SECOND", "每秒", "秒":
		return i18n.T("common.modelzoo.price_unit_per_second", nil)
	case "CALL", "CALLS", "PER_CALL", "每次", "次":
		return i18n.T("common.modelzoo.price_unit_per_call", nil)
	case "IMAGE", "IMAGES", "IMG", "PER_IMAGE", "每张", "张":
		return i18n.T("common.modelzoo.price_unit_per_image", nil)
	case "MTOKEN", "MTOK", "MT", "PER_MTOKEN", "PER_MT", "百万TOKEN":
		return i18n.T("common.modelzoo.price_unit_per_mtoken", nil)
	default:
		return name
	}
}

// PriceKindLabel 将计价组类型名称（Input/Output）翻译为本地化文本。
func PriceKindLabel(name string) string {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "INPUT":
		return i18n.T("common.modelzoo.price_kind_input", nil)
	case "OUTPUT":
		return i18n.T("common.modelzoo.price_kind_output", nil)
	default:
		return name
	}
}

// PriceGroupHeading 组合计价组标题，如 “Input（按秒计费）”。
func PriceGroupHeading(pricingName, unitName string) string {
	kind := PriceKindLabel(pricingName)
	if unit := PriceUnitLabel(unitName); unit != "" {
		if kind != "" {
			return kind + "（" + unit + "）"
		}
		return unit
	}
	if kind == "" {
		return "-"
	}
	return kind
}

// PriceTableColumnTitle 返回价格表参数列的显示标题（优先本地化标签，其次字段名）。
func PriceTableColumnTitle(col lib.PriceTableColumn) string {
	if title := i18n.APITranslate("param_label", col.FieldLabel); title != "" {
		return title
	}
	if col.FieldName != "" {
		return col.FieldName
	}
	return "-"
}

// PriceTableColumns 构建价格表的显示列（含各参数列）。withUnitColumn 为 true 时
// 在参数列后追加“计价单位”列（CLI 样式）；无论哪种布局都会追加价格列，有备注时再追加备注列。
func PriceTableColumns(pt lib.PriceTable, withUnitColumn bool) []string {
	cols := make([]string, 0, len(pt.Columns)+3)
	for _, col := range pt.Columns {
		cols = append(cols, PriceTableColumnTitle(col))
	}
	if withUnitColumn {
		cols = append(cols, i18n.T("common.modelzoo.price_unit_header", nil))
	}
	cols = append(cols, i18n.T("common.modelzoo.price_value_header", nil))
	if len(pt.Remarks) > 0 {
		cols = append(cols, i18n.T("common.modelzoo.price_remark_header", nil))
	}
	return cols
}

// PriceTableRows 构建价格表的显示行（每行 = 参数值 [计价单位] 价格 [备注]）。
// withUnitColumn 需与 PriceTableColumns 保持一致。
func PriceTableRows(pt lib.PriceTable, withUnitColumn bool) [][]string {
	n := len(pt.PricingValues)
	if len(pt.CellsV2) > n {
		n = len(pt.CellsV2)
	}
	if len(pt.Remarks) > n {
		n = len(pt.Remarks)
	}
	rows := make([][]string, 0, n)
	for r := 0; r < n; r++ {
		row := make([]string, 0, len(pt.Columns)+3)
		for c := 0; c < len(pt.Columns); c++ {
			cell := "-"
			if r < len(pt.CellsV2) && c < len(pt.CellsV2[r]) {
				cell = pt.CellsV2[r][c]
			}
			row = append(row, cell)
		}
		if withUnitColumn {
			row = append(row, PriceGroupHeading(pt.PricingName, pt.UnitName))
		}
		val := "-"
		if r < len(pt.PricingValues) {
			val = PriceValue(pt.PricingValues[r])
		}
		row = append(row, val)
		if len(pt.Remarks) > 0 {
			remark := "-"
			if r < len(pt.Remarks) && strings.TrimSpace(pt.Remarks[r]) != "" {
				remark = i18n.APITranslate("remark", pt.Remarks[r])
			}
			row = append(row, remark)
		}
		rows = append(rows, row)
	}
	return rows
}

// PriceTableRecords 返回价格表的行记录数（用于计算渲染高度）。
func PriceTableRecords(pt lib.PriceTable) int {
	n := len(pt.PricingValues)
	if len(pt.CellsV2) > n {
		n = len(pt.CellsV2)
	}
	if len(pt.Remarks) > n {
		n = len(pt.Remarks)
	}
	return n
}