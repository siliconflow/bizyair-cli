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

func Whoami(c *cli.Context) error {
	args, err := globalArgs.Parse(c, meta.CmdWhoami)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

	// 获取API Key
	var apiKey string
	if args.ApiKey != "" {
		apiKey = args.ApiKey
	} else {
		apiKey, err = lib.NewSfFolder().GetKey()
		if err != nil {
			return cli.Exit(err, meta.LoadError)
		}
	}

	// 调用统一的whoami业务逻辑
	result := actions.ExecuteWhoami(apiKey)
	if result.Error != nil {
		return cli.Exit(result.Error, meta.LoadError)
	}

	// 格式化输出
	if result.Name != "" {
		fmt.Fprintf(os.Stdout, "Your account name: %s\n", result.Name)
	} else {
		fmt.Fprintf(os.Stdout, "Your account email: %s\n", result.Email)
	}

	return nil
}
