package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
)

func loginCmd(ctx context.Context, baseDomain, apiKey string) tea.Cmd {
	if ctx == nil {
		ctx = context.Background()
	}
	return func() tea.Msg {
		api := lib.NewClient(baseDomain, apiKey)
		result := actions.ExecuteLoginContext(ctx, api, apiKey)
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
