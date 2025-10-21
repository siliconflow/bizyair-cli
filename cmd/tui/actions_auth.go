package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/lib/actions"
)

// 登录校验 + 保存
func loginCmd(apiKey string) tea.Cmd {
	return func() tea.Msg {
		// 调用统一的登录业务逻辑
		result := actions.ExecuteLogin(apiKey)
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
