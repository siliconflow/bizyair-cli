package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/format"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

// Upgrade 升级命令
func Upgrade(c *cli.Context) error {
	setLogVerbose(globalArgs.Verbose)

	if errors.Is(c.Context.Err(), context.Canceled) {
		return canceledUpgradeExitError()
	}

	checkOnly := c.Bool("check")
	force := c.Bool("force")

	fmt.Println(i18n.T("cli.upgrade.title"))
	fmt.Println(i18n.T("cli.upgrade.current_version", map[string]any{"Version": meta.Version}))
	fmt.Printf("==================\n")

	// 创建升级选项
	opts := lib.UpgradeOptions{
		CheckOnly:      checkOnly,
		Force:          force,
		CurrentVersion: meta.Version,
		Context:        c.Context,
		ManifestURL:    meta.ManifestURL,
		StatusFunc: func(status string) {
			fmt.Printf("%s\n", status)
		},
		ProgressFunc: func(downloaded, total int64) {
			percentage := float64(downloaded) / float64(total) * 100
			fmt.Printf("\r%s", i18n.T("cli.upgrade.download_progress", map[string]any{
				"Percentage": fmt.Sprintf("%.1f", percentage),
				"Downloaded": format.FormatBytes(downloaded),
				"Total":      format.FormatBytes(total),
			}))
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
		if errors.Is(result.Error, context.Canceled) || errors.Is(c.Context.Err(), context.Canceled) {
			return canceledUpgradeExitError()
		}
		fmt.Fprintf(os.Stderr, "❌ %s\n", result.Message)
		if result.Error != nil {
			fmt.Fprintln(os.Stderr, i18n.T("cli.upgrade.error_detail", map[string]any{"Cause": result.Error}))
		}
		return cli.Exit(i18n.T("cli.upgrade.failed"), meta.LoadError)
	}

	fmt.Printf("%s\n", result.Message)

	if result.NeedUpgrade && !checkOnly {
		fmt.Printf("\n%s\n", i18n.T("cli.upgrade.restart_hint"))
	}

	return nil
}

func canceledUpgradeExitError() error {
	return cli.Exit(
		i18n.NewError("cli.upgrade.canceled", nil, context.Canceled),
		meta.InterruptedExitCode,
	)
}
