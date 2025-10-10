package tui

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/table"
)

func (m *mainModel) resizeModelTable(totalWidth int) {
	if totalWidth <= 0 {
		return
	}
	usable := totalWidth - 8
	if usable < 60 {
		usable = 60
	}

	idW := 10
	typeW := 12
	verW := 8
	baseMin := 15

	remaining := usable - (idW + typeW + verW)
	if remaining < 20+15+20 {
		remaining = 20 + 15 + 20
	}
	nameW := int(float64(remaining) * 0.28)
	baseW := int(float64(remaining) * 0.18)
	fileW := remaining - nameW - baseW
	if baseW < baseMin {
		deficit := baseMin - baseW
		baseW = baseMin
		if fileW > deficit+10 {
			fileW -= deficit
		} else if nameW > deficit+10 {
			nameW -= deficit
		}
	}
	m.nameColWidth = maxInt(5, nameW-1)
	m.fileColWidth = maxInt(8, fileW-1)

	cols := []table.Column{
		{Title: "ID", Width: idW},
		{Title: "名称", Width: nameW},
		{Title: "类型", Width: typeW},
		{Title: "版本数", Width: verW},
		{Title: "基础模型", Width: baseW},
		{Title: "文件名", Width: fileW},
	}
	m.modelTable.SetColumns(cols)
	if len(m.modelList) > 0 {
		m.rebuildModelTableRows()
	}
}

func (m *mainModel) rebuildModelTableRows() {
	rows := []table.Row{}
	for _, model := range m.modelList {
		versionCount := fmt.Sprintf("%d", len(model.Versions))
		baseModels := []string{}
		fileNames := []string{}
		for _, version := range model.Versions {
			if version.BaseModel != "" {
				baseModels = append(baseModels, version.BaseModel)
			}
			if version.FileName != "" {
				fileNames = append(fileNames, version.FileName)
			}
		}
		baseModelStr := "-"
		if len(baseModels) > 0 {
			uniqueBaseModels := uniqueStrings(baseModels)
			if len(uniqueBaseModels) == 1 {
				baseModelStr = uniqueBaseModels[0]
			} else {
				baseModelStr = fmt.Sprintf("%s (共%d个)", uniqueBaseModels[0], len(uniqueBaseModels))
			}
		}
		fileNameStr := "-"
		if len(fileNames) > 0 {
			if len(fileNames) == 1 {
				fileNameStr = fileNames[0]
			} else {
				fileNameStr = fmt.Sprintf("%s (共%d个)", fileNames[0], len(fileNames))
			}
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", model.Id),
			truncateString(model.Name, maxInt(5, m.nameColWidth)),
			model.Type,
			versionCount,
			baseModelStr,
			truncateString(fileNameStr, maxInt(8, m.fileColWidth)),
		})
	}
	m.modelTable.SetRows(rows)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func uniqueStrings(stringsArr []string) []string {
	set := map[string]struct{}{}
	out := make([]string, 0, len(stringsArr))
	for _, s := range stringsArr {
		if _, ok := set[s]; !ok {
			set[s] = struct{}{}
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
