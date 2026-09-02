package lib

import (
	"strconv"
	"time"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

// AIApplicationBaseModel 返回应用的基座模型（来自第一个版本的 base_model）。
func AIApplicationBaseModel(app *BizyModelInfo) string {
	if app == nil {
		return ""
	}
	if len(app.Versions) > 0 && app.Versions[0] != nil {
		return app.Versions[0].BaseModel
	}
	return ""
}

// AIApplicationCreatedAt 返回应用发布时间：优先取列表项的 created_at，
// 为空时回退到第一个版本的 created_at。
func AIApplicationCreatedAt(app *BizyModelInfo) string {
	if app == nil {
		return ""
	}
	if app.CreatedAt != "" {
		return app.CreatedAt
	}
	if len(app.Versions) > 0 && app.Versions[0] != nil {
		return app.Versions[0].CreatedAt
	}
	return ""
}

// AIApplicationTableColumns 返回 AI 应用列表表头，CLI 与 TUI 共用。
func AIApplicationTableColumns() []string {
	return []string{
		i18n.T("common.aiapp.col_name", nil),
		i18n.T("common.aiapp.col_base_model", nil),
		i18n.T("common.aiapp.col_created", nil),
		i18n.T("common.aiapp.col_used", nil),
		i18n.T("common.aiapp.col_liked", nil),
		i18n.T("common.aiapp.col_forked", nil),
	}
}

// AIApplicationTableRow 返回应用的一行展示单元格，空值显示为 "-"。
func AIApplicationTableRow(app *BizyModelInfo) []string {
	if app == nil {
		return []string{"-", "-", "-", "-", "-", "-"}
	}
	return []string{
		DisplayOrDash(app.Name),
		DisplayOrDash(AIApplicationBaseModel(app)),
		formatCreatedAt(AIApplicationCreatedAt(app)),
		DisplayOrDash(countText(app.Counter.UsedCount)),
		DisplayOrDash(countText(app.Counter.LikedCount)),
		DisplayOrDash(countText(app.Counter.ForkedCount)),
	}
}

// formatCreatedAt 将时间串格式化为日期（仅到日）；支持带时区与不带时区的常见格式，
// 空值或解析失败时返回 "-"。
func formatCreatedAt(ts string) string {
	if ts == "" {
		return "-"
	}
	for _, layout := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
	} {
		if t, err := time.Parse(layout, ts); err == nil {
			return t.Local().Format("2006-01-02")
		}
	}
	for _, layout := range []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if t, err := time.ParseInLocation(layout, ts, time.Local); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return "-"
}

func countText(v int) string {
	return strconv.Itoa(v)
}
