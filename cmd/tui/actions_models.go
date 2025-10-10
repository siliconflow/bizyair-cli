package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
)

// 模型详情加载完成消息
type modelDetailLoadedMsg struct {
	detail *lib.BizyModelDetail
	err    error
}

// 加载模型列表（直接调用API）
func loadModelList(apiKey string) tea.Cmd {
	return func() tea.Msg {
		client := lib.NewClient(meta.DefaultDomain, apiKey)

		var modelTypes []string
		for _, t := range meta.ModelTypes {
			modelTypes = append(modelTypes, string(t))
		}

		resp, err := client.ListModel(1, 100, "", "Recently", modelTypes, []string{})
		if err != nil {
			return modelListLoadedMsg{err: err}
		}
		return modelListLoadedMsg{models: resp.Data.List, total: resp.Data.Total, err: nil}
	}
}

// 加载模型详情
func loadModelDetail(apiKey string, bizyModelId int64) tea.Cmd {
	return func() tea.Msg {
		client := lib.NewClient(meta.DefaultDomain, apiKey)
		resp, err := client.GetBizyModelDetail(bizyModelId)
		if err != nil {
			return modelDetailLoadedMsg{detail: nil, err: err}
		}
		if resp == nil || resp.Data.Id == 0 {
			return modelDetailLoadedMsg{detail: nil, err: fmt.Errorf("未获取到模型详情")}
		}
		return modelDetailLoadedMsg{detail: &resp.Data, err: nil}
	}
}

// 删除模型
type deleteModelDoneMsg struct {
	msg string
	err error
}

func deleteBizyModel(apiKey string, bizyModelId int64) tea.Cmd {
	return func() tea.Msg {
		client := lib.NewClient(meta.DefaultDomain, apiKey)
		_, err := client.DeleteBizyModelById(bizyModelId)
		if err != nil {
			return deleteModelDoneMsg{err: err}
		}
		return deleteModelDoneMsg{msg: "删除成功", err: nil}
	}
}
