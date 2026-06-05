package cmd

import (
	"fmt"
	"os"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func Login(c *cli.Context) error {
	args := parseArgument(c, meta.CmdLogin)
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

	if args.ApiKey == "" {
		return cli.Exit(fmt.Errorf("api key is required, you can specify \"--api_key\" or environment variable \"%s\" to set", meta.EnvAPIKey), meta.LoadError)
	}

	client := lib.NewClient(meta.DefaultDomain, args.ApiKey)
	result := actions.ExecuteLogin(client, args.ApiKey)
	if !result.Success {
		return cli.Exit(result.Error, meta.LoadError)
	}

	fmt.Fprintln(os.Stdout, "Login successfully")
	return nil
}
