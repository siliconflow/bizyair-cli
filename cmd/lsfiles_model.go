package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func ListFilesModel(c *cli.Context) error {
	args, err := globalArgs.Parse(c, meta.CmdLsFiles)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

	if err := lib.ValidateModelType(args.Type); err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	if args.Name != "" {
		if err := lib.ValidateModelName(args.Name); err != nil {
			return cli.Exit(err, meta.LoadError)
		}
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

	modelFilesResp, err := client.ListModelFiles(args.Type, args.Name, args.ExtName, args.Public)
	if err != nil {
		return err
	}

	modelFiles := modelFilesResp.Data.Files

	if len(modelFiles) < 1 {
		fmt.Fprintln(os.Stdout, "No files found.")
		return nil
	}

	if args.FormatTree {
		root := lib.NewNode("")
		for _, mf := range modelFiles {
			root.AddPath(mf.LabelPath)
		}
		root.PrintTree("")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "PATH\tAVAILABLE\t")
		// Print data rows
		for _, mr := range modelFiles {
			fmt.Fprintf(w, "%s\t%s\t\n", mr.LabelPath, func() string {
				if mr.Available {
					return "Yes"
				}
				return "No"
			}())
		}
		w.Flush()
	}

	return nil
}
