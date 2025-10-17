package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
)

// 模型详情加载完成消息
type modelDetailLoadedMsg struct {
	detail *lib.BizyModelDetail
	err    error
}

// 加载模型列表
func loadModelList(apiKey string) tea.Cmd {
	return func() tea.Msg {
		// 准备输入参数
		input := actions.ListModelsInput{
			ApiKey:     apiKey,
			BaseDomain: meta.DefaultDomain,
			Current:    1,
			PageSize:   100,
			Sort:       "Recently",
		}

		// 调用统一的业务逻辑
		result := actions.ListModels(input)
		if result.Error != nil {
			return modelListLoadedMsg{err: result.Error}
		}

		return modelListLoadedMsg{
			models: result.Models,
			total:  result.Total,
			err:    nil,
		}
	}
}

// 加载模型详情
func loadModelDetail(apiKey string, bizyModelId int64) tea.Cmd {
	return func() tea.Msg {
		// 调用统一的业务逻辑
		result := actions.GetModelDetail(apiKey, meta.DefaultDomain, bizyModelId)
		if result.Error != nil {
			return modelDetailLoadedMsg{detail: nil, err: result.Error}
		}

		return modelDetailLoadedMsg{detail: result.Detail, err: nil}
	}
}

// 删除模型
type deleteModelDoneMsg struct {
	msg string
	err error
}

func deleteBizyModel(apiKey string, bizyModelId int64) tea.Cmd {
	return func() tea.Msg {
		// 调用统一的业务逻辑
		result := actions.DeleteModel(apiKey, meta.DefaultDomain, bizyModelId)
		if !result.Success {
			return deleteModelDoneMsg{err: result.Error}
		}

		return deleteModelDoneMsg{msg: "删除成功", err: nil}
	}
}
