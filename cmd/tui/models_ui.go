package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/lib/format"
)

// 模型列表/详情输入更新
func (m *mainModel) updateListModelsInputs(msg tea.Msg) tea.Cmd {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc", "q":
			if m.confirmingDelete {
				m.confirmingDelete = false
				m.deleteTargetId = 0
				m.deleteTargetName = ""
				return nil
			}
			if m.viewingModelDetail {
				m.viewingModelDetail = false
				m.modelDetail = nil
				return nil
			}
			m.step = mainStepMenu
			m.act = actionInputs{}
			m.loadingModelList = false
			m.modelList = nil
			m.viewingModelDetail = false
			m.modelDetail = nil
			return nil
		case "r":
			if !m.viewingModelDetail {
				m.loadingModelList = true
				return loadModelList(m.apiKey)
			}
			return nil
		case "ctrl+d":
			// 详情界面删除
			if m.viewingModelDetail {
				if m.modelDetail != nil {
					m.confirmingDelete = true
					m.deleteTargetId = m.modelDetail.Id
					m.deleteTargetName = m.modelDetail.Name
				}
				return nil
			}
			// 列表界面删除
			if !m.loadingModelList && len(m.modelList) > 0 {
				selectedRow := m.modelTable.SelectedRow()
				if len(selectedRow) > 0 {
					idStr := selectedRow[0]
					nameStr := ""
					if len(selectedRow) > 1 {
						nameStr = selectedRow[1]
					}
					if idStr != "" {
						if id64, err := strconv.ParseInt(idStr, 10, 64); err == nil {
							m.confirmingDelete = true
							m.deleteTargetId = id64
							m.deleteTargetName = nameStr
							return nil
						}
					}
				}
			}
		case "enter":
			if m.confirmingDelete && m.deleteTargetId > 0 {
				m.running = true
				return deleteBizyModel(m.apiKey, m.deleteTargetId)
			}
			if !m.loadingModelList && len(m.modelList) > 0 && !m.viewingModelDetail && !m.confirmingDelete {
				selectedRow := m.modelTable.SelectedRow()
				if len(selectedRow) > 0 {
					idStr := selectedRow[0]
					if idStr != "" {
						if id64, err := strconv.ParseInt(idStr, 10, 64); err == nil {
							var targetId int64 = id64
							for _, mInfo := range m.modelList {
								if mInfo.Id == targetId {
									m.loadingModelDetail = true
									m.viewingModelDetail = true
									return loadModelDetail(m.apiKey, targetId)
								}
							}
						}
					}
				}
			}
		}
	}
	if m.viewingModelDetail || m.confirmingDelete {
		return nil
	}
	var cmd tea.Cmd
	m.modelTable, cmd = m.modelTable.Update(msg)
	return cmd
}

// 渲染模型列表/详情视图
func (m *mainModel) renderListModelsView() string {
	if m.confirmingDelete {
		var b strings.Builder
		title := "删除确认"
		if m.deleteTargetName != "" {
			title = title + " · " + m.deleteTargetName
		}
		b.WriteString(m.titleStyle.Render(title))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("确定删除模型 #%d 吗？此操作不可恢复。\n\n", m.deleteTargetId))
		b.WriteString(m.hintStyle.Render("确认：Enter，取消：Esc"))
		return b.String()
	}
	if m.viewingModelDetail {
		if m.loadingModelDetail {
			return m.titleStyle.Render("模型详情") + "\n\n" + m.sp.View() + " 加载中...\n\n" + m.hintStyle.Render("返回：Esc/q")
		}
		if m.modelDetail == nil {
			return m.titleStyle.Render("模型详情") + "\n\n" + "未加载到数据" + "\n\n" + m.hintStyle.Render("返回：Esc/q")
		}
		var b strings.Builder
		b.WriteString(m.titleStyle.Render(fmt.Sprintf("%s (#%d) · %s", m.modelDetail.Name, m.modelDetail.Id, m.modelDetail.Type)))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("作者：%s\n创建：%s  更新：%s\n", dash(m.modelDetail.UserName), dash(m.modelDetail.CreatedAt), dash(m.modelDetail.UpdatedAt)))
		b.WriteString("\n")
		if len(m.modelDetail.Versions) == 0 {
			b.WriteString("暂无版本\n")
		} else {
			for i, v := range m.modelDetail.Versions {
				b.WriteString(m.hintStyle.Render(fmt.Sprintf("[%d] 版本 %s", i+1, dash(v.Version))))
				b.WriteString("\n")
				b.WriteString(fmt.Sprintf("  基础模型: %s\n", dash(v.BaseModel)))
				if v.FileSize > 0 {
					b.WriteString(fmt.Sprintf("  文件: %s (%s)\n", dash(v.FileName), format.FormatBytes(v.FileSize)))
				} else {
					b.WriteString(fmt.Sprintf("  文件: %s\n", dash(v.FileName)))
				}
				if v.Sign != "" {
					b.WriteString(fmt.Sprintf("  文件签名: %s\n", v.Sign))
				}
				if v.Intro != "" {
					b.WriteString(fmt.Sprintf("  介绍: %s\n", v.Intro))
				}
				if v.ModelId > 0 {
					b.WriteString(fmt.Sprintf("  模型ID: %d\n", v.ModelId))
				}
				b.WriteString(fmt.Sprintf("  创建: %s  更新: %s\n", dash(v.CreatedAt), dash(v.UpdatedAt)))
				b.WriteString("\n")
			}
		}
		b.WriteString(m.hintStyle.Render("返回：Esc/q，删除：Ctrl+D，列表：Enter 选择前需回退到列表"))
		return b.String()
	}
	if m.loadingModelList {
		return m.titleStyle.Render("列出模型") + "\n\n" + m.sp.View() + " 加载中...\n\n" + m.hintStyle.Render("请稍候")
	}
	if len(m.modelList) == 0 {
		return m.titleStyle.Render("列出模型") + "\n\n" + "暂无模型数据\n\n" + m.hintStyle.Render("返回：Esc/q，刷新：r")
	}
	if _, ih := m.innerSize(); ih > 0 {
		h := ih - 12
		if h < 5 {
			h = 5
		}
		m.modelTable.SetHeight(h)
	}
	if w, _ := m.innerSize(); w > 0 {
		m.resizeModelTable(w)
	}
	return m.titleStyle.Render(fmt.Sprintf("列出模型（共 %d 个）", m.modelListTotal)) + "\n\n" + m.modelTable.View() + "\n\n" + m.hintStyle.Render("导航：↑↓，进入详情：Enter，删除：Ctrl+D，返回：Esc/q，刷新：r")
}
