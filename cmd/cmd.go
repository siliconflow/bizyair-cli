package cmd

import (
	"fmt"
	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/siliconcloud-cli/config"
	"github.com/siliconflow/siliconcloud-cli/meta"
	"github.com/urfave/cli/v2"
)

var globalArgs = config.NewArgument()

func Init() *cli.App {
	// flags
	verboseFlag := cli.BoolFlag{Name: "verbose,vv", Usage: "turn on verbose mode", Destination: &globalArgs.Verbose}
	baseDomainFlag := cli.StringFlag{Name: "base_domain", Usage: "Specify the request domain.", Destination: &globalArgs.BaseDomain, Value: meta.DefaultDomain, Required: false}
	apiKeyFlag := cli.StringFlag{Name: "api_key", Aliases: []string{"k"}, Usage: "Specify the api key.", EnvVars: []string{meta.EnvAPIKey}, Destination: &globalArgs.ApiKey}
	typeFlag := cli.StringFlag{Name: "type", Aliases: []string{"t"}, Usage: fmt.Sprintf("Specify the mode type. (Only works for %s)", meta.ModelTypesStr), Destination: &globalArgs.Type}
	pathFlag := cli.StringFlag{Name: "path", Aliases: []string{"p"}, Usage: "Specify the path to upload.", Destination: &globalArgs.Path}
	nameFlag := cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Specify the name of model.", Destination: &globalArgs.Name}
	formatTreeFlag := cli.BoolFlag{Name: "tree", Usage: "Display in file tree format.", Destination: &globalArgs.FormatTree, Required: false}
	extFlag := cli.StringFlag{Name: "ext", Usage: "Specify the ext name of file.", Destination: &globalArgs.ExtName, Required: false}
	overwriteFlag := cli.BoolFlag{Name: "overwrite", Usage: "Overwrite existent model", Destination: &globalArgs.Overwrite, Value: false, Required: false}
	publicFlag := cli.BoolFlag{Name: "public", Usage: "Only show public model", Destination: &globalArgs.Public, Value: false, Required: false}

	app := cli.NewApp()
	app.Name = meta.Name
	app.Usage = meta.Description
	app.Version = fmt.Sprintf("v%s", meta.Version)
	cli.VersionPrinter = func(cCtx *cli.Context) {
		fmt.Printf("Version: %s\nRevision: %s\nBuild At: %s\n", cCtx.App.Version, meta.Commit, meta.BuildDate)
	}

	// global flags
	app.Flags = []cli.Flag{
		&verboseFlag,
		&baseDomainFlag,
		&apiKeyFlag,
	}

	// Commands
	app.Commands = []*cli.Command{
		{
			Name:  meta.CmdLogin,
			Usage: "Login to the SiliconCloud",
			Flags: []cli.Flag{
				&apiKeyFlag,
			},
			Action: Login,
		},
		{
			Name:   meta.CmdWhoami,
			Usage:  "Find out which user is logged in",
			Flags:  []cli.Flag{},
			Action: Whoami,
		},
		{
			Name:   meta.CmdLogout,
			Usage:  "Log out",
			Flags:  []cli.Flag{},
			Action: Logout,
		},
		{
			Name:  meta.CmdUpload,
			Usage: "Upload a file or a folder to your model directory on SiliconCloud",
			Flags: []cli.Flag{
				&typeFlag,
				&pathFlag,
				&nameFlag,
				&overwriteFlag,
			},
			Action: Upload,
		},
		{
			Name:  meta.CmdModel,
			Usage: "{ls, ls-files, rm} Commands to interact with your models.",
			Subcommands: []*cli.Command{
				{
					Name:  meta.CmdLs,
					Usage: "List your models",
					Flags: []cli.Flag{
						&typeFlag,
						&publicFlag,
					},
					Action: ListModel,
				},
				{
					Name:  meta.CmdLsFiles,
					Usage: "List files in your model",
					Flags: []cli.Flag{
						&typeFlag,
						&nameFlag,
						&extFlag,
						&formatTreeFlag,
						&publicFlag,
					},
					Action: ListFilesModel,
				},
				{
					Name:  meta.CmdRm,
					Usage: "Remove your model",
					Flags: []cli.Flag{
						&typeFlag,
						&nameFlag,
					},
					Action: RemoveModel,
				},
			},
		},
	}

	return app
}

func setLogVerbose(verbose bool) {
	if verbose {
		logs.SetLevel(logs.LevelDebug)
	} else {
		logs.SetLevel(logs.LevelWarn)
	}
}
