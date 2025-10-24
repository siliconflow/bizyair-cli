package cmd

import (
	"context"
	"fmt"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

// Upgrade 升级命令
func Upgrade(c *cli.Context) error {
	setLogVerbose(globalArgs.Verbose)

	checkOnly := c.Bool("check")
	force := c.Bool("force")

	logs.Infof("BizyAir CLI 升级工具\n")
	logs.Infof("当前版本: %s\n", meta.Version)
	logs.Infof("==================\n")

	// 创建升级选项
	opts := lib.UpgradeOptions{
		CheckOnly:      checkOnly,
		Force:          force,
		CurrentVersion: meta.Version,
		Context:        context.Background(),
		StatusFunc: func(status string) {
			logs.Infof("%s\n", status)
		},
		ProgressFunc: func(downloaded, total int64) {
			percentage := float64(downloaded) / float64(total) * 100
			fmt.Printf("\r下载进度: %.1f%% (%s / %s)",
				percentage,
				formatBytes(downloaded),
				formatBytes(total))
		},
	}

	// 执行升级
	result := lib.PerformUpgrade(opts)

	// 清除进度行
	if !checkOnly {
		fmt.Println()
	}

	logs.Infof("==================\n")

	if !result.Success {
		logs.Errorf("❌ %s\n", result.Message)
		if result.Error != nil {
			logs.Errorf("错误详情: %v\n", result.Error)
		}
		return cli.Exit("升级失败", meta.LoadError)
	}

	logs.Infof("%s\n", result.Message)

	if result.NeedUpgrade && !checkOnly {
		logs.Infof("\n提示: 请重新运行命令以使用新版本\n")
	}

	return nil
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
