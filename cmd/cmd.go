package cmd

import (
	"fmt"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	tuiPkg "github.com/siliconflow/bizyair-cli/cmd/tui"
	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

var globalArgs = config.NewArgument()

func Init() *cli.App {
	// flags
	verboseFlag := cli.BoolFlag{Name: "verbose,vv", Usage: "turn on verbose mode", Destination: &globalArgs.Verbose}
	baseDomainFlag := cli.StringFlag{Name: "base_domain", Usage: "Specify the request domain.", Destination: &globalArgs.BaseDomain, Value: meta.DefaultDomain, Required: false}
	apiKeyFlag := cli.StringFlag{Name: "api_key", Aliases: []string{"k"}, Usage: "Specify the api key.", EnvVars: []string{meta.EnvAPIKey}, Destination: &globalArgs.ApiKey}
	typeFlag := cli.StringFlag{Name: "type", Aliases: []string{"t"}, Usage: fmt.Sprintf("Specify the mode type. (Only works for %s)", meta.ModelTypesStr), Destination: &globalArgs.Type}
	pathFlag := cli.StringSliceFlag{Name: "path", Aliases: []string{"p"}, Usage: "Specify the path to upload.", Destination: &cli.StringSlice{}}
	nameFlag := cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Specify the name of model.", Destination: &globalArgs.Name}
	overwriteFlag := cli.BoolFlag{Name: "overwrite", Usage: "Overwrite existent model", Destination: &globalArgs.Overwrite, Value: false, Required: false}
	// hostFlag := cli.StringFlag{Name: "host", Usage: fmt.Sprintf("Specify the request host, default: %s", meta.DefaultHost), Destination: &globalArgs.Host, Value: meta.DefaultHost}
	// portFlag := cli.StringFlag{Name: "port", Usage: fmt.Sprintf("Specify the request port, default: %s", meta.DefaultPort), Destination: &globalArgs.Port, Value: meta.DefaultPort}
	versionFlag := cli.StringSliceFlag{Name: "version", Aliases: []string{"v", "V"}, Usage: "Specify the version of model.", Destination: &cli.StringSlice{}}
	versionPublicFlag := cli.StringSliceFlag{Name: "public", Aliases: []string{"pub"}, Usage: "Set corresponding model version public (true/false). Can be specified multiple times for multiple versions.", Destination: &cli.StringSlice{}}
	introFlag := cli.StringSliceFlag{Name: "intro", Aliases: []string{"i"}, Usage: "An introduction to the model version.", Destination: &cli.StringSlice{}}
	introPathFlag := cli.StringSliceFlag{Name: "intro-path", Usage: "Path to .txt or .md file containing the introduction (auto-truncated to 5000 chars). Can be specified multiple times for multiple versions.", Destination: &cli.StringSlice{}}
	coverUrlsFlag := cli.StringSliceFlag{Name: "cover", Usage: "Urls of model covers, use ';' as separator.", Destination: &cli.StringSlice{}}
	baseModelFlag := cli.StringSliceFlag{Name: "base", Aliases: []string{"b"}, Usage: fmt.Sprintf("Specify the base model of uploaded model. (Only works for %s)", meta.BaseModelStr), Required: false, Destination: &cli.StringSlice{}}
	fileFlag := cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "从 YAML 配置文件批量上传", Destination: &globalArgs.FilePath}

	app := cli.NewApp()
	app.Name = meta.Name
	app.Usage = meta.Description
	app.Version = meta.Version
	cli.VersionPrinter = func(cCtx *cli.Context) {
		fmt.Printf("Version: %s\nRevision: %s\nBuild At: %s\n", cCtx.App.Version, meta.Commit, meta.BuildDate)
	}

	// global flags
	app.Flags = []cli.Flag{
		&verboseFlag,
		&baseDomainFlag,
		&apiKeyFlag,
	}

	// 默认无参进入主 TUI
	app.Action = tuiPkg.MainTUI

	// Commands
	app.Commands = []*cli.Command{
		{
			Name:  meta.CmdLogin,
			Usage: "登录到 BizyAir",
			Flags: []cli.Flag{
				&apiKeyFlag,
			},
			Action: Login,
		},
		{
			Name:   meta.CmdLogout,
			Usage:  "退出登录",
			Flags:  []cli.Flag{},
			Action: Logout,
		},
		{
			Name:  meta.CmdUpload,
			Usage: "上传文件或文件夹到 BizyAir 模型目录",
			Flags: []cli.Flag{
				&fileFlag,
				&typeFlag,
				&pathFlag,
				&nameFlag,
				&overwriteFlag,
				&versionFlag,
				&versionPublicFlag,
				&introFlag,
				&introPathFlag,
				&baseModelFlag,
				&coverUrlsFlag,
				// &hostFlag,
				// &portFlag,
			},
			Action: Upload,
		},
		{
			Name:  meta.CmdModel,
			Usage: "{ls, rm} 与模型交互的命令集",
			Subcommands: []*cli.Command{
				{
					Name:  meta.CmdLs,
					Usage: "列出你的模型",
					Flags: []cli.Flag{
						&typeFlag,
					},
					Action: ListModel,
				},
				{
					Name:  meta.CmdRm,
					Usage: "删除你的模型",
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
