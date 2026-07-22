package cmd

import (
	"fmt"
	"os"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func ListModel(c *cli.Context) error {
	args := parseArgument(c, meta.CmdLs)
	setLogVerbose(args.Verbose)
	logArguments(args)

	endpoints, resolveErr := lib.ResolveServiceEndpoints(args.BaseDomain)
	if resolveErr != nil {
		return cli.Exit(resolveErr, meta.LoadError)
	}
	myModelsURL := endpoints.MyModelsURL()

	// 打开浏览器查看"我的模型"
	msg, err := lib.OpenBrowser(myModelsURL)
	if err != nil {
		// 如果无法打开浏览器，提示用户手动访问
		fmt.Fprintln(os.Stderr, i18n.T("cli.model.browser_failed", map[string]any{"Cause": err}))
		fmt.Fprintln(os.Stdout, i18n.T("cli.model.visit_manually", map[string]any{"URL": myModelsURL}))
		return cli.Exit(err, meta.LoadError)
	}

	// 成功打开浏览器
	fmt.Fprintln(os.Stdout, msg)
	fmt.Fprintln(os.Stdout, i18n.T("cli.model.visit", map[string]any{"URL": myModelsURL}))
	return nil
}
