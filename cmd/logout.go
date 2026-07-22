package cmd

import (
	"fmt"
	"os"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func Logout(c *cli.Context) error {
	args := parseArgument(c, meta.CmdLogout)
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

	// 调用统一的登出业务逻辑
	if err := actions.ExecuteLogout(); err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	fmt.Fprintln(os.Stdout, i18n.T("cli.logout.success"))
	return nil
}
