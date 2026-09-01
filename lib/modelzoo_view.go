package lib

import (
	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

// ModelzooTableColumns 返回模型广场列表的 5 列表头，CLI 与 TUI 共用。
func ModelzooTableColumns() []string {
	return []string{
		i18n.T("common.modelzoo.col_name", nil),
		i18n.T("common.modelzoo.col_endpoint", nil),
		i18n.T("common.modelzoo.col_manufacturer", nil),
		i18n.T("common.modelzoo.col_category", nil),
		i18n.T("common.modelzoo.col_version", nil),
	}
}

// ModelzooTableRow 返回模型的一行展示单元格，空值显示为 "-"。
func ModelzooTableRow(m ModelzooModelFlat) []string {
	return []string{
		DisplayOrDash(m.DisplayName),
		DisplayOrDash(m.Endpoint),
		DisplayOrDash(i18n.APITranslate("manufacturer", m.Manufacturer)),
		DisplayOrDash(m.Category),
		DisplayOrDash(i18n.APITranslate("version", m.ModelVersion)),
	}
}

// DisplayOrDash 空字符串显示为 "-"。
func DisplayOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}