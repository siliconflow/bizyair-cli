package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
)

func loginCmd(apiKey string) tea.Cmd {
	return func() tea.Msg {
		api := lib.NewClient(meta.AuthDomain, apiKey)
		result := actions.ExecuteLogin(api, apiKey)
		if !result.Success {
			return loginDoneMsg{ok: false, err: result.Error}
		}
		return loginDoneMsg{ok: true}
	}
}

// 运行 logout
func runLogout() tea.Cmd {
	return func() tea.Msg {
		// 调用统一的登出业务逻辑
		err := actions.ExecuteLogout()
		if err != nil {
			return actionDoneMsg{out: "", err: err}
		}
		return actionDoneMsg{out: "Logged out successfully\n", err: nil}
	}
}
