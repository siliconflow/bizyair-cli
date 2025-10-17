package cmd

import (
	"fmt"
	"os"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func Logout(c *cli.Context) error {
	args, err := globalArgs.Parse(c, meta.CmdLogout)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

	// 调用统一的登出业务逻辑
	err = actions.ExecuteLogout()
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	fmt.Fprintln(os.Stdout, "Logged out successfully")
	return nil
}
