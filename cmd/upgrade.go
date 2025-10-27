package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

// Upgrade 升级命令
func Upgrade(c *cli.Context) error {
	setLogVerbose(globalArgs.Verbose)

	checkOnly := c.Bool("check")
	force := c.Bool("force")

	fmt.Printf("BizyAir CLI 升级工具\n")
	fmt.Printf("当前版本: %s\n", meta.Version)
	fmt.Printf("==================\n")

	// 创建升级选项
	opts := lib.UpgradeOptions{
		CheckOnly:      checkOnly,
		Force:          force,
		CurrentVersion: meta.Version,
		Context:        context.Background(),
		StatusFunc: func(status string) {
			fmt.Printf("%s\n", status)
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

	fmt.Printf("==================\n")

	if !result.Success {
		fmt.Fprintf(os.Stderr, "❌ %s\n", result.Message)
		if result.Error != nil {
			fmt.Fprintf(os.Stderr, "错误详情: %v\n", result.Error)
		}
		return cli.Exit("升级失败", meta.LoadError)
	}

	fmt.Printf("%s\n", result.Message)

	if result.NeedUpgrade && !checkOnly {
		fmt.Printf("\n提示: 请重新运行命令以使用新版本\n")
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
