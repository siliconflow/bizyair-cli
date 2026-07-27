package cmd

import (
	"fmt"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	tuiPkg "github.com/siliconflow/bizyair-cli/cmd/tui"
	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

var globalArgs = config.NewArgument()

func Init() *cli.App {
	globalArgs = config.NewArgument()
	// flags
	languageFlag := cli.StringFlag{Name: "lang", Usage: i18n.T("app.language"), Value: string(i18n.CurrentLanguage())}
	verboseFlag := cli.BoolFlag{Name: "verbose,vv", Usage: i18n.T("cli.flag.verbose"), Destination: &globalArgs.Verbose}
	baseDomainFlag := cli.StringFlag{Name: "base_domain", Usage: i18n.T("cli.flag.base_domain"), Destination: &globalArgs.BaseDomain, Value: meta.DefaultDomain, Required: false}
	apiKeyFlag := cli.StringFlag{Name: "api_key", Aliases: []string{"k"}, Usage: i18n.T("cli.flag.api_key"), EnvVars: []string{meta.EnvAPIKey}, Destination: &globalArgs.ApiKey}
	typeFlag := cli.StringFlag{Name: "type", Aliases: []string{"t"}, Usage: i18n.T("cli.flag.type", map[string]any{"Types": meta.ModelTypesStr}), Destination: &globalArgs.Type}
	pathFlag := cli.StringSliceFlag{Name: "path", Aliases: []string{"p"}, Usage: i18n.T("cli.flag.path"), Destination: &cli.StringSlice{}}
	nameFlag := cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: i18n.T("cli.flag.name"), Destination: &globalArgs.Name}
	// hostFlag := cli.StringFlag{Name: "host", Usage: fmt.Sprintf("Specify the request host, default: %s", meta.DefaultHost), Destination: &globalArgs.Host, Value: meta.DefaultHost}
	// portFlag := cli.StringFlag{Name: "port", Usage: fmt.Sprintf("Specify the request port, default: %s", meta.DefaultPort), Destination: &globalArgs.Port, Value: meta.DefaultPort}
	versionFlag := cli.StringSliceFlag{Name: "version", Aliases: []string{"v", "V"}, Usage: i18n.T("cli.flag.model_version"), Destination: &cli.StringSlice{}}
	versionPublicFlag := cli.StringSliceFlag{Name: "public", Aliases: []string{"pub"}, Usage: i18n.T("cli.flag.public"), Destination: &cli.StringSlice{}}
	introFlag := cli.StringSliceFlag{Name: "intro", Aliases: []string{"i"}, Usage: i18n.T("cli.flag.intro"), Destination: &cli.StringSlice{}}
	introPathFlag := cli.StringSliceFlag{Name: "intro-path", Usage: i18n.T("cli.flag.intro_path"), Destination: &cli.StringSlice{}}
	coverUrlsFlag := cli.StringSliceFlag{Name: "cover", Usage: i18n.T("cli.flag.cover"), Destination: &cli.StringSlice{}}
	baseModelFlag := cli.StringSliceFlag{Name: "base", Aliases: []string{"b"}, Usage: i18n.T("cli.flag.base", map[string]any{"Models": meta.BaseModelStr}), Required: false, Destination: &cli.StringSlice{}}
	fileFlag := cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: i18n.T("cli.flag.file"), Destination: &globalArgs.FilePath}

	configureCLIHelp()

	app := cli.NewApp()
	app.Name = meta.Name
	app.Usage = i18n.T("app.summary")
	app.Version = meta.Version
	app.HideHelpCommand = true
	app.OnUsageError = localizedUsageError
	cli.VersionPrinter = func(cCtx *cli.Context) {
		fmt.Fprintln(cCtx.App.Writer, i18n.T("app.version_format", map[string]any{
			"Version": cCtx.App.Version, "Revision": meta.Commit, "BuildDate": meta.BuildDate,
		}))
	}

	// global flags
	app.Flags = []cli.Flag{
		&languageFlag,
		&verboseFlag,
		&baseDomainFlag,
		&apiKeyFlag,
	}

	// 默认无参进入主 TUI
	app.Action = func(c *cli.Context) error {
		if c.Args().Present() {
			return i18n.NewError("cli.error.unknown_command", map[string]any{"Command": c.Args().First()}, nil)
		}
		return tuiPkg.MainTUI(c)
	}

	// Commands
	app.Commands = []*cli.Command{
		{
			Name:  meta.CmdLogin,
			Usage: i18n.T("cli.command.login"),
			Flags: []cli.Flag{
				&apiKeyFlag,
			},
			Action: Login,
		},
		{
			Name:   meta.CmdLogout,
			Usage:  i18n.T("cli.command.logout"),
			Flags:  []cli.Flag{},
			Action: Logout,
		},
		{
			Name:  meta.CmdUpload,
			Usage: i18n.T("cli.command.upload"),
			Flags: []cli.Flag{
				&fileFlag,
				&typeFlag,
				&pathFlag,
				&nameFlag,
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
			Usage: i18n.T("cli.command.model"),
			Action: func(c *cli.Context) error {
				if c.Args().Present() {
					return i18n.NewError("cli.error.unknown_command", map[string]any{"Command": c.Args().First()}, nil)
				}
				return cli.ShowSubcommandHelp(c)
			},
			Subcommands: []*cli.Command{
				{
					Name:  meta.CmdLs,
					Usage: i18n.T("cli.command.model_list"),
					Flags: []cli.Flag{
						&typeFlag,
					},
					Action: ListModel,
				},
				{
					Name:  meta.CmdRm,
					Usage: i18n.T("cli.command.model_remove"),
					Flags: []cli.Flag{
						&typeFlag,
						&nameFlag,
					},
					Action: RemoveModel,
				},
			},
		},
		{
			Name:  meta.CmdUpgrade,
			Usage: i18n.T("cli.command.upgrade"),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "check",
					Aliases: []string{"c"},
					Usage:   i18n.T("cli.flag.check"),
				},
				&cli.BoolFlag{
					Name:    "force",
					Aliases: []string{"f"},
					Usage:   i18n.T("cli.flag.force"),
				},
			},
			Action: Upgrade,
		},
		localizedHelpCommand(),
	}

	localizeUsageHandlers(app.Commands)
	app.Setup()

	return app
}

func setLogVerbose(verbose bool) {
	if verbose {
		logs.SetLevel(logs.LevelDebug)
	} else {
		logs.SetLevel(logs.LevelWarn)
	}
}

// Verbose reports whether the current invocation enabled verbose diagnostics.
func Verbose() bool {
	return globalArgs != nil && globalArgs.Verbose
}

func logArguments(args *config.Argument) {
	if args == nil {
		return
	}
	logs.Debugf("args: %#v\n", redactedArguments(args))
}

func redactedArguments(args *config.Argument) *config.Argument {
	if args == nil {
		return nil
	}
	safe := *args
	if safe.ApiKey != "" {
		safe.ApiKey = "[REDACTED]"
	}
	return &safe
}
