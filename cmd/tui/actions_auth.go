package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
)

func loginCmd(baseDomain, apiKey string) tea.Cmd {
	return func() tea.Msg {
		api := lib.NewClient(baseDomain, apiKey)
		result := actions.ExecuteLoginContext(context.Background(), api, apiKey)
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
		return actionDoneMsg{out: i18n.T("cli.logout.success", nil) + "\n", err: nil}
	}
}
