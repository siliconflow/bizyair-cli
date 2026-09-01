package lib

import (
	"strings"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

// ModelzooFilter 模型广场列表筛选条件集合，CLI 与 TUI 共用。
type ModelzooFilter struct {
	Capability   string
	Manufacturer string
	Series       string
	Version      string
}

// Empty 判断筛选条件是否为空。
func (f ModelzooFilter) Empty() bool {
	return f.Capability == "" && f.Manufacturer == "" && f.Series == "" && f.Version == ""
}

// ModelzooSeries 返回模型的系列名：优先取价格表字段，其次从 endpoint 推断。
func ModelzooSeries(m ModelzooModelFlat) string {
	if m.Series != "" {
		return m.Series
	}
	return SeriesFromEndpoint(m.Endpoint)
}

// FilterModelzooModels 按筛选条件过滤模型列表；条件为空时原样返回。
func FilterModelzooModels(models []ModelzooModelFlat, f ModelzooFilter) []ModelzooModelFlat {
	if f.Empty() {
		return models
	}
	keep := make([]ModelzooModelFlat, 0, len(models))
	for _, m := range models {
		if ModelzooModelMatches(m, f) {
			keep = append(keep, m)
		}
	}
	return keep
}

// ModelzooModelMatches 判断单个模型是否满足筛选条件。
// 匹配时对原始 API 值与当前语言下的展示值均做大小写不敏感比较，
// 例如厂商原始值 "117" 展示为 "ByteDance"，两者都可命中。
func ModelzooModelMatches(m ModelzooModelFlat, f ModelzooFilter) bool {
	if f.Capability != "" && !modelzooHasCapability(m, f.Capability) {
		return false
	}
	if f.Manufacturer != "" && !modelzooFilterValueMatch(m.Manufacturer, "manufacturer", f.Manufacturer) {
		return false
	}
	if f.Series != "" {
		s := ModelzooSeries(m)
		if s != "" && !modelzooFilterValueMatch(s, "series", f.Series) {
			return false
		}
	}
	if f.Version != "" && !modelzooFilterValueMatch(m.ModelVersion, "version", f.Version) {
		return false
	}
	return true
}

// modelzooHasCapability 判断模型是否具备指定能力（命中标签或类别）。
func modelzooHasCapability(m ModelzooModelFlat, capability string) bool {
	for _, t := range m.Tags {
		if modelzooFilterValueMatch(t, "tag", capability) {
			return true
		}
	}
	return modelzooFilterValueMatch(m.Category, "category", capability)
}

func modelzooFilterValueMatch(raw, domain, want string) bool {
	if raw == "" || want == "" {
		return false
	}
	if strings.EqualFold(raw, want) {
		return true
	}
	return strings.EqualFold(i18n.APITranslate(domain, raw), want)
}