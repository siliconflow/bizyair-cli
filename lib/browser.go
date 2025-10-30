package lib

import (
	"fmt"
	"os/exec"
	"runtime"
)

// MyModelsURL 是"我的模型"页面的 URL
const MyModelsURL = "https://bizyair.cn/community?path=my"

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
			return "", fmt.Errorf("未找到可用的浏览器命令")
		}
	default:
		return "", fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf("无法启动浏览器: %v", err)
	}

	return "已在浏览器中打开", nil
}
