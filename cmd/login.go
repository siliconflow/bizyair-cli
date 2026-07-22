package cmd

import (
	"fmt"
	"os"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
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
		return cli.Exit(i18n.NewError("cli.login.api_key_required", map[string]any{"Env": meta.EnvAPIKey}, nil), meta.LoadError)
	}

	client := lib.NewClient(meta.DefaultDomain, args.ApiKey)
	result := actions.ExecuteLogin(client, args.ApiKey)
	if !result.Success {
		return cli.Exit(result.Error, meta.LoadError)
	}

	fmt.Fprintln(os.Stdout, i18n.T("cli.login.success"))
	return nil
}
