package lib

import (
	"os/exec"
	"runtime"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/meta"
)

// MyModelsURL 是"我的模型"页面的 URL
const MyModelsURL = meta.DefaultBaseDomain + "/community?path=my"

// OpenBrowser 尝试在系统默认浏览器中打开指定的 URL
// 返回 (成功消息, 错误)
func OpenBrowser(url string) (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "linux":
		// 尝试多个常见的命令
		for _, browser := range []string{"xdg-open", "x-www-browser", "gnome-open", "kde-open"} {
			if _, err := exec.LookPath(browser); err == nil {
				cmd = exec.Command(browser, url)
				break
			}
		}
		if cmd == nil {
			return "", i18n.NewError("error.browser.command_missing", nil, nil)
		}
	default:
		return "", i18n.NewError("error.browser.os_unsupported", map[string]any{"OS": runtime.GOOS}, nil)
	}

	err := cmd.Start()
	if err != nil {
		return "", i18n.NewError("error.browser.start_failed", nil, err)
	}

	return i18n.T("browser.opened"), nil
}
