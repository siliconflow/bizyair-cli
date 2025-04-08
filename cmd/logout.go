package cmd

import (
	"fmt"
	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
	"os"
)

func Logout(c *cli.Context) error {
	args, err := globalArgs.Parse(c, meta.CmdLogout)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

	err = lib.NewSfFolder().RemoveKey()
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "Logged out successfully")
	return nil
}
