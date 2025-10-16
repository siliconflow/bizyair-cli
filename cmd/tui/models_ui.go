package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/lib"
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
				m.detailContent = ""
				return nil
			}
			m.step = mainStepMenu
			m.act = actionInputs{}
			m.loadingModelList = false
			m.modelList = nil
			m.viewingModelDetail = false
			m.modelDetail = nil
			m.detailContent = ""
			return nil
		case "r":
			if !m.viewingModelDetail {
				m.loadingModelList = true
				return loadModelList(m.apiKey)
			}
			return nil
		case "up", "k", "down", "j":
			// 在详情视图中处理滚动
			if m.viewingModelDetail && !m.loadingModelDetail {
				var cmd tea.Cmd
				m.detailViewport, cmd = m.detailViewport.Update(msg)
				return cmd
			}
			// 在列表视图中，让这些按键继续传递到 modelTable
			// 不要在这里 return，让代码继续执行到底部的 modelTable.Update
		case "pgup", "pgdown", "home", "end":
			// 这些按键只在详情视图中使用
			if m.viewingModelDetail && !m.loadingModelDetail {
				var cmd tea.Cmd
				m.detailViewport, cmd = m.detailViewport.Update(msg)
				return cmd
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

		// 使用 viewport 展示详情内容
		var b strings.Builder
		b.WriteString(m.titleStyle.Render("模型详情"))
		b.WriteString("\n\n")
		b.WriteString(m.detailViewport.View())
		b.WriteString("\n\n")

		// 滚动提示信息
		scrollInfo := fmt.Sprintf("%.f%%", m.detailViewport.ScrollPercent()*100)
		hints := fmt.Sprintf("返回：Esc/q，删除：Ctrl+D，滚动：↑↓/PgUp/PgDn/Home/End (%s)", scrollInfo)
		b.WriteString(m.hintStyle.Render(hints))
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

// buildModelDetailMarkdown 生成模型详情的 Markdown 内容
func buildModelDetailMarkdown(detail *lib.BizyModelDetail) string {
	if detail == nil {
		return "# 错误\n\n未加载到模型数据"
	}

	var b strings.Builder

	// 标题和基本信息（普通换行显示）
	b.WriteString(fmt.Sprintf("# %s\n\n", detail.Name))
	b.WriteString(fmt.Sprintf("**模型 ID**: #%d\n\n", detail.Id))
	b.WriteString(fmt.Sprintf("**类型**: %s\n\n", dash(detail.Type)))
	b.WriteString(fmt.Sprintf("**作者**: %s\n\n", dash(detail.UserName)))
	if detail.Source != "" {
		b.WriteString(fmt.Sprintf("**来源**: %s\n\n", detail.Source))
	}
	b.WriteString(fmt.Sprintf("**创建时间**: %s\n\n", dash(detail.CreatedAt)))
	b.WriteString(fmt.Sprintf("**更新时间**: %s\n\n", dash(detail.UpdatedAt)))
	b.WriteString("\n")

	// 统计信息
	if detail.Counter.UsedCount > 0 || detail.Counter.LikedCount > 0 || detail.Counter.DownloadedCount > 0 {
		b.WriteString("## 📊 统计信息\n\n")
		if detail.Counter.UsedCount > 0 {
			b.WriteString(fmt.Sprintf("- 使用次数: %d\n", detail.Counter.UsedCount))
		}
		if detail.Counter.LikedCount > 0 {
			b.WriteString(fmt.Sprintf("- 点赞数: %d\n", detail.Counter.LikedCount))
		}
		if detail.Counter.DownloadedCount > 0 {
			b.WriteString(fmt.Sprintf("- 下载次数: %d\n", detail.Counter.DownloadedCount))
		}
		if detail.Counter.ForkedCount > 0 {
			b.WriteString(fmt.Sprintf("- Fork 次数: %d\n", detail.Counter.ForkedCount))
		}
		if detail.Counter.ViewCount > 0 {
			b.WriteString(fmt.Sprintf("- 浏览次数: %d\n", detail.Counter.ViewCount))
		}
		b.WriteString("\n")
	}

	// 版本信息
	b.WriteString("## 📦 版本列表\n\n")
	if len(detail.Versions) == 0 {
		b.WriteString("*暂无版本*\n\n")
	} else {
		for i, v := range detail.Versions {
			b.WriteString(fmt.Sprintf("### 版本 %d: %s\n\n", i+1, dash(v.Version)))

			// 使用表格展示版本基本信息
			b.WriteString("| 属性 | 值 |\n")
			b.WriteString("|------|----|\n")

			if v.BaseModel != "" {
				b.WriteString(fmt.Sprintf("| 基础模型 | %s |\n", v.BaseModel))
			}
			if v.FileName != "" {
				if v.FileSize > 0 {
					b.WriteString(fmt.Sprintf("| 文件名 | %s |\n", v.FileName))
					b.WriteString(fmt.Sprintf("| 文件大小 | %s |\n", format.FormatBytes(v.FileSize)))
				} else {
					b.WriteString(fmt.Sprintf("| 文件名 | %s |\n", v.FileName))
				}
			}
			if v.Sign != "" {
				b.WriteString(fmt.Sprintf("| 文件签名 | `%s` |\n", v.Sign))
			}
			if v.Path != "" {
				b.WriteString(fmt.Sprintf("| 路径 | %s |\n", v.Path))
			}
			if v.ModelId > 0 {
				b.WriteString(fmt.Sprintf("| 模型 ID | %d |\n", v.ModelId))
			}
			if v.Available {
				b.WriteString("| 状态 | ✓ 可用 |\n")
			} else {
				b.WriteString("| 状态 | ✗ 不可用 |\n")
			}
			b.WriteString(fmt.Sprintf("| 创建时间 | %s |\n", dash(v.CreatedAt)))
			b.WriteString(fmt.Sprintf("| 更新时间 | %s |\n", dash(v.UpdatedAt)))

			// 版本统计信息
			if v.Counter.UsedCount > 0 || v.Counter.LikedCount > 0 || v.Counter.DownloadedCount > 0 {
				if v.Counter.UsedCount > 0 {
					b.WriteString(fmt.Sprintf("| 使用次数 | %d |\n", v.Counter.UsedCount))
				}
				if v.Counter.LikedCount > 0 {
					b.WriteString(fmt.Sprintf("| 点赞数 | %d |\n", v.Counter.LikedCount))
				}
				if v.Counter.DownloadedCount > 0 {
					b.WriteString(fmt.Sprintf("| 下载次数 | %d |\n", v.Counter.DownloadedCount))
				}
				if v.Counter.ForkedCount > 0 {
					b.WriteString(fmt.Sprintf("| Fork 次数 | %d |\n", v.Counter.ForkedCount))
				}
			}

			b.WriteString("\n")

			// 封面图片（如果有）
			if len(v.CoverUrls) > 0 {
				b.WriteString("**封面图片**:\n\n")
				for idx, url := range v.CoverUrls {
					b.WriteString(fmt.Sprintf("%d. %s\n", idx+1, url))
				}
				b.WriteString("\n")
			}

			// 版本介绍（可能是 Markdown 格式）
			if v.Intro != "" {
				b.WriteString("#### 📝 介绍\n\n")
				// 直接输出介绍内容，glamour 会处理 Markdown 格式
				intro := strings.TrimSpace(v.Intro)
				b.WriteString(intro)
				b.WriteString("\n\n")
			}

			// 版本分隔线
			if i < len(detail.Versions)-1 {
				b.WriteString("---\n\n")
			}
		}
	}

	return b.String()
}
