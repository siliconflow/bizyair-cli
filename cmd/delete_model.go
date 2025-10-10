package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
)

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




