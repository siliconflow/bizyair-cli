package lib

import (
	"sort"
	"strings"
)

// AI 应用列表排序方式常量。
const (
	AIAppSortRecent     = "Recently"
	AIAppSortMostUsed   = "MostUsed"
	AIAppSortMostLiked  = "MostLiked"
	AIAppSortMostForked = "MostForked"
)

// AIApplicationFilter AI 应用列表筛选条件，CLI 与 TUI 共用。
type AIApplicationFilter struct {
	Sort      string
	BaseModel string
	Keyword   string
}

// FilterAIApplications 按筛选条件过滤应用列表；条件为空时原样返回。
func FilterAIApplications(apps []*BizyModelInfo, f AIApplicationFilter) []*BizyModelInfo {
	result := apps
	if f.BaseModel != "" {
		keep := make([]*BizyModelInfo, 0, len(result))
		for _, app := range result {
			if AIApplicationBaseModel(app) == f.BaseModel {
				keep = append(keep, app)
			}
		}
		result = keep
	}
	if f.Keyword != "" {
		keep := make([]*BizyModelInfo, 0, len(result))
		for _, app := range result {
			if ContainsApplicationKeyword(app, f.Keyword) {
				keep = append(keep, app)
			}
		}
		result = keep
	}
	if f.Sort != "" {
		SortAIApplications(result, f.Sort)
	}
	return result
}

// ContainsApplicationKeyword 判断应用是否命中关键字（名称/描述/基座模型，大小写不敏感）。
func ContainsApplicationKeyword(app *BizyModelInfo, q string) bool {
	if app == nil {
		return false
	}
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return true
	}
	fields := []string{app.Name, app.Description, AIApplicationBaseModel(app)}
	for _, s := range fields {
		if strings.Contains(strings.ToLower(s), q) {
			return true
		}
	}
	return false
}

// SortAIApplications 按指定排序方式对应用列表排序。
func SortAIApplications(apps []*BizyModelInfo, sortBy string) {
	sort.SliceStable(apps, func(i, j int) bool {
		a, b := apps[i], apps[j]
		switch sortBy {
		case AIAppSortMostUsed:
			return a.Counter.UsedCount > b.Counter.UsedCount
		case AIAppSortMostLiked:
			return a.Counter.LikedCount > b.Counter.LikedCount
		case AIAppSortMostForked:
			return a.Counter.ForkedCount > b.Counter.ForkedCount
		default:
			return AIApplicationCreatedAt(a) > AIApplicationCreatedAt(b)
		}
	})
}

// AIApplicationBaseModels 返回列表中的去重基座模型集合（按名称排序）。
func AIApplicationBaseModels(apps []*BizyModelInfo) []string {
	set := make(map[string]bool)
	for _, app := range apps {
		if bm := AIApplicationBaseModel(app); bm != "" {
			set[bm] = true
		}
	}
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// AIAppSortOptions 返回可选的排序方式列表。
func AIAppSortOptions() []string {
	return []string{AIAppSortRecent, AIAppSortMostUsed, AIAppSortMostLiked, AIAppSortMostForked}
}
