package cmd

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

// 加载模型详情的命令
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




