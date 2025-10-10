package tui

import (
	"bytes"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
)

// 登录校验 + 保存
func loginCmd(apiKey string) tea.Cmd {
	return func() tea.Msg {
		client := lib.NewClient(meta.AuthDomain, apiKey)
		if _, err := client.UserInfo(); err != nil {
			return loginDoneMsg{ok: false, err: withStep("登录校验", err)}
		}
		if err := lib.NewSfFolder().SaveKey(apiKey); err != nil {
			return loginDoneMsg{ok: false, err: withStep("保存凭据", err)}
		}
		return loginDoneMsg{ok: true}
	}
}

// 运行 whoami（复用现有子命令）
func runWhoami(apiKey string) tea.Cmd {
	return func() tea.Msg {
		exe, _ := os.Executable()
		args := []string{}
		if apiKey != "" {
			args = append(args, "--api_key", apiKey)
		}
		args = append(args, "whoami")
		cmd := exec.Command(exe, args...)
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		err := cmd.Run()
		return actionDoneMsg{out: buf.String(), err: withStep("whoami 执行", err)}
	}
}

// 运行 logout
func runLogout() tea.Cmd {
	return func() tea.Msg {
		err := lib.NewSfFolder().RemoveKey()
		if err != nil {
			return actionDoneMsg{out: "", err: withStep("登出", err)}
		}
		return actionDoneMsg{out: "Logged out successfully\n", err: nil}
	}
}
