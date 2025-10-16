package cmd

import (
	"fmt"
	"os"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func RemoveModel(c *cli.Context) error {
	args, err := globalArgs.Parse(c, meta.CmdRm)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

	if err := lib.ValidateModelType(args.Type); err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	var apiKey string
	if args.ApiKey != "" {
		apiKey = args.ApiKey
	} else {
		apiKey, err = lib.NewSfFolder().GetKey()
		if err != nil {
			return err
		}
	}

	client := lib.NewClient(args.BaseDomain, apiKey)

	_, err = client.RemoveModel(args.Type, args.Name)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "Model removed successfully.")

	return nil
}
